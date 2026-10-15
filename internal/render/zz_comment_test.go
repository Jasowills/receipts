package render

import (
	"os"
	"path/filepath"
	"testing"
)

func TestZZWriteCommentGolden(t *testing.T) {
	if os.Getenv("UPDATE_GOLDENS") != "1" {
		t.Skip("set UPDATE_GOLDENS=1")
	}
	out, err := CommentRenderer{}.Render(fixedReceipt(), Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if err := os.WriteFile(filepath.Join("..", "..", "testdata", "golden", "ci-comment.md"), out, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
