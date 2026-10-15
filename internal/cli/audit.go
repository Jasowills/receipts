package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/receipts/receipts/internal/adapters"
	"github.com/receipts/receipts/internal/adapters/registry"
	"github.com/receipts/receipts/internal/collectors"
	"github.com/receipts/receipts/internal/integrity"
	"github.com/receipts/receipts/internal/model"
	"github.com/receipts/receipts/internal/render"
	"github.com/receipts/receipts/internal/store"
	"github.com/receipts/receipts/internal/verdict"
)

const auditUse = "receipts audit [--workdir PATH] [--agent auto|opencode|claude] [--format terminal|md|json|html] [--out PATH]"

func printAuditHelp(w io.Writer) {
	fmt.Fprintln(w, "audit - Audit the latest agent session against Git evidence")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintf(w, "  %s\n", auditUse)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --agent auto|opencode|claude   Transcript adapter (default auto)")
	fmt.Fprintln(w, "  --format terminal|md|json|html     Output format (default terminal)")
	fmt.Fprintln(w, "  --workdir PATH                Project directory (default .)")
	fmt.Fprintln(w, "  --out PATH                    Write output to file instead of stdout")
	fmt.Fprintln(w, "  --help                        Show help for this command")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Exit 0 means the audit ran; it says nothing about claim verdicts.")
	fmt.Fprintln(w, "Without a transcript the receipt still reports Git checks with zero claims.")
}

// runAudit implements receipts audit. Exit codes: 0 ran (any verdicts),
// 2 usage error, 3 collection failure preventing a valid receipt,
// 4 store or schema failure.
func runAudit(args []string, stdout, stderr io.Writer) int {
	workdir, agentName, format, out := ".", "auto", "terminal", ""
	agentSet := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--help" || a == "-h":
			printAuditHelp(stdout)
			return 0
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
		case a == "--format" && i+1 < len(args):
			i++
			format = args[i]
		case strings.HasPrefix(a, "--format="):
			format = strings.TrimPrefix(a, "--format=")
		case a == "--out" && i+1 < len(args):
			i++
			out = args[i]
		case strings.HasPrefix(a, "--out="):
			out = strings.TrimPrefix(a, "--out=")
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(stderr, "unknown flag: %s\n", a)
			fmt.Fprintln(stderr, "Run receipts audit --help for usage.")
			return 2
		default:
			fmt.Fprintf(stderr, "unexpected argument: %s\n", a)
			return 2
		}
	}
	if agentName != "auto" && agentName != "opencode" && agentName != "claude" {
		fmt.Fprintf(stderr, "unknown agent: %s (expected auto|opencode|claude)\n", agentName)
		return 2
	}
	switch format {
	case "terminal", "md", "markdown", "json", "html":
	default:
		fmt.Fprintf(stderr, "unknown format: %s (expected terminal|md|json)\n", format)
		return 2
	}

	start := time.Now()
	ctx := context.Background()
	abs, err := filepath.Abs(workdir)
	if err != nil {
		fmt.Fprintf(stderr, "audit: %v\n", err)
		return 3
	}
	if _, err := os.Stat(abs); err != nil {
		fmt.Fprintf(stderr, "audit: workdir %s: %v\n", abs, err)
		return 3
	}
	cfg, err := loadPolicy(abs)
	if err != nil {
		fmt.Fprintf(stderr, "audit: %v\n", err)
		return 2
	}
	if !agentSet {
		agentName = cfg.Agent
	}

	gev, err := collectors.BuildGitEvidenceWith(ctx, abs, collectors.GitOptions{
		Scope: policyScope(cfg), Checks: policyChecks(cfg),
	})
	if err != nil {
		fmt.Fprintf(stderr, "audit: git collection failed: %v\n", err)
		return 3
	}

	rec := model.Receipt{
		SchemaVersion: model.SchemaVersion,
		CreatedAt:     time.Now().UTC(),
		Tier:          model.Tier0,
		Repo:          gev.Repo,
		Claims:        []model.Claim{},
		Flags:         gev.Flags,
		Evidence:      gev.Evidence,
		Warnings:      append(gev.Warnings, policyNotes(cfg)...),
		Summary:       model.Summary{},
		Chain:         model.Chain{Algorithm: model.ChainAlgorithmSHA256},
	}

	nextEID := func() string { return fmt.Sprintf("E-%03d", len(rec.Evidence)+1) }

	// Transcript collection never blocks the Git receipt.
	agentDisplay, sessionID := "none", ""
	if a, ref, attempts, ok := selectAdapter(ctx, abs, agentName); ok {
		agentDisplay, sessionID = a.Name(), ref.ID
		msgs, merr := a.Messages(ctx, ref)
		if merr != nil {
			rec.Warnings = append(rec.Warnings, "adapter issue: "+a.Name()+": "+merr.Error())
		} else {
			rec.Evidence = append(rec.Evidence, model.Evidence{
				ID: nextEID(), Kind: model.EvidenceTranscript, Timestamp: time.Now().UTC(),
				Data: map[string]any{"adapter": a.Name(), "session_id": ref.ID, "message_count": len(msgs)},
			})
			got, ewarns := extractWithPolicy(ctx, msgs, cfg)
			rec.Claims = got
			for _, w := range ewarns {
				if w.Code == model.ExtractionConflictAgents {
					rec.Flags = append(rec.Flags, model.Flag{
						Code: w.Code, Severity: model.SeverityWarning,
						Title: "Conflicting agent claims", Details: w.Message, EvidenceRefs: []string{},
					})
				} else {
					rec.Warnings = append(rec.Warnings, w.Code+": "+w.Message)
				}
			}
		}
		_ = attempts
		rec.Evidence = append(rec.Evidence, model.Evidence{
			ID: nextEID(), Kind: model.EvidenceAdapterStatus, Timestamp: time.Now().UTC(),
			Data: map[string]any{"selected": a.Name(), "session_id": ref.ID, "attempts": attemptData(attempts)},
		})
	} else {
		rec.Warnings = append(rec.Warnings, "adapter issue: "+attemptSummary(attempts))
		rec.Evidence = append(rec.Evidence, model.Evidence{
			ID: nextEID(), Kind: model.EvidenceAdapterStatus, Timestamp: time.Now().UTC(),
			Data: map[string]any{"selected": "", "attempts": attemptData(attempts)},
		})
	}
	rec.Agent = model.Agent{Name: agentDisplay, SessionID: sessionID}

	eng := verdict.DeterministicEngine{}
	if _, _, err := eng.Evaluate(&rec); err != nil {
		fmt.Fprintf(stderr, "audit: %v\n", err)
		return 4
	}
	integrity.FillFlagIDs(rec.Flags)

	st := store.New(filepath.Join(abs, ".receipts"))
	id, err := store.GenerateID(func(id string) bool {
		_, serr := os.Stat(filepath.Join(abs, ".receipts", "receipts", id+".json"))
		return serr == nil
	}, nil)
	if err != nil {
		fmt.Fprintf(stderr, "audit: %v\n", err)
		return 4
	}
	rec.ID = id
	rec.DurationMS = time.Since(start).Milliseconds()
	if err := verdict.CheckInvariants(&rec); err != nil {
		fmt.Fprintf(stderr, "audit: %v\n", err)
		return 4
	}
	if err := model.ValidateReceipt(rec); err != nil {
		fmt.Fprintf(stderr, "audit: %v\n", err)
		return 4
	}

	if err := st.Save(ctx, rec); err != nil {
		fmt.Fprintf(stderr, "audit: store failed: %v\n", err)
		return 4
	}
	stored, err := st.Get(ctx, id)
	if err != nil {
		fmt.Fprintf(stderr, "audit: %v\n", err)
		return 4
	}

	var data []byte
	switch format {
	case "terminal":
		data, err = render.TerminalRenderer{}.Render(stored, render.Options{})
	case "md", "markdown":
		data, err = render.MarkdownRenderer{}.Render(stored, render.Options{})
	case "html":
		data, err = render.HTMLRenderer{}.Render(stored, render.Options{})
	case "json":
		var jb []byte
		jb, err = store.MarshalCanonical(stored)
		if err == nil {
			data = append(jb, '\n')
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "audit: render failed: %v\n", err)
		return 4
	}
	if out != "" {
		if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
			fmt.Fprintf(stderr, "audit: %v\n", err)
			return 4
		}
		if err := os.WriteFile(out, data, 0644); err != nil {
			fmt.Fprintf(stderr, "audit: %v\n", err)
			return 4
		}
		fmt.Fprintf(stdout, "Wrote %s\n", out)
	} else if _, err := stdout.Write(data); err != nil {
		fmt.Fprintf(stderr, "audit: %v\n", err)
		return 4
	}
	rel, _ := filepath.Rel(abs, filepath.Join(abs, ".receipts", "receipts", id+".json"))
	fmt.Fprintf(stderr, "Saved: %s\n", rel)
	if data, err := os.ReadFile(filepath.Join(abs, ".gitignore")); err != nil || !hasIgnoreEntry(string(data)) {
		fmt.Fprintln(stderr, "recommend: run receipts init to ignore .receipts/")
	}
	return 0
}

