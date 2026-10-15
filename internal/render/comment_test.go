package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommentBody(t *testing.T) {
	r := CommentRenderer{}
	out, err := r.Render(fixedReceipt(), Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	s := string(out)
	for _, want := range []string{
		"<!-- receipts-report -->",
		"RC-4F2A",
		"Tier: 0",
		"low",
		"1 verified",
		"TEST_ASSERTIONS_REDUCED",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("comment missing %q:\n%s", want, s)
		}
	}
}

func TestCommentNoSecretsOrLinks(t *testing.T) {
	r := CommentRenderer{}
	rec := fixedReceipt()
	rec.Warnings = []string{"token AKIAIOSFODNN7EXAMPLE leaked"}
	out, err := r.Render(rec, Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	s := string(out)
	if strings.Contains(s, "AKIAIOSFODNN7EXAMPLE") {
		t.Errorf("comment must not carry raw secrets")
	}
	if strings.Contains(lower(s), "http://") || strings.Contains(lower(s), "https://") {
		t.Errorf("comment must not invent links without a base URL")
	}
}

func TestCommentBaseURLLink(t *testing.T) {
	r := CommentRenderer{}
	out, err := r.Render(fixedReceipt(), Options{BaseURL: "https://receipts.example.com/r"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), "https://receipts.example.com/r") {
		t.Errorf("configured base URL should link the receipt")
	}
}

func lower(s string) string { return strings.ToLower(s) }

func TestCommentGolden(t *testing.T) {
	r := CommentRenderer{}
	out, err := r.Render(fixedReceipt(), Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	golden, err := os.ReadFile(filepath.Join("..", "..", "testdata", "golden", "ci-comment.md"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if string(out) != string(golden) {
		t.Errorf("comment differs from golden.\nGot:\n%s\nWant:\n%s", out, golden)
	}
}
