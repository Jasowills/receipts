package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/receipts/receipts/internal/integrity"
	"github.com/receipts/receipts/internal/model"
)

// CommentMarker lets the GitHub Action find and update its own comment
// instead of posting duplicates.
const CommentMarker = "<!-- receipts-report -->"

// CommentRenderer renders concise PR comment bodies. Warnings are
// secret redacted. No link is emitted unless Options.BaseURL names an
// explicit base; the tool never invents hosted links.
type CommentRenderer struct{}

// Format returns FormatComment.
func (CommentRenderer) Format() Format { return FormatComment }

// Render returns the deterministic comment body.
func (CommentRenderer) Render(r model.Receipt, opts Options) ([]byte, error) {
	claims := append([]model.Claim{}, r.Claims...)
	flags := append([]model.Flag{}, r.Flags...)
	sort.Slice(claims, func(a, b int) bool { return claims[a].ID < claims[b].ID })
	sort.Slice(flags, func(a, b int) bool {
		if flags[a].Code != flags[b].Code {
			return flags[a].Code < flags[b].Code
		}
		return flags[a].ID < flags[b].ID
	})

	var b strings.Builder
	b.WriteString(CommentMarker + "\n")
	fmt.Fprintf(&b, "## Receipt %s\n\n", r.ID)
	fmt.Fprintf(&b, "- Tier: %d\n", r.Tier)
	fmt.Fprintf(&b, "- Baseline confidence: %s\n", strings.ToLower(string(r.Repo.BaselineConfidence)))
	fmt.Fprintf(&b, "- Claims: %d verified, %d contradicted, %d unverified, %d unverifiable\n",
		r.Summary.Verified, r.Summary.Contradicted, r.Summary.Unverified, r.Summary.Unverifiable)
	fmt.Fprintf(&b, "- Flags: %d\n", r.Summary.Flags)
	if opts.BaseURL != "" {
		base := strings.TrimRight(opts.BaseURL, "/")
		fmt.Fprintf(&b, "- Receipt: %s/%s\n", base, r.ID)
	}
	if len(claims) > 0 {
		b.WriteString("\n### Claims\n\n")
		for _, c := range claims {
			fmt.Fprintf(&b, "- %s `%s`: **%s**\n", c.ID, c.Type, c.Verdict)
		}
	}
	if len(flags) > 0 {
		b.WriteString("\n### Flags\n\n")
		for _, f := range flags {
			fmt.Fprintf(&b, "- %s `%s` (%s)\n", f.ID, f.Code, f.Severity)
		}
	}
	if len(r.Warnings) > 0 {
		b.WriteString("\n### Warnings\n\n")
		for _, w := range r.Warnings {
			fmt.Fprintf(&b, "- %s\n", integrity.RedactSecrets(w))
		}
	}
	return []byte(b.String()), nil
}
