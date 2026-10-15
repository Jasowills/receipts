package claims

import (
	"strings"
	"testing"

	"github.com/receipts/receipts/internal/adapters"
	"github.com/receipts/receipts/internal/model"
)

func TestSourceRefs(t *testing.T) {
	ex := DeterministicExtractor{}
	claims, _, err := ex.Extract(t.Context(), []adapters.Message{assistantMsg("m-42", "All tests pass")})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("got %+v", claims)
	}
	c := claims[0]
	if c.ID != "C-001" {
		t.Errorf("id = %q, want C-001", c.ID)
	}
	if c.Source.MessageID != "m-42" || c.Source.Role != "assistant" {
		t.Errorf("source = %+v", c.Source)
	}
	if len(c.Source.ExcerptHash) != 64 {
		t.Errorf("excerpt hash = %q, want 64 hex chars", c.Source.ExcerptHash)
	}
	if c.Verdict != model.VerdictUnverified {
		t.Errorf("initial verdict = %q, want UNVERIFIED placeholder for the engine", c.Verdict)
	}
}

func TestDedupIdentical(t *testing.T) {
	ex := DeterministicExtractor{}
	claims, _, err := ex.Extract(t.Context(), []adapters.Message{
		assistantMsg("m-1", "All tests pass"),
		assistantMsg("m-2", "all tests pass!"),
	})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("identical repeats should dedup, got %+v", claims)
	}
	if claims[0].Source.MessageID != "m-1" {
		t.Errorf("earliest source should win, got %+v", claims[0].Source)
	}
}

func TestDistinctClaimsKept(t *testing.T) {
	ex := DeterministicExtractor{}
	claims, _, err := ex.Extract(t.Context(), []adapters.Message{
		assistantMsg("m-1", "All tests pass. Lint is clean."),
	})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(claims) != 2 || claims[0].ID != "C-001" || claims[1].ID != "C-002" {
		t.Fatalf("distinct claims should both persist with stable IDs, got %+v", claims)
	}
}

func TestConflictingFileClaims(t *testing.T) {
	ex := DeterministicExtractor{}
	claims, warnings, err := ex.Extract(t.Context(), []adapters.Message{
		assistantMsg("m-1", "I only changed src/auth.ts"),
		assistantMsg("m-2", "I only changed src/other.ts"),
	})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(claims) != 2 {
		t.Fatalf("conflicting claims must both persist, got %+v", claims)
	}
	found := false
	for _, w := range warnings {
		if w.Code == model.ExtractionConflictAgents &&
			strings.Contains(w.Message, "C-001") && strings.Contains(w.Message, "C-002") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected CONFLICTING_AGENT_CLAIMS warning, got %+v", warnings)
	}
}

func TestNoFalseConflict(t *testing.T) {
	ex := DeterministicExtractor{}
	_, warnings, err := ex.Extract(t.Context(), []adapters.Message{
		assistantMsg("m-1", "All tests pass"),
		assistantMsg("m-2", "Lint is clean"),
	})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	for _, w := range warnings {
		if w.Code == model.ExtractionConflictAgents {
			t.Errorf("unrelated claims must not conflict: %+v", warnings)
		}
	}
}

func TestPatternDebugRecorded(t *testing.T) {
	ex := DeterministicExtractor{}
	claims, _, err := ex.Extract(t.Context(), []adapters.Message{assistantMsg("m-1", "All tests pass")})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(ex.Seen) != len(claims) || ex.Seen[0] == "" {
		t.Errorf("pattern ids should align with claims, got %v", ex.Seen)
	}
}
