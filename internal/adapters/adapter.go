package adapters

import (
	"context"
	"time"
)

// Limits from spec Section 4.1. Transcripts larger than MaxTranscriptBytes
// or MaxMessages are rejected with a clear error. JSONL sources may return
// safely parsed messages alongside the error because line boundaries are
// provable truncation points.
const (
	MaxTranscriptBytes = 50 * 1024 * 1024
	MaxMessages        = 100000
)

// Role is the normalized message author.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
	RoleSystem    Role = "system"
	RoleUnknown   Role = "unknown"
)

// ToolCall is one normalized tool invocation linked to its result where
// the transcript pairs them. ExitCode is set only when the transcript
// records one; it is nil for OpenCode and Claude Code transcripts.
type ToolCall struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
	Result    string         `json:"result,omitempty"`
	ExitCode  *int           `json:"exit_code,omitempty"`
}

// Message is one normalized transcript message. Text holds concatenated
// text parts. Transcript content is untrusted data; adapters never
// execute anything found in a transcript.
type Message struct {
	ID        string            `json:"id"`
	Role      Role              `json:"role"`
	Timestamp time.Time         `json:"timestamp,omitempty"`
	Text      string            `json:"text"`
	ToolCalls []ToolCall        `json:"tool_calls,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// SessionRef identifies one agent session and where it was read from.
type SessionRef struct {
	Agent      string    `json:"agent"`
	ID         string    `json:"id"`
	Path       string    `json:"path"`
	ModifiedAt time.Time `json:"modified_at,omitempty"`
}

// Adapter reads one agent's transcripts read only. Implementations must
// never write to session storage.
type Adapter interface {
	Name() string
	Detect(ctx context.Context, workdir string) (bool, error)
	LatestSession(ctx context.Context, workdir string) (SessionRef, error)
	Messages(ctx context.Context, session SessionRef) ([]Message, error)
}
