package claims

import (
	"testing"

	"github.com/receipts/receipts/internal/model"
)

func TestMergeDedupsAndAppends(t *testing.T) {
	base := []model.Claim{
		{ID: "C-001", Type: model.ClaimTestsPass, Text: "All tests pass"},
		{ID: "C-002", Type: model.ClaimFilesChanged, Text: "I only changed src/a.txt"},
	}
	extra := []model.Claim{
		{ID: "C-001", Type: model.ClaimTestsPass, Text: "all tests pass!"},
		{ID: "C-002", Type: model.ClaimLintClean, Text: "Lint is clean"},
	}
	merged, warnings := MergeAdditional(base, extra)
	if len(merged) != 3 {
		t.Fatalf("merged = %d claims, want 3", len(merged))
	}
	if merged[0].ID != "C-001" || merged[2].ID != "C-003" || merged[2].Type != model.ClaimLintClean {
		t.Errorf("ids/types wrong: %+v", merged)
	}
	if len(warnings) != 0 {
		t.Errorf("no new conflicts expected, got %+v", warnings)
	}
}

func TestMergeConflictOnlyNewPairs(t *testing.T) {
	base := []model.Claim{
		{ID: "C-001", Type: model.ClaimFilesChanged, Text: "I only changed src/a.txt"},
	}
	extra := []model.Claim{
		{ID: "C-001", Type: model.ClaimFilesChanged, Text: "I only changed src/b.txt"},
	}
	merged, warnings := MergeAdditional(base, extra)
	if len(merged) != 2 {
		t.Fatalf("both file claims persist, got %+v", merged)
	}
	if len(warnings) != 1 || warnings[0].Code != model.ExtractionConflictAgents {
		t.Fatalf("expected one conflict warning, got %+v", warnings)
	}
}
