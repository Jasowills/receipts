package render

import (
	"os"
	"path/filepath"
	"testing"
)

func TestZZWriteHTMLGolden(t *testing.T) {
	if os.Getenv("UPDATE_GOLDENS") != "1" {
		t.Skip("set UPDATE_GOLDENS=1")
	}
	out, err := HTMLRenderer{}.Render(fixedReceipt(), Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if err := os.WriteFile(filepath.Join("..", "..", "testdata", "golden", "show-html.html"), out, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
