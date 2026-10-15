package cli

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/receipts/receipts/internal/model"
	"github.com/receipts/receipts/internal/store"
	"github.com/receipts/receipts/internal/verdict"
)

func hermetic(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "xdg"))
}

func initGit(t *testing.T, dir string) {
	t.Helper()
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
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi\n"), 0644)
	run("add", ".")
	run("commit", "-m", "init")
}

func loadLatest(t *testing.T, workdir string) model.Receipt {
	t.Helper()
	st := store.New(filepath.Join(workdir, ".receipts"))
	got, err := st.Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	return got
}

func TestAuditNonGit(t *testing.T) {
	hermetic(t)
	dir := t.TempDir()
	var out, errBuf bytes.Buffer
	if code := runAudit([]string{"--workdir", dir}, &out, &errBuf); code != 0 {
		t.Fatalf("exit = %d, stderr %q", code, errBuf.String())
	}
	got := loadLatest(t, dir)
	if got.Repo.NonGit == nil || !*got.Repo.NonGit {
		t.Errorf("expected NonGit repo")
	}
	if len(got.Claims) != 0 {
		t.Errorf("expected zero claims, got %d", len(got.Claims))
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
	if err := model.ValidateReceipt(got); err != nil {
		t.Errorf("stored receipt invalid: %v", err)
	}
	if err := verdict.CheckInvariants(&got); err != nil {
		t.Errorf("invariants: %v", err)
	}
	if !strings.Contains(out.String(), "RECEIPT") {
		t.Errorf("terminal output should render receipt, got:\n%s", out.String())
	}
}

func TestAuditGitNoTranscript(t *testing.T) {
	hermetic(t)
	dir := t.TempDir()
	initGit(t, dir)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("new\n"), 0644)
	var out, errBuf bytes.Buffer
	if code := runAudit([]string{"--workdir", dir}, &out, &errBuf); code != 0 {
		t.Fatalf("exit = %d, stderr %q", code, errBuf.String())
	}
	got := loadLatest(t, dir)
	if len(got.Claims) != 0 {
		t.Errorf("expected zero claims without transcript, got %d", len(got.Claims))
	}
	if len(got.Evidence) < 3 {
		t.Errorf("expected git_snapshot + diff + adapter_status, got %d", len(got.Evidence))
	}
	found := false
	for _, w := range got.Warnings {
		if strings.Contains(w, "adapter issue") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected adapter issue warning, got %v", got.Warnings)
	}
	if got.Tier != model.Tier0 {
		t.Errorf("tier = %d, want 0", got.Tier)
	}
	if err := model.ValidateReceipt(got); err != nil {
		t.Errorf("invalid: %v", err)
	}
	if err := verdict.CheckInvariants(&got); err != nil {
		t.Errorf("invariants: %v", err)
	}
}

func TestAuditBadWorkdir(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := runAudit([]string{"--workdir", filepath.Join(t.TempDir(), "missing")}, &out, &errBuf); code != 3 {
		t.Errorf("exit = %d, want 3", code)
	}
}

func TestAuditBadAgent(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := runAudit([]string{"--agent", "codex"}, &out, &errBuf); code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
}

func TestAuditJSONOut(t *testing.T) {
	hermetic(t)
	dir := t.TempDir()
	initGit(t, dir)
	outPath := filepath.Join(dir, "r.json")
	var out, errBuf bytes.Buffer
	if code := runAudit([]string{"--workdir", dir, "--format", "json", "--out", outPath}, &out, &errBuf); code != 0 {
		t.Fatalf("exit = %d, stderr %q", code, errBuf.String())
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read out: %v", err)
	}
	if !strings.Contains(string(data), `"schema_version":"1.0.0"`) {
		t.Errorf("json output should be canonical receipt, got:\n%s", data)
	}
}
