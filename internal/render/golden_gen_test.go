package render

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteGoldens(t *testing.T) {
	if os.Getenv("UPDATE_GOLDENS") != "1" {
		t.Skip("set UPDATE_GOLDENS=1 to write goldens")
	}
	term, err := TerminalRenderer{}.Render(fixedReceipt(), Options{Color: false})
	if err != nil {
		t.Fatalf("terminal render: %v", err)
	}
	if err := os.WriteFile(filepath.Join("..", "..", "testdata", "golden", "show-terminal.txt"), term, 0644); err != nil {
		t.Fatalf("write terminal golden: %v", err)
	}
	md, err := MarkdownRenderer{}.Render(fixedReceipt(), Options{})
	if err != nil {
		t.Fatalf("markdown render: %v", err)
	}
	if err := os.WriteFile(filepath.Join("..", "..", "testdata", "golden", "show-markdown.md"), md, 0644); err != nil {
		t.Fatalf("write markdown golden: %v", err)
	}
}
