package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/receipts/receipts/internal/gitstate"
	"github.com/receipts/receipts/internal/model"
	"github.com/receipts/receipts/internal/render"
	"github.com/receipts/receipts/internal/reverify"
	"github.com/receipts/receipts/internal/store"
	"github.com/receipts/receipts/internal/verdict"
)

const checkUse = "receipts check [ID] [--reverify] [--command CMD] [--force] [--workdir PATH] [--format terminal|md|json] [--out PATH]"

func printCheckHelp(w io.Writer) {
	fmt.Fprintln(w, "check - Re-evaluate a stored receipt")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintf(w, "  %s\n", checkUse)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --reverify            Run the test command in an isolated worktree and append the result")
	fmt.Fprintln(w, "  --command CMD         Test command for --reverify (simple argv, no shell)")
	fmt.Fprintln(w, "  --force               Test the committed state even with a dirty checkout")
	fmt.Fprintln(w, "  --workdir PATH        Project directory (default .)")
	fmt.Fprintln(w, "  --format terminal|md|json|html   Output format (default terminal)")
	fmt.Fprintln(w, "  --out PATH            Write output to file instead of stdout")
	fmt.Fprintln(w, "  --help                Show help for this command")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Without --reverify no project command runs. Re-verification tests")
	fmt.Fprintln(w, "the committed state only in a temp worktree and never touches")
	fmt.Fprintln(w, "the original checkout. --command defaults to the configured")
	fmt.Fprintln(w, "test command from receipts.toml when present.")
}

// runCheck implements receipts check. Exit 0 evaluated, 2 usage error,
// 4 store failure, 5 re-verification unsafe or failed to start.
func runCheck(args []string, stdout, stderr io.Writer) int {
	var id, command, workdir, format, out string
	var reverifyFlag, force bool
	workdir, format = ".", "terminal"
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--help" || a == "-h":
			printCheckHelp(stdout)
			return 0
		case a == "--reverify":
			reverifyFlag = true
		case a == "--force":
			force = true
		case a == "--command" && i+1 < len(args):
			i++
			command = args[i]
		case strings.HasPrefix(a, "--command="):
			command = strings.TrimPrefix(a, "--command=")
		case a == "--workdir" && i+1 < len(args):
			i++
			workdir = args[i]
		case strings.HasPrefix(a, "--workdir="):
			workdir = strings.TrimPrefix(a, "--workdir=")
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
			return 2
		default:
			if id != "" {
				fmt.Fprintln(stderr, "too many arguments: expected at most one ID")
				return 2
			}
			id = a
		}
	}
	switch format {
	case "terminal", "md", "markdown", "json", "html":
	default:
		fmt.Fprintf(stderr, "unknown format: %s\n", format)
		return 2
	}
	abs, err := filepath.Abs(workdir)
	if err != nil {
		fmt.Fprintf(stderr, "check: %v\n", err)
		return 4
	}
	cfg, err := loadPolicy(abs)
	if err != nil {
		fmt.Fprintf(stderr, "check: %v\n", err)
		return 2
	}
	ctx := context.Background()
	st := store.New(filepath.Join(abs, ".receipts"))
	var rec model.Receipt
	if id == "" {
		rec, err = st.Latest(ctx)
	} else {
		rec, err = st.Get(ctx, id)
	}
	if err != nil {
		fmt.Fprintf(stderr, "check: %v\n", err)
		return 4
	}

	if stale := staleNote(ctx, abs, rec); stale != "" {
		fmt.Fprintf(stderr, "warning: %s\n", stale)
	}

	if !reverifyFlag {
		eng := verdict.DeterministicEngine{}
		if _, _, err := eng.Evaluate(&rec); err != nil {
			fmt.Fprintf(stderr, "check: %v\n", err)
			return 4
		}
		return renderReceipt(rec, format, out, stdout, stderr)
	}

	if command == "" {
		command = cfg.Test.Command
	}
	if command == "" {
		fmt.Fprintln(stderr, "check: --reverify needs --command CMD or a configured test.command")
		return 2
	}
	if rec.Repo.NonGit != nil && *rec.Repo.NonGit {
		fmt.Fprintln(stderr, "check: re-verification is unavailable outside a Git repository")
		return 5
	}
	head := rec.Repo.HeadAfter
	if head == "" {
		fmt.Fprintln(stderr, "check: re-verification needs a committed final state")
		return 5
	}
	root := rec.Repo.Root
	if root == "" {
		root = abs
	}
	if err := reverify.CheckSafe(ctx, root, head, force); err != nil {
		fmt.Fprintf(stderr, "check: %v\n", err)
		return 5
	}
	wt, cleanup, err := reverify.SetupCommittedWorktree(ctx, root, head)
	if err != nil {
		fmt.Fprintf(stderr, "check: %v\n", err)
		return 5
	}
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	rctx, cancel := context.WithCancel(ctx)
	interrupted := false
	done := make(chan struct{})
	defer func() {
		signal.Stop(sigCh)
		close(done)
	}()
	go func() {
		select {
		case <-sigCh:
			interrupted = true
			cleanup()
			cancel()
		case <-done:
		}
	}()
	res, err := reverify.RunCommand(rctx, wt, command, 900*time.Second)
	cancel()
	cleanup()
	if err != nil {
		fmt.Fprintf(stderr, "check: %v\n", err)
		return 5
	}
	if interrupted {
		fmt.Fprintln(stderr, "warning: interrupted: receipt covers a partial run")
	}
	ev := model.Evidence{
		ID: fmt.Sprintf("E-%03d", len(rec.Evidence)+1), Kind: model.EvidenceReverify, Timestamp: time.Now().UTC(),
		Data: map[string]any{
			"argv": res.Argv, "command": res.Command, "exit_code": float64(res.ExitCode),
			"checkout": head, "committed_state_only": true,
			"duration_ms": res.Duration.Milliseconds(), "output_tail": res.Output,
			"timed_out": res.TimedOut, "truncated": res.Truncated,
		},
	}
	rec.Evidence = append(rec.Evidence, ev)
	eng := verdict.DeterministicEngine{}
	if _, _, err := eng.Evaluate(&rec); err != nil {
		fmt.Fprintf(stderr, "check: %v\n", err)
		return 4
	}
	newID, err := store.GenerateID(func(id string) bool {
		_, serr := os.Stat(filepath.Join(abs, ".receipts", "receipts", id+".json"))
		return serr == nil
	}, nil)
	if err != nil {
		fmt.Fprintf(stderr, "check: %v\n", err)
		return 4
	}
	oldID := rec.ID
	rec.ID = newID
	if err := verdict.CheckInvariants(&rec); err != nil {
		fmt.Fprintf(stderr, "check: %v\n", err)
		return 4
	}
	if err := model.ValidateReceipt(rec); err != nil {
		fmt.Fprintf(stderr, "check: %v\n", err)
		return 4
	}
	if err := st.Save(ctx, rec); err != nil {
		fmt.Fprintf(stderr, "check: %v\n", err)
		return 4
	}
	stored, err := st.Get(ctx, newID)
	if err != nil {
		fmt.Fprintf(stderr, "check: %v\n", err)
		return 4
	}
	fmt.Fprintf(stderr, "Re-verified: %s (from %s)\n", newID, oldID)
	if code := renderReceipt(stored, format, out, stdout, stderr); code != 0 {
		return code
	}
	if interrupted {
		return 130
	}
	return 0
}

