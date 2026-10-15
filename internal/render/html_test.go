package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/receipts/receipts/internal/model"
)

func TestHTMLMatchesGolden(t *testing.T) {
	r := HTMLRenderer{}
	out, err := r.Render(fixedReceipt(), Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	golden, err := os.ReadFile(filepath.Join("..", "..", "testdata", "golden", "show-html.html"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if string(out) != string(golden) {
		t.Errorf("html differs from golden.\nGot:\n%s\nWant:\n%s", out, golden)
	}
}

func TestHTMLEscapesTranscriptMarkup(t *testing.T) {
	r := HTMLRenderer{}
	rec := fixedReceipt()
	rec.Claims[0].Text = `<script>alert("x")</script> & "quoted"`
	rec.Claims[0].Reason = `<b onload="evil()">hi</b>`
	rec.Flags[0].Title = `"><img src=x onerror=y>`
	rec.Warnings = []string{`<iframe src="http://evil.example">`}
	out, err := r.Render(rec, Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	s := string(out)
	for _, raw := range []string{`<script`, `<img`, `<iframe`, `onload="`, `onerror="`, `javascript:`} {
		if strings.Contains(s, raw) {
			t.Errorf("unescaped markup %q in output", raw)
		}
	}
	if !strings.Contains(s, "&lt;script&gt;") {
		t.Errorf("expected escaped script tag")
	}
}

func TestHTMLHasNoScriptsOrRemoteAssets(t *testing.T) {
	r := HTMLRenderer{}
	out, err := r.Render(fixedReceipt(), Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	lower := strings.ToLower(string(out))
	for _, bad := range []string{"<script", "javascript:", "http://", "https://", "<link", "<iframe", "onload=", "onerror="} {
		if strings.Contains(lower, bad) {
			t.Errorf("forbidden html construct %q present", bad)
		}
	}
}

func TestHTMLDeterministic(t *testing.T) {
	r := HTMLRenderer{}
	a, err := r.Render(fixedReceipt(), Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	b, err := r.Render(fixedReceipt(), Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if string(a) != string(b) {
		t.Errorf("html render not deterministic")
	}
}

func TestHTMLUnknownVerdictSafe(t *testing.T) {
	r := HTMLRenderer{}
	rec := fixedReceipt()
	rec.Claims[0].Verdict = model.Verdict("BOGUS")
	out, err := r.Render(rec, Options{})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), "BOGUS") {
		t.Errorf("unknown verdicts must still render escaped")
	}
}
