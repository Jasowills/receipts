package render

import (
	"fmt"
	"strings"

	"github.com/receipts/receipts/internal/model"
)

// SVGRenderer renders a deterministic status badge. Output carries no
// scripts, links, or external resources. Text lengths are fixed with
// textLength so layout does not depend on viewer fonts.
type SVGRenderer struct{}

// Format returns FormatSVG.
func (SVGRenderer) Format() Format { return FormatSVG }

// Render returns the badge for a receipt summary.
func (SVGRenderer) Render(r model.Receipt, _ Options) ([]byte, error) {
	right := fmt.Sprintf("%d verified, %d contradicted", r.Summary.Verified, r.Summary.Contradicted)
	if r.Summary.Unverified > 0 {
		right += fmt.Sprintf(", %d unverified", r.Summary.Unverified)
	}
	color := "#BA7517"
	if r.Summary.Contradicted > 0 {
		color = "#B00020"
	} else if r.Summary.Verified > 0 {
		color = "#0F6E56"
	}
	var b strings.Builder
	b.WriteString("<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"360\" height=\"40\" role=\"img\">")
	fmt.Fprintf(&b, "<title>Receipt %s</title>", htmlEscape(r.ID))
	b.WriteString("<rect width=\"360\" height=\"40\" rx=\"6\" fill=\"#1a1a1a\"/>")
	fmt.Fprintf(&b, "<rect x=\"120\" width=\"240\" height=\"40\" rx=\"6\" fill=\"%s\"/>", color)
	b.WriteString("<rect x=\"120\" width=\"12\" height=\"40\" fill=\"#1a1a1a\"/>")
	b.WriteString("<text x=\"60\" y=\"26\" text-anchor=\"middle\" font-family=\"monospace\" font-size=\"14\" fill=\"#fff\" textLength=\"96\" lengthAdjust=\"spacingAndGlyphs\">receipts</text>")
	fmt.Fprintf(&b, "<text x=\"240\" y=\"26\" text-anchor=\"middle\" font-family=\"monospace\" font-size=\"14\" fill=\"#fff\" textLength=\"216\" lengthAdjust=\"spacingAndGlyphs\">%s</text>", htmlEscape(right))
	b.WriteString("</svg>\n")
	return []byte(b.String()), nil
}
