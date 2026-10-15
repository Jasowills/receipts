package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitFresh(t *testing.T) {
	dir := t.TempDir()
	var out, errBuf bytes.Buffer
	if code := runInit([]string{"--workdir", dir}, &out, &errBuf); code != 0 {
		t.Fatalf("exit = %d, stderr %q", code, errBuf.String())
	}
	cfg, err := os.ReadFile(filepath.Join(dir, "receipts.toml"))
	if err != nil {
		t.Fatalf("config not written: %v", err)
	}
	if !strings.Contains(string(cfg), `agent = "auto"`) {
		t.Errorf("starter config missing agent key")
	}
	ig, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil || !hasIgnoreEntry(string(ig)) {
		t.Errorf("gitignore entry missing: %v %q", err, ig)
	}
}

func TestInitRefusesWithoutForce(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "receipts.toml"), []byte("custom = 1\n"), 0644)
	before, _ := os.ReadFile(filepath.Join(dir, "receipts.toml"))
	var out, errBuf bytes.Buffer
	if code := runInit([]string{"--workdir", dir}, &out, &errBuf); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "receipts.toml"))
	if string(after) != string(before) {
		t.Errorf("existing config must not be overwritten without --force")
	}
}

func TestInitForceBacksUp(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "receipts.toml"), []byte("custom = 1\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/\n"), 0644)
	var out, errBuf bytes.Buffer
	if code := runInit([]string{"--workdir", dir, "--force"}, &out, &errBuf); code != 0 {
		t.Fatalf("exit = %d, stderr %q", code, errBuf.String())
	}
	bak, err := os.ReadFile(filepath.Join(dir, "receipts.toml.bak"))
	if err != nil || string(bak) != "custom = 1\n" {
		t.Errorf("config backup missing or wrong: %v %q", err, bak)
	}
	igbak, err := os.ReadFile(filepath.Join(dir, ".gitignore.bak"))
	if err != nil || string(igbak) != "node_modules/\n" {
		t.Errorf("gitignore backup missing or wrong: %v %q", err, igbak)
	}
	ig, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if !strings.Contains(string(ig), "node_modules/") || !hasIgnoreEntry(string(ig)) {
		t.Errorf("gitignore must keep original lines plus entry, got %q", ig)
	}
}

func TestInitKeepsExistingEntry(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".receipts/\n"), 0644)
	var out, errBuf bytes.Buffer
	if code := runInit([]string{"--workdir", dir}, &out, &errBuf); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	ig, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if strings.Count(string(ig), ".receipts/") != 1 {
		t.Errorf("entry must not duplicate, got %q", ig)
	}
}

func TestInitViaRunDispatch(t *testing.T) {
	dir := t.TempDir()
	var out, errBuf bytes.Buffer
	if code := Run([]string{"init", "--workdir", dir}, &out, &errBuf); code != 0 {
		t.Fatalf("Run init exit = %d, stderr %q", code, errBuf.String())
	}
}
