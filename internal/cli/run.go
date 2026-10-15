package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/receipts/receipts/internal/collectors"
	"github.com/receipts/receipts/internal/collectors/shell"
	"github.com/receipts/receipts/internal/config"
	"github.com/receipts/receipts/internal/gitstate"
	"github.com/receipts/receipts/internal/integrity"
	"github.com/receipts/receipts/internal/model"
	"github.com/receipts/receipts/internal/render"
	"github.com/receipts/receipts/internal/store"
	"github.com/receipts/receipts/internal/verdict"
)

const runUse = "receipts run [--workdir PATH] [--agent auto|opencode|claude] [--no-shims] [--quiet] [--] <agent command> [args...]"

func printRunHelp(w io.Writer) {
	fmt.Fprintln(w, "run - Wrap an agent command with command capture")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintf(w, "  %s\n", runUse)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --agent auto|opencode|claude   Transcript adapter (default auto)")
	fmt.Fprintln(w, "  --workdir PATH                Project directory (default .)")
	fmt.Fprintln(w, "  --no-shims                    Skip PATH shims (tier 0 with capture warning)")
	fmt.Fprintln(w, "  --quiet                       Store the receipt without printing a summary")
	fmt.Fprintln(w, "  --help                        Show help for this command")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "The wrapped command inherits stdio and the exit code is preserved.")
	fmt.Fprintln(w, "Shims observe PATH lookups only; absolute paths bypass capture and")
	fmt.Fprintln(w, "are always disclosed on the receipt.")
}

// runID returns a unique run directory name.
func runID() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return "run-" + time.Now().UTC().Format("20060102T150405") + "-" + hex.EncodeToString(b[:])
}

// runRun implements receipts run. It always returns the wrapped command's
// exit code unless Receipts itself fails before launch.
func runRun(args []string, stdout, stderr io.Writer) int {
	workdir, agentName := ".", "auto"
	agentSet := false
	noShims, quiet := false, false
	var cmd []string
	sep := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		if sep {
			cmd = append(cmd, a)
			continue
		}
		switch {
		case a == "--":
			sep = true
		case a == "--help" || a == "-h":
			printRunHelp(stdout)
			return 0
		case a == "--no-shims":
			noShims = true
		case a == "--quiet":
			quiet = true
		case a == "--workdir" && i+1 < len(args):
			i++
			workdir = args[i]
		case strings.HasPrefix(a, "--workdir="):
			workdir = strings.TrimPrefix(a, "--workdir=")
		case a == "--agent" && i+1 < len(args):
			i++
			agentName = args[i]
			agentSet = true
		case strings.HasPrefix(a, "--agent="):
			agentName = strings.TrimPrefix(a, "--agent=")
			agentSet = true
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(stderr, "unknown flag: %s\n", a)
			return 2
		default:
			cmd = append(cmd, a)
		}
	}
	if agentName != "auto" && agentName != "opencode" && agentName != "claude" {
		fmt.Fprintf(stderr, "unknown agent: %s\n", agentName)
		return 2
	}
	if len(cmd) == 0 {
		fmt.Fprintln(stderr, "run: missing agent command (see receipts run --help)")
		return 2
	}
	abs, err := filepath.Abs(workdir)
	if err != nil {
		fmt.Fprintf(stderr, "run: %v\n", err)
		return 3
	}
	if _, err := os.Stat(abs); err != nil {
		fmt.Fprintf(stderr, "run: %v\n", err)
		return 3
	}
	cfg, err := loadPolicy(abs)
	if err != nil {
		fmt.Fprintf(stderr, "run: %v\n", err)
		return 2
	}
	if !agentSet {
		agentName = cfg.Agent
	}
	start := time.Now()
	ctx := context.Background()

	var pre *gitstate.Snapshot
	if snap, err := gitstate.Collect(ctx, abs); err != nil {
		if !gitstate.IsNotGit(err) {
			fmt.Fprintf(stderr, "run: pre-run snapshot failed: %v\n", err)
			return 3
		}
	} else {
		pre = &snap
	}

	runDir := filepath.Join(abs, ".receipts", "runs", runID())
	binDir := filepath.Join(runDir, "bin")
	tier := model.Tier0
	shimsActive := false
	origPath := os.Getenv("PATH")
	if !noShims {
		if err := installShims(binDir); err != nil {
			fmt.Fprintf(stderr, "run: shim setup degraded (%v); continuing tier 0\n", err)
		} else {
			shimsActive = true
			tier = model.Tier1
		}
	}

	child := exec.Command(cmd[0], cmd[1:]...)
	child.Dir = abs
	child.Stdin, child.Stdout, child.Stderr = os.Stdin, os.Stdout, os.Stderr
	child.Env = append(os.Environ(),
		shell.EnvRunDir+"="+runDir,
		shell.EnvRealPath+"="+origPath,
		shell.EnvShimDir+"="+binDir,
	)
	if shimsActive {
		child.Env = prependPath(child.Env, binDir+":"+origPath)
	}
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	interrupted := false
	if err := child.Start(); err != nil {
		fmt.Fprintf(stderr, "run: failed to launch: %v\n", err)
		return 3
	}
	waitDone := make(chan error, 1)
	go func() { waitDone <- child.Wait() }()
	var waitErr error
	select {
	case sig := <-sigCh:
		interrupted = true
		_ = child.Process.Signal(sig)
		waitErr = <-waitDone
	case waitErr = <-waitDone:
	}
	childCode := exitCodeOfWait(waitErr)

	rec, saveErr := assembleRunReceipt(ctx, abs, tier, shimsActive, agentName, runDir, pre, start, interrupted, cfg)
	if saveErr != nil {
		fmt.Fprintf(stderr, "run: receipt failed: %v\n", saveErr)
		return 4
	}
	if !quiet {
		if data, rerr := (render.TerminalRenderer{}).Render(rec, render.Options{}); rerr == nil {
			stdout.Write(data)
		}
	}
	rel, _ := filepath.Rel(abs, filepath.Join(abs, ".receipts", "receipts", rec.ID+".json"))
	fmt.Fprintf(stderr, "Saved: %s\n", rel)
	return childCode
}

