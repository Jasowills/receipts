package render

import (
	"github.com/receipts/receipts/internal/model"
)

// Format identifies a renderer output format.
type Format string

const (
	FormatTerminal Format = "terminal"
	FormatMarkdown Format = "md"
	FormatHTML     Format = "html"
	FormatJSON     Format = "json"
	FormatComment  Format = "comment"
	FormatSVG      Format = "svg"
)

// Options controls renderer output.
type Options struct {
	Color      bool
	Compact    bool
	IncludeRaw bool
	BaseURL    string
}

// Renderer renders a receipt in one format.
// Renderers are pure functions over a receipt and options.
type Renderer interface {
	Format() Format
	Render(receipt model.Receipt, options Options) ([]byte, error)
}
