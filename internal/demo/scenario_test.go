package demo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/receipts/receipts/internal/model"
	"github.com/receipts/receipts/internal/render"
)

func TestDefaultScenarioVerdicts(t *testing.T) {
	rec, dir, cleanup, err := Run("default", false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	cleanup()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("temp dir should be cleaned up")
	}
	byType := map[model.ClaimType]model.Verdict{}
	for _, c := range rec.Claims {
		byType[c.Type] = c.Verdict
	}
	if byType[model.ClaimTestsPass] != model.VerdictVerified {
		t.Errorf("TESTS_PASS = %s, want VERIFIED", byType[model.ClaimTestsPass])
	}
	if byType[model.ClaimBuildOK] != model.VerdictContradicted {
		t.Errorf("BUILD_OK = %s, want CONTRADICTED", byType[model.ClaimBuildOK])
	}
	if byType[model.ClaimLintClean] != model.VerdictUnverified {
		t.Errorf("LINT_CLEAN = %s, want UNVERIFIED", byType[model.ClaimLintClean])
	}
	if byType[model.ClaimFilesChanged] != model.VerdictContradicted {
		t.Errorf("FILES_CHANGED = %s, want CONTRADICTED (reliable demo baseline)", byType[model.ClaimFilesChanged])
	}
	found := map[string]bool{}
	for _, f := range rec.Flags {
		found[f.Code] = true
	}
	for _, want := range []string{"TEST_FILE_MODIFIED", "ASSERTION_TOKEN_COUNT_REDUCED", "SKIP_ADDED", "CI_CONFIG_CHANGED"} {
		if !found[want] {
			t.Errorf("missing flag %s, got %+v", want, rec.Flags)
		}
	}
	if rec.ID != "RC-DE07" {
		t.Errorf("demo id = %q, want deterministic RC-DE07", rec.ID)
	}
}

func TestDefaultGolden(t *testing.T) {
	rec, _, cleanup, err := Run("default", false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	defer cleanup()
	out, err := render.TerminalRenderer{}.Render(rec, render.Options{})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// The temp repo root varies; goldens normalize it out.
	normalized := strings.ReplaceAll(string(out), rec.Repo.Root, ".")
	golden, err := os.ReadFile(filepath.Join("..", "..", "testdata", "golden", "demo-terminal.txt"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if normalized != string(golden) {
		t.Errorf("demo output differs from golden.\nGot:\n%s\nWant:\n%s", normalized, golden)
	}
}

func TestScopeDriftScenario(t *testing.T) {
	rec, _, cleanup, err := Run("scope-drift", false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	defer cleanup()
	if len(rec.Claims) != 1 || rec.Claims[0].Type != model.ClaimFilesChanged {
		t.Fatalf("claims = %+v", rec.Claims)
	}
	if rec.Claims[0].Verdict != model.VerdictContradicted {
		t.Errorf("scope drift claim = %s, want CONTRADICTED", rec.Claims[0].Verdict)
	}
}

func TestDuplicateClaimsScenario(t *testing.T) {
	rec, _, cleanup, err := Run("duplicate-claims", false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	defer cleanup()
	if len(rec.Claims) != 1 {
		t.Errorf("duplicates should dedup to 1, got %+v", rec.Claims)
	}
}

func TestWeakeningScenario(t *testing.T) {
	rec, _, cleanup, err := Run("test-weakening", false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	defer cleanup()
	codes := map[string]bool{}
	for _, f := range rec.Flags {
		codes[f.Code] = true
	}
	if !codes["ASSERTION_TOKEN_COUNT_REDUCED"] || !codes["SKIP_ADDED"] {
		t.Errorf("weakening flags missing: %+v", rec.Flags)
	}
}

func TestUnknownScenario(t *testing.T) {
	if _, _, _, err := Run("nope", false); err == nil {
		t.Errorf("expected error for unknown scenario")
	}
}

func TestKeepLeavesDir(t *testing.T) {
	_, dir, _, err := Run("default", true)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("keep should leave dir: %v", err)
	}
	os.RemoveAll(dir)
}
