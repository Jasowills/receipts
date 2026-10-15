package gitstate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrNotGit is returned when dir is not inside a Git working tree.
var ErrNotGit = errors.New("gitstate: not a git repository")

// runGit runs git -C dir args through a controlled process runner.
// It captures stdout and stderr separately and supports context
// cancellation. Stderr is truncated in errors to avoid large output.
func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	full := append([]string{"-C", dir}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		se := strings.TrimSpace(stderr.String())
		if strings.Contains(se, "not a git repository") {
			return "", fmt.Errorf("%w: %s", ErrNotGit, firstLine(se))
		}
		if ctx.Err() != nil {
			return "", fmt.Errorf("git %v canceled: %w", args, ctx.Err())
		}
		return "", fmt.Errorf("git %v failed: %s", args, firstLine(se))
	}
	return stdout.String(), nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 500 {
		s = s[:500]
	}
	if s == "" {
		s = "unknown git error"
	}
	return s
}

// FindRoot returns the repository top level for workdir.
func FindRoot(ctx context.Context, workdir string) (string, error) {
	out, err := runGit(ctx, workdir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	root := strings.TrimSpace(out)
	if root == "" {
		return "", fmt.Errorf("git rev-parse returned empty root")
	}
	return root, nil
}

// Head returns the full HEAD commit SHA for a repository directory.
func Head(ctx context.Context, dir string) (string, error) {
	out, err := runGit(ctx, dir, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Branch returns the current branch name, or HEAD when detached.
func Branch(ctx context.Context, dir string) (string, error) {
	out, err := runGit(ctx, dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
