package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckLatest(t *testing.T) {
	hermetic(t)
	dir := t.TempDir()
	initGit(t, dir)
	var out, errBuf bytes.Buffer
	if code := runAudit([]string{"--workdir", dir}, &out, &errBuf); code != 0 {
		t.Fatalf("audit exit = %d", code)
	}
	out.Reset()
	errBuf.Reset()
	if code := runCheck([]string{"--workdir", dir}, &out, &errBuf); code != 0 {
		t.Fatalf("check exit = %d, stderr %q", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "RECEIPT") {
		t.Errorf("check should render receipt, got:\n%s", out.String())
	}
}

func TestCheckUnknownID(t *testing.T) {
	hermetic(t)
	var out, errBuf bytes.Buffer
	if code := runCheck([]string{"RC-FFFF", "--workdir", t.TempDir()}, &out, &errBuf); code != 4 {
		t.Errorf("exit = %d, want 4", code)
	}
}

func TestCheckStaleWarning(t *testing.T) {
	hermetic(t)
	dir := t.TempDir()
	initGit(t, dir)
	var out, errBuf bytes.Buffer
	if code := runAudit([]string{"--workdir", dir}, &out, &errBuf); code != 0 {
		t.Fatalf("audit exit = %d", code)
	}
	cmd := exec.Command("git", "-C", dir, "commit", "--allow-empty", "-m", "drift")
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@e.c", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@e.c")
	if data, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit: %v\n%s", err, data)
	}
	out.Reset()
	errBuf.Reset()
	if code := runCheck([]string{"--workdir", dir}, &out, &errBuf); code != 0 {
		t.Fatalf("check exit = %d", code)
	}
	if !strings.Contains(errBuf.String(), "stale") {
		t.Errorf("expected stale warning, stderr %q", errBuf.String())
	}
}

func TestCheckReverifyTrue(t *testing.T) {
	hermetic(t)
	dir := t.TempDir()
	initGit(t, dir)
	var out, errBuf bytes.Buffer
	if code := runAudit([]string{"--workdir", dir}, &out, &errBuf); code != 0 {
		t.Fatalf("audit exit = %d", code)
	}
	out.Reset()
	errBuf.Reset()
	if code := runCheck([]string{"--workdir", dir, "--reverify", "--command", "true"}, &out, &errBuf); code != 0 {
		t.Fatalf("check exit = %d, stderr %q", code, errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "Re-verified:") {
		t.Errorf("expected Re-verified line, stderr %q", errBuf.String())
	}
}

func TestCheckReverifyDirtyRefuses(t *testing.T) {
	hermetic(t)
	dir := t.TempDir()
	initGit(t, dir)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("dirty\n"), 0644)
	var out, errBuf bytes.Buffer
	if code := runAudit([]string{"--workdir", dir}, &out, &errBuf); code != 0 {
		t.Fatalf("audit exit = %d", code)
	}
	out.Reset()
	errBuf.Reset()
	if code := runCheck([]string{"--workdir", dir, "--reverify", "--command", "true"}, &out, &errBuf); code != 5 {
		t.Errorf("dirty without force exit = %d, want 5", code)
	}
	out.Reset()
	errBuf.Reset()
	if code := runCheck([]string{"--workdir", dir, "--reverify", "--command", "true", "--force"}, &out, &errBuf); code != 0 {
		t.Errorf("force exit = %d, stderr %q", code, errBuf.String())
	}
}

func TestCheckReverifyNeedsCommand(t *testing.T) {
	hermetic(t)
	dir := t.TempDir()
	initGit(t, dir)
	var out, errBuf bytes.Buffer
	if code := runAudit([]string{"--workdir", dir}, &out, &errBuf); code != 0 {
		t.Fatalf("audit exit = %d", code)
	}
	out.Reset()
	errBuf.Reset()
	if code := runCheck([]string{"--workdir", dir, "--reverify"}, &out, &errBuf); code != 2 {
		t.Errorf("exit = %d, want 2 without --command", code)
	}
}

func TestCheckViaDispatch(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := Run([]string{"check", "--help"}, &out, &errBuf); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "Usage") {
		t.Errorf("help should print usage")
	}
}