// staleNote reports when current source drifted from the receipt snapshot.
// Tool owned .receipts/ noise never counts as drift.
func staleNote(ctx context.Context, abs string, rec model.Receipt) string {
	if rec.Repo.NonGit != nil && *rec.Repo.NonGit {
		return ""
	}
	root := rec.Repo.Root
	if root == "" {
		root = abs
	}
	snap, err := gitstate.Collect(ctx, root)
	if err != nil {
		return ""
	}
	if snap.Head != "" && rec.Repo.HeadAfter != "" && snap.Head != rec.Repo.HeadAfter {
		return "receipt is stale: repository HEAD moved since the receipt"
	}
	if rec.Repo.DirtyAfter != nil {
		changes, err := gitstate.Changed(ctx, root)
		if err == nil {
			dirty := false
			for _, c := range changes {
				if c.Path == ".receipts" || strings.HasPrefix(c.Path, ".receipts/") {
					continue
				}
				dirty = true
				break
			}
			if dirty != *rec.Repo.DirtyAfter {
				return "receipt is stale: working tree dirtiness changed since the receipt"
			}
		}
	}
	return ""
}

func renderReceipt(rec model.Receipt, format, out string, stdout, stderr io.Writer) int {
	var data []byte
	var err error
	switch format {
	case "terminal":
		data, err = render.TerminalRenderer{}.Render(rec, render.Options{})
	case "md", "markdown":
		data, err = render.MarkdownRenderer{}.Render(rec, render.Options{})
	case "html":
		data, err = render.HTMLRenderer{}.Render(rec, render.Options{})
	case "json":
		var jb []byte
		jb, err = store.MarshalCanonical(rec)
		if err == nil {
			data = append(jb, '\n')
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "check: render failed: %v\n", err)
		return 4
	}
	if out != "" {
		if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
			fmt.Fprintf(stderr, "check: %v\n", err)
			return 4
		}
		if err := os.WriteFile(out, data, 0644); err != nil {
			fmt.Fprintf(stderr, "check: %v\n", err)
			return 4
		}
		fmt.Fprintf(stdout, "Wrote %s\n", out)
		return 0
	}
	_, err = stdout.Write(data)
	if err != nil {
		fmt.Fprintf(stderr, "check: %v\n", err)
		return 4
	}
	return 0
}
