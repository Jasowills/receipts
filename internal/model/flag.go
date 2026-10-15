package model

// Flag is an integrity observation. Flags never automatically
// contradict a claim; the verdict engine joins claims to evidence
// separately.
type Flag struct {
	ID           string   `json:"id"`
	Code         string   `json:"code"`
	Severity     Severity `json:"severity"`
	Title        string   `json:"title"`
	Details      string   `json:"details"`
	Paths        []string `json:"paths,omitempty"`
	EvidenceRefs []string `json:"evidence_refs"`
}
