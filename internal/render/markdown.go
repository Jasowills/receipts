package render

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/receipts/receipts/internal/model"
)

// MarkdownRenderer renders receipts as deterministic Markdown.
type MarkdownRenderer struct{}

// Format returns FormatMarkdown.
func (MarkdownRenderer) Format() Format { return FormatMarkdown }

// escapeMarkdown escapes user controlled text for Markdown output.
func escapeMarkdown(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\', '`', '*', '_', '[', ']':
			b.WriteRune('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Render returns deterministic Markdown with stable ID ordering.
func (MarkdownRenderer) Render(r model.Receipt, opts Options) ([]byte, error) {
	claims := append([]model.Claim{}, r.Claims...)
	flags := append([]model.Flag{}, r.Flags...)
	evidence := append([]model.Evidence{}, r.Evidence...)
	sort.Slice(claims, func(a, b int) bool { return claims[a].ID < claims[b].ID })
	sort.Slice(flags, func(a, b int) bool { return flags[a].ID < flags[b].ID })
	sort.Slice(evidence, func(a, b int) bool { return evidence[a].ID < evidence[b].ID })

	var b strings.Builder
	fmt.Fprintf(&b, "# Receipt %s\n\n", r.ID)
	fmt.Fprintf(&b, "- Agent: %s\n", escapeMarkdown(r.Agent.Name))
	if r.Agent.SessionID != "" {
		fmt.Fprintf(&b, "- Session: %s\n", escapeMarkdown(r.Agent.SessionID))
	}
	fmt.Fprintf(&b, "- Tier: %d\n", r.Tier)
	fmt.Fprintf(&b, "- Baseline confidence: %s\n", escapeMarkdown(string(r.Repo.BaselineConfidence)))
	fmt.Fprintf(&b, "- Created: %s\n", r.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"))
	fmt.Fprintf(&b, "- Duration: %dms\n", r.DurationMS)

	fmt.Fprintf(&b, "\n## Claims (%d)\n", len(claims))
	if len(claims) == 0 {
		b.WriteString("\nNone.\n")
	}
	for _, c := range claims {
		fmt.Fprintf(&b, "\n### %s %s %s\n\n", c.ID, escapeMarkdown(string(c.Type)), c.Verdict)
		fmt.Fprintf(&b, "> %s\n\n", escapeMarkdown(c.Text))
		fmt.Fprintf(&b, "- Source: %s (%s)\n", escapeMarkdown(c.Source.MessageID), escapeMarkdown(c.Source.Role))
		fmt.Fprintf(&b, "- Reason: %s\n", escapeMarkdown(c.Reason))
		if len(c.EvidenceRefs) > 0 {
			fmt.Fprintf(&b, "- Evidence: %s\n", escapeMarkdown(strings.Join(c.EvidenceRefs, ", ")))
		} else {
			b.WriteString("- Evidence: none\n")
		}
	}

	fmt.Fprintf(&b, "\n## Flags (%d)\n", len(flags))
	if len(flags) == 0 {
		b.WriteString("\nNone.\n")
	}
	for _, f := range flags {
		fmt.Fprintf(&b, "\n### %s %s %s\n\n", f.ID, escapeMarkdown(f.Code), escapeMarkdown(string(f.Severity)))
		fmt.Fprintf(&b, "%s\n\n", escapeMarkdown(f.Title))
		if f.Details != "" {
			fmt.Fprintf(&b, "%s\n\n", escapeMarkdown(f.Details))
		}
		if len(f.Paths) > 0 {
			paths := make([]string, len(f.Paths))
			for i, p := range f.Paths {
				paths[i] = escapeMarkdown(p)
			}
			fmt.Fprintf(&b, "- Paths: %s\n", strings.Join(paths, ", "))
		}
		if len(f.EvidenceRefs) > 0 {
			fmt.Fprintf(&b, "- Evidence: %s\n", escapeMarkdown(strings.Join(f.EvidenceRefs, ", ")))
		} else {
			b.WriteString("- Evidence: none\n")
		}
	}

	fmt.Fprintf(&b, "\n## Evidence (%d)\n", len(evidence))
	if len(evidence) == 0 {
		b.WriteString("\nNone.\n")
	}
	for _, e := range evidence {
		fmt.Fprintf(&b, "\n- %s %s %s\n", e.ID, escapeMarkdown(string(e.Kind)), e.Timestamp.UTC().Format("2006-01-02T15:04:05Z"))
		if opts.IncludeRaw {
			data, _ := json.Marshal(e.Data)
			fmt.Fprintf(&b, "\n  ```json\n  %s\n  ```\n", string(data))
		}
	}

	if len(r.Warnings) > 0 {
		fmt.Fprintf(&b, "\n## Warnings (%d)\n", len(r.Warnings))
		for _, w := range r.Warnings {
			fmt.Fprintf(&b, "\n- %s\n", escapeMarkdown(w))
		}
	}

	b.WriteString("\n## Summary\n\n")
	fmt.Fprintf(&b, "- Verified: %d\n", r.Summary.Verified)
	fmt.Fprintf(&b, "- Contradicted: %d\n", r.Summary.Contradicted)
	fmt.Fprintf(&b, "- Unverified: %d\n", r.Summary.Unverified)
	fmt.Fprintf(&b, "- Unverifiable: %d\n", r.Summary.Unverifiable)
	fmt.Fprintf(&b, "- Flags: %d\n", r.Summary.Flags)
	return []byte(b.String()), nil
}
