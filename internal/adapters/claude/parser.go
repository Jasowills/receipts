package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/receipts/receipts/internal/adapters"
)

// contentBlock is one element of a message content array.
type contentBlock struct {
	Type    string         `json:"type"`
	Text    string         `json:"text"`
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Input   map[string]any `json:"input"`
	ToolUse string         `json:"tool_use_id"`
	Content any            `json:"content"`
}

// line is one JSONL transcript line. Only decoded fields are listed;
// unknown fields are ignored so schema drift cannot crash parsing.
type line struct {
	Type      string `json:"type"`
	UUID      string `json:"uuid"`
	Timestamp string `json:"timestamp"`
	Cwd       string `json:"cwd"`
	Sidechain bool   `json:"isSidechain"`
	Subtype   string `json:"subtype"`
	Message   *struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

// parseJSONL normalizes transcript bytes. Unknown line types are skipped.
// tool_result blocks pair into earlier tool_use calls by id; orphan
// results are dropped. At the message cap it returns the safely parsed
// prefix alongside the error because JSONL line boundaries are provable
// truncation points.
func parseJSONL(data []byte) ([]adapters.Message, error) {
	if int64(len(data)) > adapters.MaxTranscriptBytes {
		return nil, fmt.Errorf("transcript exceeds %d bytes", adapters.MaxTranscriptBytes)
	}
	lines := bytes.Split(data, []byte("\n"))
	var out []adapters.Message
	pending := map[string][2]int{}
	for li, raw := range lines {
		raw = bytes.TrimSpace(raw)
		if len(raw) == 0 {
			continue
		}
		if len(out) >= adapters.MaxMessages {
			return out, fmt.Errorf("message cap %d reached with content remaining", adapters.MaxMessages)
		}
		var l line
		if err := json.Unmarshal(raw, &l); err != nil {
			return nil, fmt.Errorf("corrupt line %d: %w", li+1, err)
		}
		switch l.Type {
		case "system":
			out = append(out, adapters.Message{ID: l.UUID, Role: adapters.RoleSystem, Timestamp: parseTime(l.Timestamp), Text: l.Subtype, Metadata: meta(l)})
		case "user", "assistant":
			role := adapters.RoleUser
			if l.Type == "assistant" {
				role = adapters.RoleAssistant
			}
			msg := adapters.Message{ID: l.UUID, Role: role, Timestamp: parseTime(l.Timestamp), Metadata: meta(l)}
			var ids []string
			if l.Message != nil {
				msg.Text, msg.ToolCalls, ids = splitContent(l.Message.Content)
			}
			if role == adapters.RoleUser {
				consumeResults(l.MessageContent(), out, pending)
				if msg.Text == "" {
					continue
				}
			} else {
				out = append(out, msg)
				for ci, id := range ids {
					if id != "" {
						pending[id] = [2]int{len(out) - 1, ci}
					}
				}
				continue
			}
			out = append(out, msg)
		default:
			// Meta lines (summary, cost-state, attachments) carry no
			// claims or command evidence; skip.
			continue
		}
	}
	return out, nil
}

func meta(l line) map[string]string {
	m := map[string]string{}
	if l.Sidechain {
		m["sidechain"] = "true"
	}
	if l.Cwd != "" {
		m["cwd"] = l.Cwd
	}
	return m
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if ts, err := time.Parse(time.RFC3339, s); err == nil {
		return ts.UTC()
	}
	return time.Time{}
}

// MessageContent returns the raw content envelope, tolerating absent mail.
func (l line) MessageContent() json.RawMessage {
	if l.Message == nil {
		return nil
	}
	return l.Message.Content
}

// splitContent separates text from tool calls, returning tool_use ids
// aligned with the returned calls.
func splitContent(raw json.RawMessage) (string, []adapters.ToolCall, []string) {
	if len(raw) == 0 {
		return "", nil, nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil, nil
	}
	var blocks []contentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return "", nil, nil
	}
	var texts []string
	var calls []adapters.ToolCall
	var ids []string
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if b.Text != "" {
				texts = append(texts, b.Text)
			}
		case "tool_use":
			args := b.Input
			if args == nil {
				args = map[string]any{}
			}
			calls = append(calls, adapters.ToolCall{Name: b.Name, Arguments: args})
			ids = append(ids, b.ID)
		default:
			// thinking blocks and unknown shapes are skipped.
			continue
		}
	}
	return strings.Join(texts, "\n"), calls, ids
}

// consumeResults fills pending tool calls from tool_result blocks.
func consumeResults(raw json.RawMessage, out []adapters.Message, pending map[string][2]int) {
	var blocks []contentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return
	}
	for _, b := range blocks {
		if b.Type != "tool_result" || b.ToolUse == "" {
			continue
		}
		idx, ok := pending[b.ToolUse]
		if !ok {
			continue
		}
		out[idx[0]].ToolCalls[idx[1]].Result = resultText(b.Content)
		delete(pending, b.ToolUse)
	}
}

func resultText(content any) string {
	switch c := content.(type) {
	case string:
		return c
	case []any:
		var parts []string
		for _, item := range c {
			if m, ok := item.(map[string]any); ok {
				if m["type"] == "text" {
					if s, ok := m["text"].(string); ok {
						parts = append(parts, s)
					}
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

// slugForDir encodes a workdir the way Claude Code names project folders.
func slugForDir(dir string) string {
	return strings.ReplaceAll(dir, "/", "-")
}
