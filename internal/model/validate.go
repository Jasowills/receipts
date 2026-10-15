package model

import (
	"fmt"
	"regexp"
)

var (
	receiptIDPattern  = regexp.MustCompile(`^RC-[A-F0-9]{4}$`)
	claimIDPattern    = regexp.MustCompile(`^C-[0-9]{3,}$`)
	evidenceIDPattern = regexp.MustCompile(`^E-[0-9]{3,}$`)
	flagIDPattern     = regexp.MustCompile(`^F-[0-9]{3,}$`)
)

// ValidateReceipt checks a receipt against the contract in
// schemas/receipt.schema.json using only the standard library.
// It mirrors required fields, patterns, and enums from the schema
// so the model and schema stay in sync without extra dependencies.
// Canonical JSON normalization of nil slices to [] lands with the
// store serializer in Phase B; validation treats nil as empty.
func ValidateReceipt(r Receipt) error {
	if r.SchemaVersion != SchemaVersion {
		return fmt.Errorf("schema_version must be %q", SchemaVersion)
	}
	if !receiptIDPattern.MatchString(r.ID) {
		return fmt.Errorf("id must match ^RC-[A-F0-9]{4}$")
	}
	if r.CreatedAt.IsZero() {
		return fmt.Errorf("created_at is required")
	}
	if r.Agent.Name == "" {
		return fmt.Errorf("agent.name is required")
	}
	if r.Tier != Tier0 && r.Tier != Tier1 && r.Tier != Tier2 {
		return fmt.Errorf("tier must be 0, 1, or 2")
	}
	if r.DurationMS < 0 {
		return fmt.Errorf("duration_ms must be >= 0")
	}
	if r.Repo.Root == "" {
		return fmt.Errorf("repo.root is required")
	}
	switch r.Repo.BaselineConfidence {
	case BaselineHigh, BaselineMedium, BaselineLow, BaselineUnknown:
	default:
		return fmt.Errorf("repo.baseline_confidence must be high, medium, low, or unknown")
	}
	if r.Repo.DirtyBefore != "" {
		switch r.Repo.DirtyBefore {
		case DirtyTrue, DirtyFalse, DirtyUnknown:
		default:
			return fmt.Errorf("repo.dirty_before must be true, false, or unknown")
		}
	}

	evidenceByID := make(map[string]bool, len(r.Evidence))
	for i := range r.Evidence {
		e := r.Evidence[i]
		if !evidenceIDPattern.MatchString(e.ID) {
			return fmt.Errorf("evidence[%d].id must match ^E-[0-9]{3,}$", i)
		}
		if evidenceByID[e.ID] {
			return fmt.Errorf("duplicate evidence id %q", e.ID)
		}
		evidenceByID[e.ID] = true
		switch e.Kind {
		case EvidenceGitSnapshot, EvidenceDiff, EvidenceCommand, EvidenceTestResult,
			EvidenceReverify, EvidenceTranscript, EvidenceAdapterStatus, EvidenceCollectionWarn:
		default:
			return fmt.Errorf("evidence %q has unknown kind %q", e.ID, e.Kind)
		}
		if e.Timestamp.IsZero() {
			return fmt.Errorf("evidence %q timestamp is required", e.ID)
		}
		if e.Data == nil {
			return fmt.Errorf("evidence %q data is required", e.ID)
		}
	}

	claimByID := make(map[string]bool, len(r.Claims))
	for i := range r.Claims {
		c := r.Claims[i]
		if !claimIDPattern.MatchString(c.ID) {
			return fmt.Errorf("claims[%d].id must match ^C-[0-9]{3,}$", i)
		}
		if claimByID[c.ID] {
			return fmt.Errorf("duplicate claim id %q", c.ID)
		}
		claimByID[c.ID] = true
		switch c.Type {
		case ClaimTestsPass, ClaimBuildOK, ClaimLintClean, ClaimFilesChanged,
			ClaimNoOtherChanges, ClaimDepAdded, ClaimIssueFixed, ClaimOther:
		default:
			return fmt.Errorf("claim %q has unknown type %q", c.ID, c.Type)
		}
		if c.Source.MessageID == "" {
			return fmt.Errorf("claim %q source.message_id is required", c.ID)
		}
		if c.Source.Role == "" {
			return fmt.Errorf("claim %q source.role is required", c.ID)
		}
		switch c.Verdict {
		case VerdictVerified, VerdictContradicted, VerdictUnverified, VerdictUnverifiable:
		default:
			return fmt.Errorf("claim %q has unknown verdict %q", c.ID, c.Verdict)
		}
		if err := checkEvidenceRefs(c.ID, c.EvidenceRefs, evidenceByID); err != nil {
			return err
		}
	}

	for i := range r.Flags {
		f := r.Flags[i]
		if !flagIDPattern.MatchString(f.ID) {
			return fmt.Errorf("flags[%d].id must match ^F-[0-9]{3,}$", i)
		}
		switch f.Severity {
		case SeverityInfo, SeverityWarning, SeverityHigh:
		default:
			return fmt.Errorf("flag %q has unknown severity %q", f.ID, f.Severity)
		}
		if err := checkEvidenceRefs(f.ID, f.EvidenceRefs, evidenceByID); err != nil {
			return err
		}
	}

	if r.Summary.Verified < 0 || r.Summary.Contradicted < 0 || r.Summary.Unverified < 0 || r.Summary.Unverifiable < 0 || r.Summary.Flags < 0 {
		return fmt.Errorf("summary counts must be >= 0")
	}
	if r.Chain.Algorithm != ChainAlgorithmSHA256 {
		return fmt.Errorf("chain.algorithm must be sha256")
	}
	return nil
}

func checkEvidenceRefs(owner string, refs []string, known map[string]bool) error {
	seen := make(map[string]bool, len(refs))
	for _, ref := range refs {
		if !evidenceIDPattern.MatchString(ref) {
			return fmt.Errorf("%q has malformed evidence ref %q", owner, ref)
		}
		if seen[ref] {
			return fmt.Errorf("%q has duplicate evidence ref %q", owner, ref)
		}
		seen[ref] = true
		if !known[ref] {
			return fmt.Errorf("%q references unknown evidence %q", owner, ref)
		}
	}
	return nil
}
