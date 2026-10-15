package adapters

import (
	"os"
	"strings"
	"testing"
)

func TestCheckFileSize(t *testing.T) {
	dir := t.TempDir()
	small := dir + "/small.jsonl"
	if err := os.WriteFile(small, []byte("{}\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := checkFileSizeLimit(small, 1024); err != nil {
		t.Errorf("small file should pass, got %v", err)
	}
	if err := checkFileSizeLimit(small, 2); err == nil {
		t.Errorf("expected oversize error, got nil")
	} else if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("error should say exceeds, got %q", err.Error())
	}
	if err := CheckFileSize(dir + "/missing.jsonl"); err == nil {
		t.Errorf("expected error for missing file, got nil")
	}
}
