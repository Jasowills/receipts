package gitstate

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/receipts/receipts/internal/model"
)

func TestIsRepo(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	if IsRepo(ctx, dir) {
		t.Errorf("fresh temp dir should not be a repo")
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// init via git directly.
	cmd := exec.Command("git", "-C", dir, "init")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	if !IsRepo(ctx, dir) {
		t.Errorf("initialized dir should be a repo")
	}
	if !IsRepo(ctx, filepath.Join(dir, "sub")) {
		t.Errorf("subdir of repo should count as repo")
	}
}

func TestNonGitRepo(t *testing.T) {
	r := NonGitRepo("/tmp/work")
	if r.NonGit == nil || !*r.NonGit {
		t.Errorf("NonGit flag must be true")
	}
	if r.BaselineConfidence != model.BaselineUnknown {
		t.Errorf("confidence = %q, want unknown", r.BaselineConfidence)
	}
	if r.Root == "" {
		t.Errorf("root must be set")
	}
	if WarnNonGitProject != "NON_GIT_PROJECT" {
		t.Errorf("warning = %q, want NON_GIT_PROJECT", WarnNonGitProject)
	}
}
