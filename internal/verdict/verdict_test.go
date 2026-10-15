package verdict

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/receipts/receipts/internal/model"
)

var ts = time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

func mkClaim(id string, typ model.ClaimType, text string) model.Claim {
	return model.Claim{
		ID:   id,
		Type: typ,
		Text: text,
		Source: model.Source{
			MessageID: "m-1", Role: "assistant",
		},
		Verdict:      model.VerdictUnverified,
		EvidenceRefs: []string{},
		Reason:       "",
	}
}

func cmdEvidence(id string, argv []string, exitCode int) model.Evidence {
	args := make([]any, len(argv))
	for i, a := range argv {
		args[i] = a
	}
	return model.Evidence{
		ID:        id,
		Kind:      model.EvidenceCommand,
		Timestamp: ts,
		Data:      map[string]any{"argv": args, "exit_code": float64(exitCode)},
	}
}

func reverifyEvidence(id string, argv []string, exitCode int) model.Evidence {
	e := cmdEvidence(id, argv, exitCode)
	e.Kind = model.EvidenceReverify
	e.Data["checkout"] = "abc123"
	return e
}

func diffEvidence(id string, paths []string) model.Evidence {
	return model.Evidence{
		ID:        id,
		Kind:      model.EvidenceDiff,
		Timestamp: ts,
		Data:      map[string]any{"changed_paths": paths},
	}
}

func testResultEvidence(id string, passed bool) model.Evidence {
	return model.Evidence{
		ID:        id,
		Kind:      model.EvidenceTestResult,
		Timestamp: ts,
		Data:      map[string]any{"passed": passed},
	}
}

func testReceipt(tier model.Tier, conf model.BaselineConfidence) *model.Receipt {
	return &model.Receipt{
		SchemaVersion: model.SchemaVersion,
		ID:            "RC-0001",
		CreatedAt:     ts,
		Agent:         model.Agent{Name: "opencode"},
		Tier:          tier,
		Repo:          model.Repo{Root: ".", BaselineConfidence: conf},
		Claims:        []model.Claim{},
		Flags:         []model.Flag{},
		Evidence:      []model.Evidence{},
		Warnings:      []string{},
		Summary:       model.Summary{},
		Chain:         model.Chain{Algorithm: model.ChainAlgorithmSHA256},
	}
}

func evaluate(t *testing.T, r *model.Receipt) []model.Claim {
	t.Helper()
	eng := DeterministicEngine{}
	got, _, err := eng.Evaluate(r)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	return got
}

func TestTestsPassVerified(t *testing.T) {
	r := testReceipt(model.Tier1, model.BaselineHigh)
	r.Claims = []model.Claim{mkClaim("C-001", model.ClaimTestsPass, "all tests pass")}
	r.Evidence = []model.Evidence{cmdEvidence("E-001", []string{"npm", "test"}, 0)}
	got := evaluate(t, r)
	if got[0].Verdict != model.VerdictVerified {
		t.Errorf("verdict = %s, want VERIFIED", got[0].Verdict)
	}
	if len(got[0].EvidenceRefs) != 1 || got[0].EvidenceRefs[0] != "E-001" {
		t.Errorf("refs = %v, want [E-001]", got[0].EvidenceRefs)
	}
	if got[0].Reason == "" {
		t.Errorf("reason must be set")
	}
}

func TestTestsPassContradicted(t *testing.T) {
	r := testReceipt(model.Tier1, model.BaselineHigh)
	r.Claims = []model.Claim{mkClaim("C-001", model.ClaimTestsPass, "all tests pass")}
	r.Evidence = []model.Evidence{cmdEvidence("E-001", []string{"go", "test", "./..."}, 1)}
	got := evaluate(t, r)
	if got[0].Verdict != model.VerdictContradicted {
		t.Errorf("verdict = %s, want CONTRADICTED", got[0].Verdict)
	}
}

func TestTestsPassUnverifiedTier0(t *testing.T) {
	r := testReceipt(model.Tier0, model.BaselineLow)
	r.Claims = []model.Claim{mkClaim("C-001", model.ClaimTestsPass, "all tests pass")}
	got := evaluate(t, r)
	if got[0].Verdict != model.VerdictUnverified {
		t.Errorf("verdict = %s, want UNVERIFIED", got[0].Verdict)
	}
}

func TestTestsPassUnverifiedTier1Gaps(t *testing.T) {
	r := testReceipt(model.Tier1, model.BaselineHigh)
	r.Claims = []model.Claim{mkClaim("C-001", model.ClaimTestsPass, "all tests pass")}
	r.Evidence = []model.Evidence{cmdEvidence("E-001", []string{"npm", "run", "build"}, 0)}
	got := evaluate(t, r)
	if got[0].Verdict != model.VerdictUnverified {
		t.Errorf("no observed test command must not contradict, got %s", got[0].Verdict)
	}
}

