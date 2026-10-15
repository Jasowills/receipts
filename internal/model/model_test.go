package model

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestEnumValuesMatchSpec(t *testing.T) {
	if string(VerdictVerified) != "VERIFIED" {
		t.Errorf("VerdictVerified = %q, want VERIFIED", VerdictVerified)
	}
	if string(VerdictContradicted) != "CONTRADICTED" {
		t.Errorf("VerdictContradicted = %q, want CONTRADICTED", VerdictContradicted)
	}
	if string(VerdictUnverified) != "UNVERIFIED" {
		t.Errorf("VerdictUnverified = %q, want UNVERIFIED", VerdictUnverified)
	}
	if string(VerdictUnverifiable) != "UNVERIFIABLE" {
		t.Errorf("VerdictUnverifiable = %q, want UNVERIFIABLE", VerdictUnverifiable)
	}

	if Tier0 != 0 || Tier1 != 1 || Tier2 != 2 {
		t.Errorf("tiers = %d,%d,%d, want 0,1,2", Tier0, Tier1, Tier2)
	}

	kinds := map[EvidenceKind]string{
		EvidenceGitSnapshot:    "git_snapshot",
		EvidenceDiff:           "diff",
		EvidenceCommand:        "command",
		EvidenceTestResult:     "test_result",
		EvidenceReverify:       "reverify",
		EvidenceTranscript:     "transcript",
		EvidenceAdapterStatus:  "adapter_status",
		EvidenceCollectionWarn: "collection_warning",
	}
	for k, want := range kinds {
		if string(k) != want {
			t.Errorf("evidence kind = %q, want %q", k, want)
		}
	}

	if string(BaselineHigh) != "high" || string(BaselineMedium) != "medium" || string(BaselineLow) != "low" || string(BaselineUnknown) != "unknown" {
		t.Errorf("baseline confidence values mismatch")
	}

	claimTypes := map[ClaimType]string{
		ClaimTestsPass:      "TESTS_PASS",
		ClaimBuildOK:        "BUILD_OK",
		ClaimLintClean:      "LINT_CLEAN",
		ClaimFilesChanged:   "FILES_CHANGED",
		ClaimNoOtherChanges: "NO_OTHER_CHANGES",
		ClaimDepAdded:       "DEPENDENCY_ADDED",
		ClaimIssueFixed:     "ISSUE_FIXED",
		ClaimOther:          "OTHER",
	}
	for k, want := range claimTypes {
		if string(k) != want {
			t.Errorf("claim type = %q, want %q", k, want)
		}
	}

	if string(SeverityInfo) != "info" || string(SeverityWarning) != "warning" || string(SeverityHigh) != "high" {
		t.Errorf("severity values mismatch")
	}

	if SchemaVersion != "1.0.0" {
		t.Errorf("SchemaVersion = %q, want 1.0.0", SchemaVersion)
	}
	if ChainAlgorithmSHA256 != "sha256" {
		t.Errorf("ChainAlgorithmSHA256 = %q, want sha256", ChainAlgorithmSHA256)
	}
}

func TestReceiptJSONRoundTrip(t *testing.T) {
	dirtyAfter := true
	created := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	r := Receipt{
		SchemaVersion: SchemaVersion,
		ID:            "RC-4F2A",
		CreatedAt:     created,
		Agent:         Agent{Name: "opencode", SessionID: "session-example", AdapterVersion: "1"},
		Tier:          Tier0,
		DurationMS:    840000,
		Repo: Repo{
			Root:               ".",
			HeadBefore:         "abc123",
			HeadAfter:          "abc123",
			BranchBefore:       "main",
			BranchAfter:        "main",
			BaselineConfidence: BaselineLow,
			DirtyBefore:        DirtyUnknown,
			DirtyAfter:         &dirtyAfter,
		},
		Claims:   []Claim{},
		Flags:    []Flag{},
		Evidence: []Evidence{},
		Warnings: []string{},
		Summary:  Summary{Verified: 0, Contradicted: 0, Unverified: 0, Unverifiable: 0, Flags: 0},
		Chain:    Chain{EventHash: "", PreviousHash: "", Algorithm: ChainAlgorithmSHA256},
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !strings.Contains(string(data), `"schema_version":"1.0.0"`) {
		t.Errorf("marshaled JSON missing schema_version, got %s", data)
	}
	if !strings.Contains(string(data), `"id":"RC-4F2A"`) {
		t.Errorf("marshaled JSON missing id, got %s", data)
	}

	var back Receipt
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if back.ID != r.ID {
		t.Errorf("round trip ID = %q, want %q", back.ID, r.ID)
	}
	if back.SchemaVersion != r.SchemaVersion {
		t.Errorf("round trip schema_version = %q, want %q", back.SchemaVersion, r.SchemaVersion)
	}
	if !back.CreatedAt.Equal(created) {
		t.Errorf("round trip created_at = %v, want %v", back.CreatedAt, created)
	}
	if back.Tier != Tier0 {
		t.Errorf("round trip tier = %d, want 0", back.Tier)
	}
	if back.Repo.BaselineConfidence != BaselineLow {
		t.Errorf("round trip baseline = %q, want low", back.Repo.BaselineConfidence)
	}
	if back.Chain.Algorithm != ChainAlgorithmSHA256 {
		t.Errorf("round trip algorithm = %q, want sha256", back.Chain.Algorithm)
	}
}

func TestClaimAndEvidenceJSON(t *testing.T) {
	msgTime := time.Date(2026, 10, 14, 12, 1, 0, 0, time.UTC)
	c := Claim{
		ID:   "C-001",
		Type: ClaimTestsPass,
		Text: "all tests pass",
		Source: Source{
			MessageID:   "m-1",
			Role:        "assistant",
			Timestamp:   &msgTime,
			ExcerptHash: "abc",
		},
		Verdict:      VerdictUnverified,
		EvidenceRefs: []string{},
		Reason:       "No command evidence exists in audit mode.",
	}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal claim failed: %v", err)
	}
	var back Claim
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("unmarshal claim failed: %v", err)
	}
	if back.ID != "C-001" || back.Type != ClaimTestsPass || back.Verdict != VerdictUnverified {
		t.Errorf("claim round trip mismatch: %+v", back)
	}
	if back.Source.MessageID != "m-1" || back.Source.Role != "assistant" {
		t.Errorf("source round trip mismatch: %+v", back.Source)
	}

	e := Evidence{
		ID:        "E-001",
		Kind:      EvidenceGitSnapshot,
		Timestamp: msgTime,
		Data:      map[string]any{"root": "."},
	}
	edata, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal evidence failed: %v", err)
	}
	var eback Evidence
	if err := json.Unmarshal(edata, &eback); err != nil {
		t.Fatalf("unmarshal evidence failed: %v", err)
	}
	if eback.ID != "E-001" || eback.Kind != EvidenceGitSnapshot {
		t.Errorf("evidence round trip mismatch: %+v", eback)
	}
}
