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
)

func buildReceipts(t *testing.T) string {
	t.Helper()
	bin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain required to build the shim host binary")
	}
	_ = bin
	out := filepath.Join(t.TempDir(), "receipts")
	cmd := exec.Command("go", "build", "-o", out, "./cmd/receipts")
	cmd.Dir = filepath.Join("..", "..")
	if data, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build receipts: %v\n%s", err, data)
	}
	return out
}

func hermeticRun(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "xdg"))
}

func writeAgent(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, "agent.sh")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body), 0755); err != nil {
		t.Fatalf("write agent: %v", err)
	}
	return p
}

func TestRunExitCode(t *testing.T) {
	hermeticRun(t)
	dir := t.TempDir()
	var out, errBuf bytes.Buffer
	code := runRun([]string{"--workdir", dir, "--", "sh", "-c", "exit 3"}, &out, &errBuf)
	if code != 3 {
		t.Errorf("exit = %d, want wrapped 3", code)
	}
	st := store.New(filepath.Join(dir, ".receipts"))
	got, err := st.Latest(context.Background())
	if err != nil {
		t.Fatalf("receipt should still be stored: %v", err)
	}
	if got.Tier != model.Tier1 {
		t.Errorf("tier = %d, want 1 (shims active)", got.Tier)
	}
}

func TestRunMissingCommand(t *testing.T) {
	hermeticRun(t)
	var out, errBuf bytes.Buffer
	if code := runRun([]string{"--workdir", t.TempDir(), "--", "no-such-cmd-xyz"}, &out, &errBuf); code != 3 {
		t.Errorf("exit = %d, want 3 for launch failure", code)
	}
}

func TestRunNoArgs(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := runRun([]string{}, &out, &errBuf); code != 2 {
		t.Errorf("exit = %d, want 2", code)
	}
}

func TestRunCapturesShimCommand(t *testing.T) {
	hermeticRun(t)
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go required as the wrapped command")
	}
	binary := buildReceipts(t)
	dir := t.TempDir()
	agent := writeAgent(t, dir, "go version\n")
	cmd := exec.Command(binary, "run", "--workdir", dir, "--", "sh", "-c", agent)
	cmd.Env = os.Environ()
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v\n%s", err, errBuf.String())
	}
	st := store.New(filepath.Join(dir, ".receipts"))
	got, err := st.Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if got.Tier != model.Tier1 {
		t.Errorf("tier = %d, want 1", got.Tier)
	}
	found := false
	for _, e := range got.Evidence {
		if e.Kind != model.EvidenceCommand {
			continue
		}
		if argv, ok := e.Data["argv"].([]any); ok && len(argv) > 0 && argv[0] == "go" {
			found = true
		}
	}
	if !found {
		t.Errorf("no go command evidence; re-run with --keep to inspect")
	}
	warns := strings.Join(got.Warnings, "\n")
	if !strings.Contains(warns, "command capture may be incomplete") {
		t.Errorf("capture disclosure missing: %v", got.Warnings)
	}
}

func TestRunPathPropagation(t *testing.T) {
	hermeticRun(t)
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go required")
	}
	binary := buildReceipts(t)
	dir := t.TempDir()
	probe := filepath.Join(dir, "probe.txt")
	agent := writeAgent(t, dir, "echo \"$PATH\" > "+probe+"\ncommand -v npm >> "+probe+"\n")
	cmd := exec.Command(binary, "run", "--workdir", dir, "--", "sh", agent)
	cmd.Env = os.Environ()
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v\n%s", err, errBuf.String())
	}
	data, err := os.ReadFile(probe)
	if err != nil {
		t.Fatalf("read probe: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("probe = %q", data)
	}
	entries := strings.Split(lines[0], ":")
	if !strings.Contains(entries[0], ".receipts"+string(filepath.Separator)+"runs") || !strings.HasSuffix(entries[0], "bin") {
		t.Errorf("shim bin dir must lead child PATH, got %q", lines[0])
	}
	if !strings.HasSuffix(lines[1], string(filepath.Separator)+"bin"+string(filepath.Separator)+"npm") && !strings.HasSuffix(lines[1], "/bin/npm") {
		t.Errorf("npm should resolve to the shim, got %q", lines[1])
	}
}

func TestRunNoShimsTier0(t *testing.T) {
	hermeticRun(t)
	dir := t.TempDir()
	var out, errBuf bytes.Buffer
	code := runRun([]string{"--workdir", dir, "--no-shims", "--quiet", "--", "sh", "-c", "true"}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	st := store.New(filepath.Join(dir, ".receipts"))
	got, err := st.Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if got.Tier != model.Tier0 {
		t.Errorf("tier = %d, want 0 without shims", got.Tier)
	}
	found := false
	for _, f := range got.Flags {
		if f.Code == "COMMAND_CAPTURE_INCOMPLETE" {
			found = true
		}
	}
	if !found {
		t.Errorf("capture incomplete flag missing: %+v", got.Flags)
	}
}

func TestRunViaDispatch(t *testing.T) {
	hermeticRun(t)
	var out, errBuf bytes.Buffer
	if code := Run([]string{"run", "--help"}, &out, &errBuf); code != 0 {
		t.Fatalf("run --help exit = %d", code)
	}
	if !strings.Contains(out.String(), "Usage") {
		t.Errorf("help should print usage")
	}
}
