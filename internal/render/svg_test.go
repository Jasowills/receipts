package render

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSVGMatchesGolden(t *testing.T) {
	r := SVGRenderer{}
	out, err := r.Render(fixedReceipt(), Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	golden, err := os.ReadFile(filepath.Join("..", "..", "testdata", "golden", "badge.svg"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if string(out) != string(golden) {
		t.Errorf("svg differs from golden.\nGot:\n%s\nWant:\n%s", out, golden)
	}
}

func TestSVGNoScriptsOrLinks(t *testing.T) {
	r := SVGRenderer{}
	out, err := r.Render(fixedReceipt(), Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	lower := strings.ToLower(string(out))
	for _, bad := range []string{"<script", "javascript:", "<a ", "href=", "xlink", "onload=", "onerror="} {
		if strings.Contains(lower, bad) {
			t.Errorf("forbidden svg construct %q present", bad)
		}
	}
	// The xmlns namespace name is required and inert; it must be the
	// only URL like string in the document.
	if n := strings.Count(string(out), "http://"); n != 1 {
		t.Errorf("want exactly the xmlns URL, found %d http references", n)
	}
	var v any
	if err := xml.Unmarshal(out, &v); err != nil {
		t.Errorf("svg must be valid XML: %v", err)
	}
}

func TestSVGStatusColors(t *testing.T) {
	r := SVGRenderer{}
	rec := fixedReceipt()
	rec.Summary.Contradicted = 2
	out, err := r.Render(rec, Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(strings.ToLower(string(out)), "#b00020") {
		t.Errorf("contradictions should color the badge red")
	}
	rec.Summary.Contradicted = 0
	rec.Summary.Verified = 0
	out, err = r.Render(rec, Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(strings.ToLower(string(out)), "#ba7517") {
		t.Errorf("unverified only should color the badge amber")
	}
}

func TestSVGDeterministic(t *testing.T) {
	r := SVGRenderer{}
	a, err := r.Render(fixedReceipt(), Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	b, err := r.Render(fixedReceipt(), Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if string(a) != string(b) {
		t.Errorf("svg render not deterministic")
	}
}
