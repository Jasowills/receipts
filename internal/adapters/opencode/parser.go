package opencode

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/receipts/receipts/internal/adapters"
)

// sessionRow is one row of the LatestSession query output.
type sessionRow struct {
	ID        string `json:"id"`
	Directory string `json:"directory"`
	Title     string `json:"title"`
	UpdatedMs int64  `json:"time_updated"`
}

// messageRow is one row of the message plus part join query output.
// MData and PData arrive as JSON encoded strings inside the -json output.
type messageRow struct {
	MID   string  `json:"mid"`
	MTime int64   `json:"mtime"`
	MData string  `json:"mdata"`
	PTime *int64  `json:"ptime"`
	PData *string `json:"pdata"`
}

func parseSessionRows(data []byte) ([]sessionRow, error) {
	var rows []sessionRow
	dec := json.NewDecoder(strings.NewReader(strings.TrimSpace(string(data))))
	// sqlite3 -json prints one array; tolerate empty output.
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, nil
	}
	if err := dec.Decode(&rows); err != nil {
		return nil, fmt.Errorf("opencode: decode session rows: %w", err)
	}
	return rows, nil
}

// selectLatest picks the most recently updated session.
func selectLatest(rows []sessionRow) (sessionRow, error) {
	if len(rows) == 0 {
		return sessionRow{}, fmt.Errorf("opencode: no sessions for this directory")
	}
	cp := append([]sessionRow{}, rows...)
	sort.Slice(cp, func(a, b int) bool { return cp[a].UpdatedMs > cp[b].UpdatedMs })
	return cp[0], nil
}

// parseMessageRows groups join rows by message and normalizes them.
func parseMessageRows(data []byte) ([]adapters.Message, error) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, nil
	}
	var rows []messageRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, fmt.Errorf("opencode: decode message rows: %w", err)
	}
	return messagesFromRows(rows)
}

func messagesFromRows(rows []messageRow) ([]adapters.Message, error) {
	// Count messages (distinct MIDs) for the bound, not rows.
	seen := map[string]bool{}
	n := 0
	for _, r := range rows {
		if !seen[r.MID] {
			seen[r.MID] = true
			n++
		}
	}
	if n > adapters.MaxMessages {
		return nil, fmt.Errorf("opencode: %d messages exceeds limit %d", n, adapters.MaxMessages)
	}
	var out []adapters.Message
	var cur *adapters.Message
	for _, r := range rows {
		if cur == nil || cur.ID != r.MID {
			var mdata struct {
				Role string `json:"role"`
			}
			_ = json.Unmarshal([]byte(r.MData), &mdata)
			role := adapters.RoleUnknown
			switch mdata.Role {
			case "user":
				role = adapters.RoleUser
			case "assistant":
				role = adapters.RoleAssistant
			}
			out = append(out, adapters.Message{
				ID:        r.MID,
				Role:      role,
				Timestamp: time.UnixMilli(r.MTime).UTC(),
			})
			cur = &out[len(out)-1]
		}
		if r.PData == nil {
			continue
		}
		var part struct {
			Type    string `json:"type"`
			Text    string `json:"text"`
			Ignored bool   `json:"ignored"`
			Tool    string `json:"tool"`
			CallID  string `json:"callID"`
			State   *struct {
				Status string         `json:"status"`
				Input  map[string]any `json:"input"`
				Output string         `json:"output"`
				Error  any            `json:"error"`
			} `json:"state"`
		}
		if err := json.Unmarshal([]byte(*r.PData), &part); err != nil {
			continue
		}
		switch part.Type {
		case "text":
			if part.Ignored || part.Text == "" {
				continue
			}
			if cur.Text != "" {
				cur.Text += "\n"
			}
			cur.Text += part.Text
		case "tool":
			if part.State == nil {
				continue
			}
			switch part.State.Status {
			case "completed":
				args := part.State.Input
				if args == nil {
					args = map[string]any{}
				}
				cur.ToolCalls = append(cur.ToolCalls, adapters.ToolCall{Name: part.Tool, Arguments: args, Result: part.State.Output})
			case "error":
				msg := ""
				switch e := part.State.Error.(type) {
				case string:
					msg = e
				case nil:
					msg = "tool failed"
				default:
					if b, err := json.Marshal(e); err == nil {
						msg = string(b)
					} else {
						msg = "tool failed"
					}
				}
				args := part.State.Input
				if args == nil {
					args = map[string]any{}
				}
				cur.ToolCalls = append(cur.ToolCalls, adapters.ToolCall{Name: part.Tool, Arguments: args, Result: "error: " + msg})
			default:
				// pending and running calls are incomplete; not evidence.
				continue
			}
		default:
			// reasoning, step markers, patches, compaction, and files
			// are not claims or command evidence; skip.
			continue
		}
	}
	if out == nil {
		return nil, nil
	}
	return out, nil
}

// escapeSQL doubles single quotes for safe query interpolation.
func escapeSQL(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
