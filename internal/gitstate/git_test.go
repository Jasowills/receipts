package gitstate

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Receipts Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Receipts Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@example.com")
	gitRun(t, dir, "config", "user.name", "Receipts Test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	gitRun(t, dir, "add", "README.md")
	gitRun(t, dir, "commit", "-m", "initial")
	return dir
}

func TestFindRoot(t *testing.T) {
	ctx := context.Background()
	dir := initRepo(t)
	sub := filepath.Join(dir, "sub", "nested")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	root, err := FindRoot(ctx, sub)
	if err != nil {
		t.Fatalf("FindRoot failed: %v", err)
	}
	want, _ := filepath.EvalSymlinks(dir)
	got, _ := filepath.EvalSymlinks(root)
	if got != want {
		t.Errorf("FindRoot = %q, want %q", root, dir)
	}
}

func TestFindRootOutsideRepo(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	_, err := FindRoot(ctx, dir)
	if !errors.Is(err, ErrNotGit) {
		t.Errorf("err = %v, want ErrNotGit", err)
	}
}

func TestHeadAndBranch(t *testing.T) {
	ctx := context.Background()
	dir := initRepo(t)
	head, err := Head(ctx, dir)
	if err != nil {
		t.Fatalf("Head failed: %v", err)
	}
	if len(head) != 40 || strings.Trim(head, "0123456789abcdef") != "" {
		t.Errorf("Head = %q, want 40 hex chars", head)
	}
	branch, err := Branch(ctx, dir)
	if err != nil {
		t.Fatalf("Branch failed: %v", err)
	}
	if branch == "" {
		t.Errorf("Branch is empty")
	}
}

func TestHeadNoCommits(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	gitRun(t, dir, "init")
	_, err := Head(ctx, dir)
	if err == nil {
		t.Errorf("expected error for HEAD with no commits, got nil")
	}
}

func TestStatusDirtyAndUntracked(t *testing.T) {
	ctx := context.Background()
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("changed\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes with space.txt"), []byte("new\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	snap, err := Collect(ctx, dir)
	if err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}
	if !snap.Dirty {
		t.Errorf("Snapshot.Dirty = false, want true")
	}
	paths := map[string]bool{}
	for _, f := range snap.Files {
		paths[f.Path] = true
	}
	if !paths["README.md"] {
		t.Errorf("status missing modified README.md, got %+v", snap.Files)
	}
	if !paths["notes with space.txt"] {
		t.Errorf("status missing spaced path, got %+v", snap.Files)
	}
}

func TestStatusClean(t *testing.T) {
	ctx := context.Background()
	dir := initRepo(t)
	snap, err := Collect(ctx, dir)
	if err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}
	if snap.Dirty {
		t.Errorf("clean repo Dirty = true, files %+v", snap.Files)
	}
}

func TestRunnerRespectsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dir := initRepo(t)
	_, err := FindRoot(ctx, dir)
	if err == nil {
		t.Errorf("expected error for canceled context, got nil")
	}
}

func TestSnapshotOutsideRepoFails(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dir := t.TempDir()
	_, err := Collect(ctx, dir)
	if !errors.Is(err, ErrNotGit) {
		t.Errorf("err = %v, want ErrNotGit", err)
	}
}

func TestGitMissingNamesExecutable(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := FindRoot(context.Background(), t.TempDir())
	if err == nil {
		t.Fatalf("expected error without git, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "not found in PATH") {
		t.Errorf("error should name the missing executable, got %q", got)
	}
	if errors.Is(err, ErrNotGit) {
		t.Errorf("missing git must not masquerade as non repo")
	}
}
