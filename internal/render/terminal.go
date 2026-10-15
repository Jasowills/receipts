package render

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/receipts/receipts/internal/model"
)

// TerminalRenderer renders receipts as plain text with optional ANSI color.
// Color is suppressed when Options.Color is false or NO_COLOR is set.
type TerminalRenderer struct{}

// Format returns FormatTerminal.
func (TerminalRenderer) Format() Format { return FormatTerminal }

const (
	ansiReset = "\x1b[0m"
	ansiRed   = "\x1b[31m"
	ansiGreen = "\x1b[32m"
	ansiDim   = "\x1b[2m"
	ansiAmber = "\x1b[33m"
)

func terminalColorEnabled(opts Options) bool {
	if !opts.Color {
		return false
	}
	return os.Getenv("NO_COLOR") == ""
}

func paint(s, code string, opts Options) string {
	if !terminalColorEnabled(opts) {
		return s
	}
	return code + s + ansiReset
}

func paintVerdict(v model.Verdict, opts Options) string {
	s := string(v)
	switch v {
	case model.VerdictVerified:
		return paint(s, ansiGreen, opts)
	case model.VerdictContradicted:
		return paint(s, ansiRed, opts)
	case model.VerdictUnverified:
		return paint(s, ansiAmber, opts)
	default:
		return paint(s, ansiDim, opts)
	}
}

func paintSeverity(sev model.Severity, opts Options) string {
	s := string(sev)
	switch sev {
	case model.SeverityHigh:
		return paint(s, ansiRed, opts)
	case model.SeverityWarning:
		return paint(s, ansiAmber, opts)
	default:
		return s
	}
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	return s
}

// Render returns deterministic terminal output with stable ID ordering.
func (TerminalRenderer) Render(r model.Receipt, opts Options) ([]byte, error) {
	claims := append([]model.Claim{}, r.Claims...)
	flags := append([]model.Flag{}, r.Flags...)
	evidence := append([]model.Evidence{}, r.Evidence...)
	sort.Slice(claims, func(a, b int) bool { return claims[a].ID < claims[b].ID })
	sort.Slice(flags, func(a, b int) bool { return flags[a].ID < flags[b].ID })
	sort.Slice(evidence, func(a, b int) bool { return evidence[a].ID < evidence[b].ID })

	var b strings.Builder
	fmt.Fprintf(&b, "RECEIPT %s  %s  tier %d\n", r.ID, r.Agent.Name, r.Tier)
	fmt.Fprintf(&b, "Baseline confidence: %s\n", strings.ToUpper(string(r.Repo.BaselineConfidence)))
	fmt.Fprintf(&b, "Created: %s\n", r.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"))
	fmt.Fprintf(&b, "Duration: %dms\n", r.DurationMS)
	b.WriteString("\nCLAIMS\n")
	if len(claims) == 0 {
		b.WriteString("  (none)\n")
	}
	for _, c := range claims {
		fmt.Fprintf(&b, "  %s %q  %s\n", c.ID, oneLine(c.Text), paintVerdict(c.Verdict, opts))
		if !opts.Compact {
			fmt.Fprintf(&b, "    %s\n", oneLine(c.Reason))
			if len(c.EvidenceRefs) > 0 {
				fmt.Fprintf(&b, "    Evidence: %s\n", strings.Join(c.EvidenceRefs, ", "))
			}
		}
	}
	b.WriteString("\nFLAGS\n")
	if len(flags) == 0 {
		b.WriteString("  (none)\n")
	}
	for _, f := range flags {
		fmt.Fprintf(&b, "  %s  %s  %s\n", paintSeverity(f.Severity, opts), f.Code, oneLine(f.Title))
		if !opts.Compact {
			if f.Details != "" {
				fmt.Fprintf(&b, "    %s\n", oneLine(f.Details))
			}
			if len(f.Paths) > 0 {
				fmt.Fprintf(&b, "    Paths: %s\n", strings.Join(f.Paths, ", "))
			}
		}
	}
	b.WriteString("\nEVIDENCE\n")
	if len(evidence) == 0 {
		b.WriteString("  (none)\n")
	}
	for _, e := range evidence {
		fmt.Fprintf(&b, "  %s  %s  %s\n", e.ID, e.Kind, e.Timestamp.UTC().Format("2006-01-02T15:04:05Z"))
		if opts.IncludeRaw {
			data, _ := json.Marshal(e.Data)
			fmt.Fprintf(&b, "    %s\n", string(data))
		}
	}
	if len(r.Warnings) > 0 {
		b.WriteString("\nWARNINGS\n")
		for _, w := range r.Warnings {
			fmt.Fprintf(&b, "  - %s\n", oneLine(w))
		}
	}
	flagNoun := "flags"
	if r.Summary.Flags == 1 {
		flagNoun = "flag"
	}
	fmt.Fprintf(&b, "\nSummary: %d verified, %d contradicted, %d unverified, %d unverifiable, %d %s\n",
		r.Summary.Verified, r.Summary.Contradicted, r.Summary.Unverified, r.Summary.Unverifiable, r.Summary.Flags, flagNoun)
	return []byte(b.String()), nil
}
