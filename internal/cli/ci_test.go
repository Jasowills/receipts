package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCiComment(t *testing.T) {
	hermetic(t)
	dir := t.TempDir()
	initGit(t, dir)
	var out, errBuf bytes.Buffer
	if code := runAudit([]string{"--workdir", dir}, &out, &errBuf); code != 0 {
		t.Fatalf("audit exit = %d", code)
	}
	out.Reset()
	errBuf.Reset()
	if code := runCi([]string{"--workdir", dir}, &out, &errBuf); code != 0 {
		t.Fatalf("ci exit = %d, stderr %q", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "<!-- receipts-report -->") {
		t.Errorf("comment must carry the update marker, got:\n%s", out.String())
	}
}

func TestCiBadFormat(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := runCi([]string{"--format", "pdf"}, &out, &errBuf); code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
}

func TestCiMissingReceipt(t *testing.T) {
	hermetic(t)
	var out, errBuf bytes.Buffer
	if code := runCi([]string{"--workdir", t.TempDir()}, &out, &errBuf); code != 4 {
		t.Errorf("exit = %d, want 4", code)
	}
}

func TestCiViaDispatch(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := Run([]string{"ci", "--help"}, &out, &errBuf); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "Usage") {
		t.Errorf("help should print usage")
	}
}
