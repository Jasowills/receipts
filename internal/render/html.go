package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/receipts/receipts/internal/model"
)

// HTMLRenderer renders receipts as a single self contained page.
// All user controlled text is HTML escaped. Output carries no inline
// scripts, remote assets, links, or event handler attributes.
type HTMLRenderer struct{}

// Format returns FormatHTML.
func (HTMLRenderer) Format() Format { return FormatHTML }

func htmlEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&#34;")
		case '\'':
			b.WriteString("&#39;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func verdictClass(v model.Verdict) string {
	switch v {
	case model.VerdictVerified:
		return "verified"
	case model.VerdictContradicted:
		return "contradicted"
	case model.VerdictUnverified:
		return "unverified"
	default:
		return "unverifiable"
	}
}

// Render returns deterministic HTML with stable ID ordering.
func (HTMLRenderer) Render(r model.Receipt, opts Options) ([]byte, error) {
	claims := append([]model.Claim{}, r.Claims...)
	flags := append([]model.Flag{}, r.Flags...)
	evidence := append([]model.Evidence{}, r.Evidence...)
	sort.Slice(claims, func(a, b int) bool { return claims[a].ID < claims[b].ID })
	sort.Slice(flags, func(a, b int) bool { return flags[a].ID < flags[b].ID })
	sort.Slice(evidence, func(a, b int) bool { return evidence[a].ID < evidence[b].ID })

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n<title>Receipt " + htmlEscape(r.ID) + "</title>\n<style>\n")
	b.WriteString("body{font-family:monospace;max-width:72em;margin:2em;color:#1a1a1a;background:#fff}")
	b.WriteString(".verified{color:#0f6e56;font-weight:bold}.contradicted{color:#b00020;font-weight:bold}.unverified{color:#ba7517;font-weight:bold}.unverifiable{color:#555}")
	b.WriteString("</style>\n</head>\n<body>\n")
	fmt.Fprintf(&b, "<h1>Receipt %s</h1>\n", htmlEscape(r.ID))
	fmt.Fprintf(&b, "<dl><dt>Agent</dt><dd>%s</dd><dt>Tier</dt><dd>%d</dd><dt>Baseline confidence</dt><dd>%s</dd><dt>Created</dt><dd>%s</dd><dt>Duration</dt><dd>%dms</dd></dl>\n",
		htmlEscape(r.Agent.Name), r.Tier, htmlEscape(string(r.Repo.BaselineConfidence)),
		htmlEscape(r.CreatedAt.UTC().Format("2006-01-02T15:04:05Z")), r.DurationMS)
	b.WriteString("<h2>Claims</h2>\n<ul>\n")
	for _, c := range claims {
		fmt.Fprintf(&b, "<li><strong>%s</strong> <code>%s</code> <span class=\"%s\">%s</span><br>%s<br>Reason: %s",
			htmlEscape(c.ID), htmlEscape(string(c.Type)), verdictClass(c.Verdict), htmlEscape(string(c.Verdict)),
			htmlEscape(c.Text), htmlEscape(c.Reason))
		if len(c.EvidenceRefs) > 0 {
			fmt.Fprintf(&b, "<br>Evidence: %s", htmlEscape(strings.Join(c.EvidenceRefs, ", ")))
		}
		b.WriteString("</li>\n")
	}
	b.WriteString("</ul>\n<h2>Flags</h2>\n<ul>\n")
	for _, f := range flags {
		fmt.Fprintf(&b, "<li><strong>%s</strong> <code>%s</code> [%s]<br>%s<br>%s",
			htmlEscape(f.ID), htmlEscape(f.Code), htmlEscape(string(f.Severity)), htmlEscape(f.Title), htmlEscape(f.Details))
		if len(f.Paths) > 0 {
			fmt.Fprintf(&b, "<br>Paths: %s", htmlEscape(strings.Join(f.Paths, ", ")))
		}
		b.WriteString("</li>\n")
	}
	b.WriteString("</ul>\n<h2>Evidence</h2>\n<ul>\n")
	for _, e := range evidence {
		fmt.Fprintf(&b, "<li>%s %s %s</li>\n", htmlEscape(e.ID), htmlEscape(string(e.Kind)), htmlEscape(e.Timestamp.UTC().Format("2006-01-02T15:04:05Z")))
	}
	b.WriteString("</ul>\n")
	if len(r.Warnings) > 0 {
		b.WriteString("<h2>Warnings</h2>\n<ul>\n")
		for _, w := range r.Warnings {
			fmt.Fprintf(&b, "<li>%s</li>\n", htmlEscape(w))
		}
		b.WriteString("</ul>\n")
	}
	fmt.Fprintf(&b, "<p>Summary: %d verified, %d contradicted, %d unverified, %d unverifiable, %d flags</p>\n",
		r.Summary.Verified, r.Summary.Contradicted, r.Summary.Unverified, r.Summary.Unverifiable, r.Summary.Flags)
	b.WriteString("</body>\n</html>\n")
	_ = opts
	return []byte(b.String()), nil
}
