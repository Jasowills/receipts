package reverify

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// WorktreePrefix marks temp directories owned by re-verification.
// Cleanup only removes paths carrying this prefix inside the temp root.
const WorktreePrefix = "receipts-reverify-"

// SetupCommittedWorktree creates a detached worktree at head in a unique
// temp directory outside the repository. The original checkout is never
// reset, cleaned, or checked out. Cleanup is idempotent and removes only
// Receipts created paths.
func SetupCommittedWorktree(ctx context.Context, repoRoot, head string) (string, func(), error) {
	noop := func() {}
	if strings.TrimSpace(repoRoot) == "" || strings.TrimSpace(head) == "" {
		return "", noop, fmt.Errorf("reverify: repository root and HEAD are required")
	}
	tmp, err := os.MkdirTemp("", WorktreePrefix)
	if err != nil {
		return "", noop, err
	}
	wt := filepath.Join(tmp, "worktree")
	cmd := exec.CommandContext(ctx, "git", "-C", repoRoot, "worktree", "add", "--detach", wt, head)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmp)
		return "", noop, fmt.Errorf("reverify: worktree add failed: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	cleanup := func() {
		rm := exec.Command("git", "-C", repoRoot, "worktree", "remove", "--force", wt)
		rm.Run()
		if isOwnedTemp(tmp) {
			os.RemoveAll(tmp)
		}
	}
	return wt, cleanup, nil
}

func isOwnedTemp(dir string) bool {
	return strings.HasPrefix(filepath.Base(dir), WorktreePrefix) &&
		filepath.Dir(dir) == os.TempDir()
}

// Result is one isolated command run.
type Result struct {
	Argv      []string
	Command   string
	ExitCode  int
	Output    string
	Truncated bool
	Duration  time.Duration
	TimedOut  bool
}

// MaxOutputBytes bounds captured re-verification output.
const MaxOutputBytes = 8192

// RunCommand executes argv without a shell in dir with a timeout.
// Commands containing shell metacharacters are refused; pipelines need
// explicit shell configuration, which v0.1 does not offer.
func RunCommand(ctx context.Context, dir, command string, timeout time.Duration) (Result, error) {
	argv, err := splitCommand(command)
	if err != nil {
		return Result{}, err
	}
	if hasShellSyntax(command) {
		return Result{}, fmt.Errorf("reverify: command requires a shell (%q); shell execution is disabled by default", command)
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	start := time.Now()
	cmd := exec.CommandContext(cctx, argv[0], argv[1:]...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	dur := time.Since(start)
	res := Result{Argv: argv, Command: command, Duration: dur}
	switch {
	case cctx.Err() == context.DeadlineExceeded:
		res.TimedOut = true
		res.ExitCode = 124
	case ctx.Err() != nil:
		res.ExitCode = 130
	case runErr != nil:
		if ee, ok := runErr.(*exec.ExitError); ok {
			if code := ee.ProcessState.ExitCode(); code >= 0 {
				res.ExitCode = code
			} else {
				res.ExitCode = 130
			}
		} else {
			return res, fmt.Errorf("reverify: failed to start: %w", runErr)
		}
	}
	combined := stdout.String()
	if stderr.Len() > 0 {
		combined += stderr.String()
	}
	if len(combined) > MaxOutputBytes {
		combined = combined[len(combined)-MaxOutputBytes:]
		res.Truncated = true
	}
	res.Output = combined
	return res, nil
}

// splitCommand splits on whitespace honoring single and double quotes.
func splitCommand(s string) ([]string, error) {
	var out []string
	var cur strings.Builder
	var quote byte
	esc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if esc {
			cur.WriteByte(c)
			esc = false
			continue
		}
		if c == '\\' && quote != '\'' {
			esc = true
			continue
		}
		if quote != 0 {
			if c == quote {
				quote = 0
				continue
			}
			cur.WriteByte(c)
			continue
		}
		switch {
		case c == '"' || c == '\'':
			quote = c
		case c == ' ' || c == '\t' || c == '\n':
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteByte(c)
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("reverify: unterminated quote in %q", s)
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("reverify: empty command")
	}
	return out, nil
}

func hasShellSyntax(s string) bool {
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote = c
			continue
		}
		switch c {
		case '|', '&', ';', '$', '`', '>', '<', '(', ')', '*', '?', '#', '~':
			return true
		}
	}
	return false
}
