package model

import "time"

// Source links a claim to the agent message it came from.
type Source struct {
	MessageID   string     `json:"message_id"`
	Role        string     `json:"role"`
	Timestamp   *time.Time `json:"timestamp,omitempty"`
	ExcerptHash string     `json:"excerpt_hash,omitempty"`
}

// ExtractionWarning is a non fatal claim extraction note. The audit
// pipeline converts conflict warnings into CONFLICTING_AGENT_CLAIMS flags.
type ExtractionWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ExtractionConflictAgents marks contradictory claims in one transcript.
const ExtractionConflictAgents = "CONFLICTING_AGENT_CLAIMS"

// VerdictWarning is a non fatal verdict engine note.
type VerdictWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Claim is one agent completion assertion plus its verdict.
type Claim struct {
	ID           string    `json:"id"`
	Type         ClaimType `json:"type"`
	Text         string    `json:"text"`
	Source       Source    `json:"source"`
	Verdict      Verdict   `json:"verdict"`
	EvidenceRefs []string  `json:"evidence_refs"`
	Reason       string    `json:"reason"`
}
