package model

// SchemaVersion is the current receipt schema version.
const SchemaVersion = "1.0.0"

// ChainAlgorithmSHA256 is the only supported chain hash algorithm in v0.1.
const ChainAlgorithmSHA256 = "sha256"

// Tier records collection fidelity. Tier 2 is reserved for future use.
type Tier int

const (
	Tier0 Tier = 0
	Tier1 Tier = 1
	Tier2 Tier = 2
)

// Verdict is the outcome of joining one claim to evidence.
type Verdict string

const (
	VerdictVerified     Verdict = "VERIFIED"
	VerdictContradicted Verdict = "CONTRADICTED"
	VerdictUnverified   Verdict = "UNVERIFIED"
	VerdictUnverifiable Verdict = "UNVERIFIABLE"
)

// EvidenceKind identifies the type of collected evidence.
type EvidenceKind string

const (
	EvidenceGitSnapshot    EvidenceKind = "git_snapshot"
	EvidenceDiff           EvidenceKind = "diff"
	EvidenceCommand        EvidenceKind = "command"
	EvidenceTestResult     EvidenceKind = "test_result"
	EvidenceReverify       EvidenceKind = "reverify"
	EvidenceTranscript     EvidenceKind = "transcript"
	EvidenceAdapterStatus  EvidenceKind = "adapter_status"
	EvidenceCollectionWarn EvidenceKind = "collection_warning"
)

// BaselineConfidence records how trustworthy the before state is.
type BaselineConfidence string

const (
	BaselineHigh    BaselineConfidence = "high"
	BaselineMedium  BaselineConfidence = "medium"
	BaselineLow     BaselineConfidence = "low"
	BaselineUnknown BaselineConfidence = "unknown"
)

// ClaimType identifies the kind of agent completion claim.
type ClaimType string

const (
	ClaimTestsPass      ClaimType = "TESTS_PASS"
	ClaimBuildOK        ClaimType = "BUILD_OK"
	ClaimLintClean      ClaimType = "LINT_CLEAN"
	ClaimFilesChanged   ClaimType = "FILES_CHANGED"
	ClaimNoOtherChanges ClaimType = "NO_OTHER_CHANGES"
	ClaimDepAdded       ClaimType = "DEPENDENCY_ADDED"
	ClaimIssueFixed     ClaimType = "ISSUE_FIXED"
	ClaimOther          ClaimType = "OTHER"
)

// Severity is the seriousness of an integrity flag.
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityHigh    Severity = "high"
)

// DirtyState records working tree dirtiness where known.
// The receipt schema represents dirty_before as a string enum.
type DirtyState string

const (
	DirtyTrue    DirtyState = "true"
	DirtyFalse   DirtyState = "false"
	DirtyUnknown DirtyState = "unknown"
)
