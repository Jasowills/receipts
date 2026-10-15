package reverify

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func initRepo(t *testing.T, dirty bool) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@e.c", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@e.c")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "t@e.c")
	run("config", "user.name", "T")
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("v1\n"), 0644)
	run("add", ".")
	run("commit", "-m", "init")
	if dirty {
		os.WriteFile(filepath.Join(dir, "a.txt"), []byte("dirty\n"), 0644)
	}
	return dir
}

func headOf(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	return string(bytesTrim(out))
}

func bytesTrim(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == ' ' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return b
}

func TestSetupAndCleanup(t *testing.T) {
	dir := initRepo(t, false)
	head := headOf(t, dir)
	ctx := context.Background()
	wt, cleanup, err := SetupCommittedWorktree(ctx, dir, head)
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if _, err := os.Stat(filepath.Join(wt, "a.txt")); err != nil {
		t.Errorf("worktree missing file: %v", err)
	}
	cleanup()
	cleanup()
	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Errorf("worktree dir should be gone")
	}
	cmd := exec.Command("git", "-C", dir, "worktree", "list", "--porcelain")
	out, _ := cmd.Output()
	for _, line := range splitLines(string(out)) {
		if strings.Contains(line, "receipts-reverify-") {
			t.Errorf("leaked worktree entry: %q", line)
		}
	}
}

func splitLines(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == '\n' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func TestRefusesNoHead(t *testing.T) {
	dir := t.TempDir()
	exec.Command("git", "-C", dir, "init").Run()
	if _, _, err := SetupCommittedWorktree(context.Background(), dir, ""); err == nil {
		t.Errorf("expected error with empty head, got nil")
	}
}

func TestSafetyDirty(t *testing.T) {
	dir := initRepo(t, true)
	head := headOf(t, dir)
	if err := CheckSafe(context.Background(), dir, head, false); err == nil {
		t.Errorf("dirty checkout should refuse without force")
	}
	if err := CheckSafe(context.Background(), dir, head, true); err != nil {
		t.Errorf("force should allow dirty checkout: %v", err)
	}
	clean := initRepo(t, false)
	if err := CheckSafe(context.Background(), clean, headOf(t, clean), false); err != nil {
		t.Errorf("clean checkout should pass: %v", err)
	}
}

func TestRunCommandPassFail(t *testing.T) {
	dir := initRepo(t, false)
	head := headOf(t, dir)
	ctx := context.Background()
	wt, cleanup, err := SetupCommittedWorktree(ctx, dir, head)
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	defer cleanup()
	res, err := RunCommand(ctx, wt, "true", 30*time.Second)
	if err != nil {
		t.Fatalf("RunCommand: %v", err)
	}
	if res.ExitCode != 0 || res.TimedOut {
		t.Errorf("true should exit 0: %+v", res)
	}
	res, err = RunCommand(ctx, wt, "false", 30*time.Second)
	if err != nil {
		t.Fatalf("RunCommand: %v", err)
	}
	if res.ExitCode == 0 {
		t.Errorf("false should exit nonzero")
	}
}

func TestRunCommandRefusesShell(t *testing.T) {
	dir := initRepo(t, false)
	if _, err := RunCommand(context.Background(), dir, "go test ./... | head", 30*time.Second); err == nil {
		t.Errorf("pipelines must refuse without a shell")
	}
}

func TestOriginalUntouched(t *testing.T) {
	dir := initRepo(t, false)
	before, _ := os.ReadFile(filepath.Join(dir, "a.txt"))
	head := headOf(t, dir)
	ctx := context.Background()
	wt, cleanup, err := SetupCommittedWorktree(ctx, dir, head)
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	defer cleanup()
	os.WriteFile(filepath.Join(wt, "a.txt"), []byte("mutated in worktree\n"), 0644)
	after, _ := os.ReadFile(filepath.Join(dir, "a.txt"))
	if string(after) != string(before) {
		t.Errorf("original checkout must never change")
	}
}

func TestSafetyIgnoresOwnedDirt(t *testing.T) {
	dir := initRepo(t, false)
	os.MkdirAll(filepath.Join(dir, ".receipts", "receipts"), 0755)
	os.WriteFile(filepath.Join(dir, ".receipts", "receipts", "RC-0001.json"), []byte("{}\n"), 0644)
	if err := CheckSafe(context.Background(), dir, headOf(t, dir), false); err != nil {
		t.Errorf("tool owned dirt must not block: %v", err)
	}
}
