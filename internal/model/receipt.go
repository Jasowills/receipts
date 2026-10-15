package model

import "time"

// Agent identifies the coding agent and session when known.
type Agent struct {
	Name           string `json:"name"`
	SessionID      string `json:"session_id,omitempty"`
	AdapterVersion string `json:"adapter_version,omitempty"`
}

// Repo records repository state and baseline confidence.
type Repo struct {
	Root               string             `json:"root"`
	HeadBefore         string             `json:"head_before,omitempty"`
	HeadAfter          string             `json:"head_after,omitempty"`
	BranchBefore       string             `json:"branch_before,omitempty"`
	BranchAfter        string             `json:"branch_after,omitempty"`
	BaselineConfidence BaselineConfidence `json:"baseline_confidence"`
	DirtyBefore        DirtyState         `json:"dirty_before,omitempty"`
	DirtyAfter         *bool              `json:"dirty_after,omitempty"`
	NonGit             *bool              `json:"non_git,omitempty"`
}

// Summary counts claims by verdict and flags separately.
type Summary struct {
	Verified     int `json:"verified"`
	Contradicted int `json:"contradicted"`
	Unverified   int `json:"unverified"`
	Unverifiable int `json:"unverifiable"`
	Flags        int `json:"flags"`
}

// Chain carries hash chain linkage for the stored event.
type Chain struct {
	EventHash    string `json:"event_hash"`
	PreviousHash string `json:"previous_hash"`
	Algorithm    string `json:"algorithm"`
}

// Receipt is the canonical evidence based check record.
type Receipt struct {
	SchemaVersion string     `json:"schema_version"`
	ID            string     `json:"id"`
	CreatedAt     time.Time  `json:"created_at"`
	Agent         Agent      `json:"agent"`
	Tier          Tier       `json:"tier"`
	DurationMS    int64      `json:"duration_ms"`
	Repo          Repo       `json:"repo"`
	Claims        []Claim    `json:"claims"`
	Flags         []Flag     `json:"flags"`
	Evidence      []Evidence `json:"evidence"`
	Warnings      []string   `json:"warnings"`
	Summary       Summary    `json:"summary"`
	Chain         Chain      `json:"chain"`
}

// ReceiptSummary is a compact listing entry for stored receipts.
type ReceiptSummary struct {
	ID           string    `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	AgentName    string    `json:"agent_name"`
	Tier         Tier      `json:"tier"`
	Verified     int       `json:"verified"`
	Contradicted int       `json:"contradicted"`
	Unverified   int       `json:"unverified"`
	Unverifiable int       `json:"unverifiable"`
	Flags        int       `json:"flags"`
}

// ChainVerification reports hash chain validation.
// OK is true when the chain is intact. Otherwise FailedSequence holds
// the first failing event sequence (1 based) and Reason explains why.
type ChainVerification struct {
	OK             bool   `json:"ok"`
	FailedSequence int    `json:"failed_sequence,omitempty"`
	Reason         string `json:"reason,omitempty"`
}
