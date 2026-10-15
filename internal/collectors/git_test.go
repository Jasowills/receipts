package collectors

import (
	"context"
	"github.com/receipts/receipts/internal/gitstate"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@e.c", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@e.c")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestBuildGitEvidence(t *testing.T) {
	dir := t.TempDir()
	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "t@e.c")
	gitRun(t, dir, "config", "user.name", "T")
	os.WriteFile(filepath.Join(dir, "src.test.ts"), []byte("expect(a)\n"), 0644)
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "init")
	os.WriteFile(filepath.Join(dir, "src.test.ts"), []byte("expect(a)\nexpect(b)\n"), 0644)

	got, err := BuildGitEvidence(context.Background(), dir)
	if err != nil {
		t.Fatalf("BuildGitEvidence: %v", err)
	}
	if got.Repo.BaselineConfidence != "low" {
		t.Errorf("confidence = %q, want low", got.Repo.BaselineConfidence)
	}
	if len(got.Evidence) != 2 || got.Evidence[0].ID != "E-001" || got.Evidence[1].ID != "E-002" {
		t.Fatalf("evidence = %+v, want E-001 git_snapshot + E-002 diff", got.Evidence)
	}
	found := false
	for _, f := range got.Flags {
		if f.Code == "TEST_FILE_MODIFIED" {
			found = true
			if len(f.EvidenceRefs) == 0 || f.EvidenceRefs[0] != "E-002" {
				t.Errorf("flag should reference diff evidence, got %+v", f)
			}
		}
	}
	if !found {
		t.Errorf("expected TEST_FILE_MODIFIED, got %+v", got.Flags)
	}
}

func TestBuildGitEvidenceNonGit(t *testing.T) {
	got, err := BuildGitEvidence(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("non git must not error: %v", err)
	}
	if got.Repo.NonGit == nil || !*got.Repo.NonGit {
		t.Errorf("expected NonGit repo, got %+v", got.Repo)
	}
	if len(got.Evidence) != 0 {
		t.Errorf("non git must have no git evidence, got %+v", got.Evidence)
	}
	found := false
	for _, w := range got.Warnings {
		if w == "NON_GIT_PROJECT" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected NON_GIT_PROJECT warning, got %v", got.Warnings)
	}
}

func TestDropOwnedDir(t *testing.T) {
	in := []gitstate.ChangedPath{{Path: ".receipts/receipts/RC-1.json"}, {Path: "src/a.ts"}, {Path: ".receipts"}}
	got := DropOwnedDir(in)
	if len(got) != 1 || got[0].Path != "src/a.ts" {
		t.Errorf("got %+v, want only src/a.ts", got)
	}
}
