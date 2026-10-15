package claims

import (
	"testing"

	"github.com/receipts/receipts/internal/adapters"
	"github.com/receipts/receipts/internal/model"
	"github.com/receipts/receipts/internal/verdict"
)

// Adversarial tests for the authority principle: no wording, however
// persuasive, and no chat pasted output can manufacture a VERIFIED
// verdict. Only joined evidence decides.

func judge(t *testing.T, text string, evidences []model.Evidence, tier model.Tier) []model.Claim {
	t.Helper()
	ex := DeterministicExtractor{}
	claims, _, err := ex.Extract(t.Context(), []adapters.Message{{ID: "m-1", Role: adapters.RoleAssistant, Text: text}})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(claims) == 0 {
		t.Fatalf("no claim extracted from %q", text)
	}
	r := &model.Receipt{
		SchemaVersion: model.SchemaVersion, ID: "RC-0001",
		Agent: model.Agent{Name: "test"}, Tier: tier,
		Repo:     model.Repo{Root: ".", BaselineConfidence: model.BaselineHigh},
		Claims:   claims,
		Evidence: evidences,
		Chain:    model.Chain{Algorithm: model.ChainAlgorithmSHA256},
	}
	eng := verdict.DeterministicEngine{}
	got, _, err := eng.Evaluate(r)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	return got
}

func cmdArgv(argv []string, exit int) model.Evidence {
	args := make([]any, len(argv))
	for i, a := range argv {
		args[i] = a
	}
	return model.Evidence{ID: "E-001", Kind: model.EvidenceCommand,
		Data: map[string]any{"argv": args, "exit_code": float64(exit)}}
}

// An agent claims success despite a failing test. Wording strength is
// irrelevant; the failing run contradicts the claim.
func TestAdversarialFalseSuccessClaim(t *testing.T) {
	got := judge(t,
		"All tests pass. I verified thoroughly, everything is green and I take full responsibility for this result.",
		[]model.Evidence{cmdArgv([]string{"npm", "test"}, 1)},
		model.Tier1)
	if got[0].Verdict != model.VerdictContradicted {
		t.Errorf("verdict = %s, want CONTRADICTED", got[0].Verdict)
	}
}

// An agent claims a check it skipped. A test run never verifies lint.
func TestAdversarialSkippedCheck(t *testing.T) {
	got := judge(t,
		"Lint is clean. The full suite passed so quality gates are satisfied.",
		[]model.Evidence{cmdArgv([]string{"npm", "test"}, 0)},
		model.Tier1)
	if len(got) != 1 || got[0].Type != model.ClaimLintClean {
		t.Fatalf("want the lint claim, got %+v", got)
	}
	if got[0].Verdict != model.VerdictUnverified {
		t.Errorf("verdict = %s, want UNVERIFIED", got[0].Verdict)
	}
}

// Missing evidence can never become VERIFIED, for any claim type.
func TestAdversarialMissingEvidenceNeverVerifies(t *testing.T) {
	texts := map[model.ClaimType]string{
		model.ClaimTestsPass:      "All tests pass",
		model.ClaimBuildOK:        "The build passed",
		model.ClaimLintClean:      "Lint is clean",
		model.ClaimFilesChanged:   "I only changed src/a.txt",
		model.ClaimNoOtherChanges: "No other files were changed",
		model.ClaimDepAdded:       "Added lodash as a dependency",
	}
	for typ, text := range texts {
		got := judge(t, text, nil, model.Tier1)
		if len(got) != 1 || got[0].Type != typ {
			t.Fatalf("%s: got %+v", typ, got)
		}
		if got[0].Verdict == model.VerdictVerified {
			t.Errorf("%s: missing evidence must never verify", typ)
		}
	}
}

// Pasted output in chat is text, not a run. Transcript metadata alone
// verifies nothing.
func TestAdversarialPastedOutputProvesNothing(t *testing.T) {
	ev := model.Evidence{ID: "E-001", Kind: model.EvidenceTranscript,
		Data: map[string]any{"adapter": "claude", "session_id": "s", "message_count": 12}}
	got := judge(t,
		"All tests pass. Proof: ok example 0.1s, 3 passed, 0 failed.",
		[]model.Evidence{ev},
		model.Tier1)
	if got[0].Verdict != model.VerdictUnverified {
		t.Errorf("verdict = %s, want UNVERIFIED", got[0].Verdict)
	}
}
