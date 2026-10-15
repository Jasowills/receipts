package model

import "time"

// Evidence is one independently collected observation.
type Evidence struct {
	ID        string         `json:"id"`
	Kind      EvidenceKind   `json:"kind"`
	Timestamp time.Time      `json:"timestamp"`
	Data      map[string]any `json:"data"`
	Sensitive *bool          `json:"sensitive,omitempty"`
}
