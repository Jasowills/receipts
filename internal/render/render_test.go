package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/receipts/receipts/internal/model"
)

func fixedReceipt() model.Receipt {
	ts := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	dirty := true
	msgTs := time.Date(2026, 10, 14, 12, 1, 0, 0, time.UTC)
	return model.Receipt{
		SchemaVersion: model.SchemaVersion,
		ID:            "RC-4F2A",
		CreatedAt:     ts,
		Agent:         model.Agent{Name: "opencode", SessionID: "session-example", AdapterVersion: "1"},
		Tier:          model.Tier0,
		DurationMS:    840000,
		Repo: model.Repo{
			Root:               ".",
			HeadBefore:         "abc123",
			HeadAfter:          "abc123",
			BranchBefore:       "main",
			BranchAfter:        "main",
			BaselineConfidence: model.BaselineLow,
			DirtyBefore:        model.DirtyUnknown,
			DirtyAfter:         &dirty,
		},
		Claims: []model.Claim{
			{ID: "C-001", Type: model.ClaimTestsPass, Text: "all tests pass", Source: model.Source{MessageID: "m-1", Role: "assistant", Timestamp: &msgTs}, Verdict: model.VerdictUnverified, EvidenceRefs: []string{}, Reason: "No command evidence exists in audit mode."},
			{ID: "C-002", Type: model.ClaimFilesChanged, Text: "only touched src/auth", Source: model.Source{MessageID: "m-2", Role: "assistant"}, Verdict: model.VerdictVerified, EvidenceRefs: []string{"E-001"}, Reason: "Observed changed paths match the stated list."},
		},
		Flags: []model.Flag{
			{ID: "F-001", Code: "TEST_ASSERTIONS_REDUCED", Severity: model.SeverityHigh, Title: "Assertion tokens reduced", Details: "tests/auth.test.ts: 8 -> 5 assertion tokens", Paths: []string{"tests/auth.test.ts"}, EvidenceRefs: []string{"E-002"}},
		},
		Evidence: []model.Evidence{
			{ID: "E-001", Kind: model.EvidenceDiff, Timestamp: ts, Data: map[string]any{"paths": "src/auth"}},
			{ID: "E-002", Kind: model.EvidenceDiff, Timestamp: ts, Data: map[string]any{"paths": "tests/auth.test.ts"}},
		},
		Warnings: []string{"adapter issue: no transcript detected"},
		Summary:  model.Summary{Verified: 1, Contradicted: 0, Unverified: 1, Unverifiable: 0, Flags: 1},
		Chain:    model.Chain{EventHash: "abc123", PreviousHash: "0000000000000000000000000000000000000000000000000000000000000000", Algorithm: model.ChainAlgorithmSHA256},
	}
}

func TestTerminalMatchesGolden(t *testing.T) {
	r := TerminalRenderer{}
	out, err := r.Render(fixedReceipt(), Options{Color: false})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	golden, err := os.ReadFile(filepath.Join("..", "..", "testdata", "golden", "show-terminal.txt"))
	if err != nil {
		t.Fatalf("read golden failed: %v", err)
	}
	if string(out) != string(golden) {
		t.Errorf("terminal output differs from golden.\nGot:\n%s\nWant:\n%s", out, golden)
	}
}

func TestTerminalColorSuppression(t *testing.T) {
	r := TerminalRenderer{}
	plain, err := r.Render(fixedReceipt(), Options{Color: false})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if strings.Contains(string(plain), "\x1b[") {
		t.Errorf("Color false output must not contain ANSI codes")
	}
	colored, err := r.Render(fixedReceipt(), Options{Color: true})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if !strings.Contains(string(colored), "\x1b[") {
		t.Errorf("Color true output should contain ANSI codes")
	}
	t.Setenv("NO_COLOR", "1")
	suppressed, err := r.Render(fixedReceipt(), Options{Color: true})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if strings.Contains(string(suppressed), "\x1b[") {
		t.Errorf("NO_COLOR must suppress ANSI codes even when Color true")
	}
}

func TestTerminalDeterministic(t *testing.T) {
	r := TerminalRenderer{}
	a, err := r.Render(fixedReceipt(), Options{Color: false})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	b, err := r.Render(fixedReceipt(), Options{Color: false})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if string(a) != string(b) {
		t.Errorf("terminal render not deterministic")
	}
}

func TestMarkdownMatchesGolden(t *testing.T) {
	r := MarkdownRenderer{}
	out, err := r.Render(fixedReceipt(), Options{})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	golden, err := os.ReadFile(filepath.Join("..", "..", "testdata", "golden", "show-markdown.md"))
	if err != nil {
		t.Fatalf("read golden failed: %v", err)
	}
	if string(out) != string(golden) {
		t.Errorf("markdown output differs from golden.\nGot:\n%s\nWant:\n%s", out, golden)
	}
}

func TestMarkdownEscapesUserText(t *testing.T) {
	r := MarkdownRenderer{}
	rec := fixedReceipt()
	rec.Claims[0].Text = "fix `code` and *stars* [link]"
	out, err := r.Render(rec, Options{})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "\\`code\\`") {
		t.Errorf("backticks in user text must be escaped as \\`code\\`, got:\n%s", s)
	}
	if !strings.Contains(s, "\\*stars\\*") {
		t.Errorf("asterisks in user text must be escaped as \\*stars\\*, got:\n%s", s)
	}
}