func TestReverifyVerified(t *testing.T) {
	r := testReceipt(model.Tier1, model.BaselineHigh)
	r.Claims = []model.Claim{mkClaim("C-001", model.ClaimTestsPass, "all tests pass")}
	r.Evidence = []model.Evidence{reverifyEvidence("E-001", []string{"go", "test", "./..."}, 0)}
	got := evaluate(t, r)
	if got[0].Verdict != model.VerdictVerified {
		t.Errorf("verdict = %s, want VERIFIED", got[0].Verdict)
	}
}

func TestReverifyFailedContradicted(t *testing.T) {
	r := testReceipt(model.Tier1, model.BaselineHigh)
	r.Claims = []model.Claim{mkClaim("C-001", model.ClaimTestsPass, "all tests pass")}
	r.Evidence = []model.Evidence{reverifyEvidence("E-001", []string{"go", "test", "./..."}, 1)}
	got := evaluate(t, r)
	if got[0].Verdict != model.VerdictContradicted {
		t.Errorf("verdict = %s, want CONTRADICTED", got[0].Verdict)
	}
}

func TestBuildRules(t *testing.T) {
	r := testReceipt(model.Tier1, model.BaselineHigh)
	r.Claims = []model.Claim{mkClaim("C-001", model.ClaimBuildOK, "build passed")}
	r.Evidence = []model.Evidence{cmdEvidence("E-001", []string{"npm", "run", "build"}, 0)}
	if got := evaluate(t, r); got[0].Verdict != model.VerdictVerified {
		t.Errorf("build exit 0 = %s, want VERIFIED", got[0].Verdict)
	}
	r.Evidence = []model.Evidence{cmdEvidence("E-001", []string{"go", "build", "./..."}, 1)}
	if got := evaluate(t, r); got[0].Verdict != model.VerdictContradicted {
		t.Errorf("build exit 1 = %s, want CONTRADICTED", got[0].Verdict)
	}
	r.Evidence = nil
	if got := evaluate(t, r); got[0].Verdict != model.VerdictUnverified {
		t.Errorf("no build evidence = %s, want UNVERIFIED", got[0].Verdict)
	}
}

func TestLintNotInferredFromTests(t *testing.T) {
	r := testReceipt(model.Tier1, model.BaselineHigh)
	r.Claims = []model.Claim{mkClaim("C-001", model.ClaimLintClean, "lint is clean")}
	r.Evidence = []model.Evidence{cmdEvidence("E-001", []string{"npm", "test"}, 0)}
	got := evaluate(t, r)
	if got[0].Verdict != model.VerdictUnverified {
		t.Errorf("lint must not infer from test output, got %s", got[0].Verdict)
	}
	r.Evidence = append(r.Evidence, cmdEvidence("E-002", []string{"eslint", "."}, 0))
	got = evaluate(t, r)
	if got[0].Verdict != model.VerdictVerified {
		t.Errorf("lint exit 0 = %s, want VERIFIED", got[0].Verdict)
	}
}

func diffEvidenceDeps(id string, paths, addedDeps, manifests []string) model.Evidence {
	e := diffEvidence(id, paths)
	e.Data["added_dependencies"] = addedDeps
	e.Data["manifests_changed"] = manifests
	return e
}

func TestFilesChangedVerified(t *testing.T) {
	r := testReceipt(model.Tier0, model.BaselineHigh)
	r.Claims = []model.Claim{mkClaim("C-001", model.ClaimFilesChanged, "I only changed src/a.txt")}
	r.Evidence = []model.Evidence{diffEvidence("E-001", []string{"src/a.txt"})}
	got := evaluate(t, r)
	if got[0].Verdict != model.VerdictVerified {
		t.Errorf("exact match high = %s, want VERIFIED", got[0].Verdict)
	}
}

func TestFilesChangedContradicted(t *testing.T) {
	r := testReceipt(model.Tier0, model.BaselineHigh)
	r.Claims = []model.Claim{mkClaim("C-001", model.ClaimFilesChanged, "I only changed src/a.txt")}
	r.Evidence = []model.Evidence{diffEvidence("E-001", []string{"src/a.txt", "src/b.txt"})}
	got := evaluate(t, r)
	if got[0].Verdict != model.VerdictContradicted {
		t.Errorf("extra path high = %s, want CONTRADICTED", got[0].Verdict)
	}
}

func TestFilesChangedLowConfidenceUnverified(t *testing.T) {
	r := testReceipt(model.Tier0, model.BaselineLow)
	r.Claims = []model.Claim{mkClaim("C-001", model.ClaimFilesChanged, "I only changed src/a.txt")}
	r.Evidence = []model.Evidence{diffEvidence("E-001", []string{"src/a.txt", "src/b.txt"})}
	got := evaluate(t, r)
	if got[0].Verdict != model.VerdictUnverified {
		t.Errorf("low confidence mismatch = %s, want UNVERIFIED", got[0].Verdict)
	}
}

