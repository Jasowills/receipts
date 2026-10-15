package demo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/receipts/receipts/internal/render"
)

func TestWriteGolden(t *testing.T) {
	if os.Getenv("UPDATE_GOLDENS") != "1" {
		t.Skip("set UPDATE_GOLDENS=1 to write goldens")
	}
	rec, _, cleanup, err := Run("default", false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	defer cleanup()
	out, err := render.TerminalRenderer{}.Render(rec, render.Options{})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	normalized := strings.ReplaceAll(string(out), rec.Repo.Root, ".")
	if err := os.WriteFile(filepath.Join("..", "..", "testdata", "golden", "demo-terminal.txt"), []byte(normalized), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
