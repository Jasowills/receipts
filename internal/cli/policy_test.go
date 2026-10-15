package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuditInvalidConfig(t *testing.T) {
	hermetic(t)
	dir := t.TempDir()
	initGit(t, dir)
	os.WriteFile(filepath.Join(dir, "receipts.toml"), []byte("agent = \"hal\"\n"), 0644)
	var out, errBuf bytes.Buffer
	if code := runAudit([]string{"--workdir", dir}, &out, &errBuf); code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
	if !strings.Contains(errBuf.String(), "agent:") {
		t.Errorf("error should name the field, got %q", errBuf.String())
	}
}

func TestAuditScopePolicy(t *testing.T) {
	hermetic(t)
	dir := t.TempDir()
	initGit(t, dir)
	os.WriteFile(filepath.Join(dir, "docs-note.md"), []byte("new\n"), 0644)
	os.WriteFile(filepath.Join(dir, "receipts.toml"), []byte("[scope]\nallowed_paths = [\"src/**\"]\n"), 0644)
	var out, errBuf bytes.Buffer
	if code := runAudit([]string{"--workdir", dir}, &out, &errBuf); code != 0 {
		t.Fatalf("exit = %d, stderr %q", code, errBuf.String())
	}
	s := out.String()
	if !strings.Contains(s, "OUT_OF_SCOPE_CHANGE") {
		t.Errorf("scoped file should flag, got:\n%s", s)
	}
	if !strings.Contains(s, "policy: scope") {
		t.Errorf("policy note missing, got:\n%s", s)
	}
}

func TestAuditDisabledCheck(t *testing.T) {
	hermetic(t)
	dir := t.TempDir()
	initGit(t, dir)
	os.WriteFile(filepath.Join(dir, "a.test.js"), []byte("expect(x)\n"), 0644)
	os.WriteFile(filepath.Join(dir, "receipts.toml"), []byte("[checks]\ntest_files_changed = false\n"), 0644)
	var out, errBuf bytes.Buffer
	if code := runAudit([]string{"--workdir", dir}, &out, &errBuf); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if strings.Contains(out.String(), "TEST_FILE_MODIFIED") {
		t.Errorf("disabled check must not run")
	}
	if !strings.Contains(out.String(), "policy: disabled checks: test_files_changed") {
		t.Errorf("policy note missing, got:\n%s", out.String())
	}
}

func TestAuditLLMEnabledWarns(t *testing.T) {
	hermetic(t)
	dir := t.TempDir()
	initGit(t, dir)
	home, _ := os.LookupEnv("HOME")
	slug := strings.ReplaceAll(dir, "/", "-")
	proj := filepath.Join(home, ".claude", "projects", slug)
	os.MkdirAll(proj, 0755)
	os.WriteFile(filepath.Join(proj, "ses.jsonl"), []byte(
		"{\"type\":\"assistant\",\"message\":{\"role\":\"assistant\",\"content\":[{\"type\":\"text\",\"text\":\"All tests pass\"}]},\"uuid\":\"u-1\",\"timestamp\":\"2026-10-14T12:00:00Z\",\"cwd\":"+"\""+dir+"\"}\n"), 0644)
	os.WriteFile(filepath.Join(dir, "receipts.toml"), []byte("[llm]\nenabled = true\nendpoint = \"http://127.0.0.1:9\"\n"), 0644)
	var out, errBuf bytes.Buffer
	if code := runAudit([]string{"--workdir", dir}, &out, &errBuf); code != 0 {
		t.Fatalf("llm failure must degrade, exit = %d, stderr %q", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "LLM_EXTRACTION") {
		t.Errorf("llm issue should warn, got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "All tests pass") {
		t.Errorf("deterministic claims must stand, got:\n%s", out.String())
	}
}

func TestCheckReverifyConfigCommand(t *testing.T) {
	hermetic(t)
	dir := t.TempDir()
	initGit(t, dir)
	os.WriteFile(filepath.Join(dir, "receipts.toml"), []byte("[test]\ncommand = \"true\"\n"), 0644)
	commitAll(t, dir)
	var out, errBuf bytes.Buffer
	if code := runAudit([]string{"--workdir", dir}, &out, &errBuf); code != 0 {
		t.Fatalf("audit exit = %d", code)
	}
	out.Reset()
	errBuf.Reset()
	if code := runCheck([]string{"--workdir", dir, "--reverify"}, &out, &errBuf); code != 0 {
		t.Errorf("config command fallback exit = %d, stderr %q", code, errBuf.String())
	}
}

func commitAll(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{{"add", "."}, {"commit", "-m", "config"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@e.c", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@e.c")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}