func prependPath(env []string, path string) []string {
	out := make([]string, 0, len(env)+1)
	found := false
	for _, kv := range env {
		if strings.HasPrefix(kv, "PATH=") && !found {
			out = append(out, "PATH="+path)
			found = true
			continue
		}
		out = append(out, kv)
	}
	if !found {
		out = append(out, "PATH="+path)
	}
	return out
}

func exitCodeOfWait(err error) int {
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		if code := ee.ProcessState.ExitCode(); code >= 0 {
			return code
		}
		return 130
	}
	return 1
}

// installShims links the current executable under each supported name.
func installShims(binDir string) error {
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	for _, name := range shell.Supported {
		if err := os.Symlink(exe, filepath.Join(binDir, name)); err != nil {
			return err
		}
	}
	return nil
}

func assembleRunReceipt(ctx context.Context, abs string, tier model.Tier, shimsActive bool, agentName, runDir string, pre *gitstate.Snapshot, start time.Time, interrupted bool, cfg config.Config) (model.Receipt, error) {
	var rec model.Receipt
	post, perr := gitstate.Collect(ctx, abs)
	nonGit := gitstate.IsNotGit(perr)
	if perr != nil && !nonGit {
		return rec, perr
	}
	rec = model.Receipt{
		SchemaVersion: model.SchemaVersion,
		CreatedAt:     start.UTC(),
		Tier:          tier,
		Repo:          model.Repo{Root: abs, BaselineConfidence: model.BaselineUnknown},
		Claims:        []model.Claim{},
		Flags:         []model.Flag{},
		Evidence:      []model.Evidence{},
		Warnings:      []string{},
		Summary:       model.Summary{},
		Chain:         model.Chain{Algorithm: model.ChainAlgorithmSHA256},
	}
	nextEID := func() string { return fmt.Sprintf("E-%03d", len(rec.Evidence)+1) }

	if !nonGit {
		var base gitstate.Baseline
		if pre != nil {
			base = gitstate.ResolveBaseline(post, pre, "")
		} else {
			base = gitstate.ResolveBaseline(post, nil, "")
		}
		rec.Repo = gitstate.AsRepo(post.Root, post, base)
		rec.Warnings = append(rec.Warnings, base.Warnings...)
		changes, cerr := gitstate.Changed(ctx, post.Root)
		if cerr != nil {
			return rec, cerr
		}
		changes = collectors.DropOwnedDir(changes)
		changedPaths := make([]string, 0, len(changes))
		for _, c := range changes {
			changedPaths = append(changedPaths, c.Path)
		}
		files, added, warnings := collectors.ChangedFiles(ctx, post.Root, post.Head, changes)
		rec.Warnings = append(rec.Warnings, warnings...)
		rec.Flags = integrity.Run(integrity.Input{Files: files, Added: added,
			Scope: policyScope(cfg), Checks: policyChecks(cfg), BaselineConfidence: base.Confidence})
		rec.Warnings = append(rec.Warnings, policyNotes(cfg)...)
		rec.Evidence = append(rec.Evidence, model.Evidence{ID: nextEID(), Kind: model.EvidenceGitSnapshot, Timestamp: time.Now().UTC(), Data: map[string]any{
			"root": post.Root, "head_before": base.HeadBefore, "head_after": base.HeadAfter,
			"baseline_confidence": string(base.Confidence),
			"dirty_before":        string(base.DirtyBefore), "dirty_after": base.DirtyAfter,
		}})
		diffData := map[string]any{"changed_paths": changedPaths, "diff_hash": gitstate.DiffHash(changes)}
		if depAdded, manifests := collectors.DepDeltas(files); len(manifests) > 0 {
			diffData["manifests_changed"] = manifests
			if len(depAdded) > 0 {
				diffData["added_dependencies"] = depAdded
			}
		}
		rec.Evidence = append(rec.Evidence, model.Evidence{ID: nextEID(), Kind: model.EvidenceDiff, Timestamp: time.Now().UTC(), Data: diffData})
		for i := range rec.Flags {
			rec.Flags[i].EvidenceRefs = append(rec.Flags[i].EvidenceRefs, "E-002")
		}
	} else {
		rec.Repo = gitstate.NonGitRepo(abs)
		rec.Warnings = append(rec.Warnings, gitstate.WarnNonGitProject)
	}

	if interrupted {
		rec.Warnings = append(rec.Warnings, "interrupted: receipt covers a partial run")
	}
	if !shimsActive {
		rec.Warnings = append(rec.Warnings, "command capture may be incomplete: wrapper shims were not active for this run")
		rec.Flags = append(rec.Flags, model.Flag{Code: "COMMAND_CAPTURE_INCOMPLETE", Severity: model.SeverityInfo,
			Title: "Command capture incomplete", Details: "shims observe PATH lookups only; absolute paths and custom runtimes bypass capture", EvidenceRefs: []string{}})
	} else {
		rec.Warnings = append(rec.Warnings, "command capture may be incomplete: shims observe PATH lookups only; absolute paths bypass capture")
	}

	events, rerr := shell.ReadEvents(runDir)
	if rerr != nil {
		rec.Warnings = append(rec.Warnings, "command events unreadable: "+rerr.Error())
	} else {
		for _, e := range events {
			cmdID := nextEID()
			evs := collectors.EvidenceForEvent(e, cmdID, "")
			rec.Evidence = append(rec.Evidence, evs[0])
			if len(evs) > 1 {
				evs[1].ID = nextEID()
				rec.Evidence = append(rec.Evidence, evs[1])
			}
		}
	}

	agentDisplay, sessionID := "none", ""
	if a, ref, attempts, ok := selectAdapter(ctx, abs, agentName); ok {
		agentDisplay, sessionID = a.Name(), ref.ID
		msgs, merr := a.Messages(ctx, ref)
		if merr != nil {
			rec.Warnings = append(rec.Warnings, "adapter issue: "+a.Name()+": "+merr.Error())
		} else {
			rec.Evidence = append(rec.Evidence, model.Evidence{ID: nextEID(), Kind: model.EvidenceTranscript, Timestamp: time.Now().UTC(),
				Data: map[string]any{"adapter": a.Name(), "session_id": ref.ID, "message_count": len(msgs)}})
			got, ewarns := extractWithPolicy(ctx, msgs, cfg)
			rec.Claims = got
			for _, w := range ewarns {
				if w.Code == model.ExtractionConflictAgents {
					rec.Flags = append(rec.Flags, model.Flag{Code: w.Code, Severity: model.SeverityWarning, Title: "Conflicting agent claims", Details: w.Message, EvidenceRefs: []string{}})
				} else {
					rec.Warnings = append(rec.Warnings, w.Code+": "+w.Message)
				}
			}
		}
		_ = attempts
		rec.Evidence = append(rec.Evidence, model.Evidence{ID: nextEID(), Kind: model.EvidenceAdapterStatus, Timestamp: time.Now().UTC(),
			Data: map[string]any{"selected": a.Name(), "session_id": ref.ID, "attempts": attemptData(attempts)}})
	} else {
		rec.Warnings = append(rec.Warnings, "adapter issue: "+attemptSummary(attempts))
		rec.Evidence = append(rec.Evidence, model.Evidence{ID: nextEID(), Kind: model.EvidenceAdapterStatus, Timestamp: time.Now().UTC(),
			Data: map[string]any{"selected": "", "attempts": attemptData(attempts)}})
	}
	rec.Agent = model.Agent{Name: agentDisplay, SessionID: sessionID}

	eng := verdict.DeterministicEngine{}
	if _, _, err := eng.Evaluate(&rec); err != nil {
		return rec, err
	}
	integrity.FillFlagIDs(rec.Flags)
	st := store.New(filepath.Join(abs, ".receipts"))
	id, err := store.GenerateID(func(id string) bool {
		_, serr := os.Stat(filepath.Join(abs, ".receipts", "receipts", id+".json"))
		return serr == nil
	}, nil)
	if err != nil {
		return rec, err
	}
	rec.ID = id
	rec.DurationMS = time.Since(start).Milliseconds()
	if err := verdict.CheckInvariants(&rec); err != nil {
		return rec, err
	}
	if err := model.ValidateReceipt(rec); err != nil {
		return rec, err
	}
	if err := st.Save(ctx, rec); err != nil {
		return rec, err
	}
	stored, err := st.Get(ctx, id)
	if err != nil {
		return rec, err
	}
	return stored, nil
}