func TestNoOtherChangesRules(t *testing.T) {
	r := testReceipt(model.Tier0, model.BaselineHigh)
	r.Claims = []model.Claim{
		mkClaim("C-001", model.ClaimFilesChanged, "I only changed src/a.txt"),
		mkClaim("C-002", model.ClaimNoOtherChanges, "No other files were changed"),
	}
	r.Evidence = []model.Evidence{diffEvidence("E-001", []string{"src/a.txt"})}
	got := evaluate(t, r)
	if got[1].Verdict != model.VerdictVerified {
		t.Errorf("subset high = %s, want VERIFIED", got[1].Verdict)
	}
	r.Evidence = []model.Evidence{diffEvidence("E-001", []string{"src/a.txt", "src/evil.txt"})}
	got = evaluate(t, r)
	if got[1].Verdict != model.VerdictContradicted {
		t.Errorf("extras high = %s, want CONTRADICTED", got[1].Verdict)
	}
}

func TestDependencyRules(t *testing.T) {
	r := testReceipt(model.Tier0, model.BaselineHigh)
	r.Claims = []model.Claim{mkClaim("C-001", model.ClaimDepAdded, "Added lodash as a dependency")}
	r.Evidence = []model.Evidence{diffEvidenceDeps("E-001", []string{"package.json"}, []string{"lodash"}, []string{"package.json"})}
	if got := evaluate(t, r); got[0].Verdict != model.VerdictVerified {
		t.Errorf("manifest adds named dep = %s, want VERIFIED", got[0].Verdict)
	}
	r.Evidence = []model.Evidence{diffEvidenceDeps("E-001", []string{}, []string{}, []string{})}
	if got := evaluate(t, r); got[0].Verdict != model.VerdictUnverified {
		t.Errorf("no manifest change = %s, want UNVERIFIED not contradicted", got[0].Verdict)
	}
	r.Claims = []model.Claim{mkClaim("C-001", model.ClaimNoOtherChanges, "I added no new dependencies")}
	r.Evidence = []model.Evidence{diffEvidenceDeps("E-001", []string{"package.json"}, []string{"lodash"}, []string{"package.json"})}
	if got := evaluate(t, r); got[0].Verdict != model.VerdictContradicted {
		t.Errorf("denial with manifest changes = %s, want CONTRADICTED", got[0].Verdict)
	}
}

func TestIssueAndOtherUnverifiable(t *testing.T) {
	r := testReceipt(model.Tier1, model.BaselineHigh)
	r.Claims = []model.Claim{
		mkClaim("C-001", model.ClaimIssueFixed, "Fixed the bug"),
		mkClaim("C-002", model.ClaimOther, "All tests pass except integration"),
	}
	r.Evidence = []model.Evidence{cmdEvidence("E-001", []string{"npm", "test"}, 0)}
	got := evaluate(t, r)
	if got[0].Verdict != model.VerdictUnverifiable || got[1].Verdict != model.VerdictUnverifiable {
		t.Errorf("issue/other = %s/%s, want UNVERIFIABLE", got[0].Verdict, got[1].Verdict)
	}
}

func TestSummaryAndInvariants(t *testing.T) {
	r := testReceipt(model.Tier0, model.BaselineLow)
	r.Claims = []model.Claim{
		mkClaim("C-001", model.ClaimTestsPass, "t"),
		mkClaim("C-002", model.ClaimBuildOK, "b"),
	}
	r.Claims[0].Verdict = model.VerdictVerified
	r.Claims[1].Verdict = model.VerdictContradicted
	r.Flags = []model.Flag{{ID: "F-001", Code: "X", Severity: model.SeverityInfo, Title: "t", Details: "d", EvidenceRefs: []string{}}}
	r.Summary = Summarize(r.Claims, r.Flags)
	s := Summarize(r.Claims, r.Flags)
	if s.Verified != 1 || s.Contradicted != 1 || s.Flags != 1 {
		t.Errorf("summary = %+v", s)
	}
	if err := CheckInvariants(r); err != nil {
		t.Errorf("valid receipt should pass invariants: %v", err)
	}
	r.Claims[0].EvidenceRefs = []string{"E-999"}
	if err := CheckInvariants(r); err == nil {
		t.Errorf("dangling ref should fail invariants")
	}
}

func TestEngineDeterministic(t *testing.T) {
	r := testReceipt(model.Tier1, model.BaselineHigh)
	r.Claims = []model.Claim{mkClaim("C-001", model.ClaimTestsPass, "all tests pass")}
	r.Evidence = []model.Evidence{cmdEvidence("E-001", []string{"npm", "test"}, 0)}
	a := evaluate(t, r)
	b := evaluate(t, r)
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	if string(ab) != string(bb) {
		t.Errorf("engine not deterministic")
	}
}

func TestEngineCreatesNoEvidence(t *testing.T) {
	r := testReceipt(model.Tier1, model.BaselineHigh)
	r.Claims = []model.Claim{mkClaim("C-001", model.ClaimTestsPass, "all tests pass")}
	r.Evidence = []model.Evidence{cmdEvidence("E-001", []string{"npm", "test"}, 0)}
	n := len(r.Evidence)
	evaluate(t, r)
	if len(r.Evidence) != n {
		t.Errorf("engine must not create evidence")
	}
}