func selectAdapter(ctx context.Context, workdir, name string) (adapters.Adapter, adapters.SessionRef, []adapters.Attempt, bool) {
	if name == "auto" {
		res, err := registry.Auto(ctx, workdir)
		if err != nil {
			return nil, adapters.SessionRef{}, res.Attempts, false
		}
		return res.Adapter, res.Ref, res.Attempts, true
	}
	a, err := registry.ByName(name)
	if err != nil {
		return nil, adapters.SessionRef{}, []adapters.Attempt{{Adapter: name, Reason: err.Error()}}, false
	}
	if ok, derr := a.Detect(ctx, workdir); derr != nil || !ok {
		reason := "storage not detected"
		if derr != nil {
			reason = derr.Error()
		}
		return nil, adapters.SessionRef{}, []adapters.Attempt{{Adapter: name, Reason: reason}}, false
	}
	ref, lerr := a.LatestSession(ctx, workdir)
	if lerr != nil {
		return nil, adapters.SessionRef{}, []adapters.Attempt{{Adapter: name, Detected: true, Reason: lerr.Error()}}, false
	}
	return a, ref, []adapters.Attempt{{Adapter: name, Detected: true, SessionID: ref.ID}}, true
}

func attemptData(attempts []adapters.Attempt) []any {
	out := make([]any, 0, len(attempts))
	for _, a := range attempts {
		out = append(out, map[string]any{
			"adapter": a.Adapter, "detected": a.Detected,
			"session_id": a.SessionID, "reason": a.Reason,
		})
	}
	return out
}

func attemptSummary(attempts []adapters.Attempt) string {
	if len(attempts) == 0 {
		return "no transcript adapter detected"
	}
	parts := make([]string, 0, len(attempts))
	for _, a := range attempts {
		if a.SessionID != "" {
			continue
		}
		parts = append(parts, a.Adapter+": "+a.Reason)
	}
	if len(parts) == 0 {
		return "no transcript session found"
	}
	return strings.Join(parts, "; ")
}
