package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/receipts/receipts/internal/model"
	"github.com/receipts/receipts/internal/render"
	"github.com/receipts/receipts/internal/store"
)

const showUse = "receipts show [ID] [--format terminal|md|json|html|svg] [--workdir PATH] [--out PATH] [--raw]"

func printShowHelp(w io.Writer) {
	fmt.Fprintln(w, "show - Show a stored receipt")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintf(w, "  %s\n", showUse)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --format terminal|md|json|html|svg   Output format (default terminal)")
	fmt.Fprintln(w, "  --workdir PATH             Project directory holding .receipts (default .)")
	fmt.Fprintln(w, "  --out PATH                 Write output to file instead of stdout")
	fmt.Fprintln(w, "  --raw                      Include evidence details; may contain sensitive content")
	fmt.Fprintln(w, "  --help                     Show help for this command")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Defaults to the latest receipt in terminal format without color.")
	fmt.Fprintln(w, "Color auto detection lands later; terminal output is plain text for now.")
}

func loadShowReceipt(st *store.Store, id string) (model.Receipt, error) {
	ctx := context.Background()
	if id == "" {
		return st.Latest(ctx)
	}
	return st.Get(ctx, id)
}

// runShow implements receipts show. Exit codes: 0 success, 2 usage error,
// 4 store or schema failure.
func runShow(args []string, stdout, stderr io.Writer) int {
	var id, format, workdir, out string
	var raw bool
	format = "terminal"
	workdir = "."
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--help" || a == "-h":
			printShowHelp(stdout)
			return 0
		case a == "--raw":
			raw = true
		case a == "--format" && i+1 < len(args):
			i++
			format = args[i]
		case strings.HasPrefix(a, "--format="):
			format = strings.TrimPrefix(a, "--format=")
		case a == "--workdir" && i+1 < len(args):
			i++
			workdir = args[i]
		case strings.HasPrefix(a, "--workdir="):
			workdir = strings.TrimPrefix(a, "--workdir=")
		case a == "--out" && i+1 < len(args):
			i++
			out = args[i]
		case strings.HasPrefix(a, "--out="):
			out = strings.TrimPrefix(a, "--out=")
		case strings.HasPrefix(a, "--"):
			fmt.Fprintf(stderr, "unknown flag: %s\n", a)
			fmt.Fprintln(stderr, "Run receipts show --help for usage.")
			return 2
		case strings.HasPrefix(a, "-") && len(a) > 1:
			fmt.Fprintf(stderr, "unknown flag: %s\n", a)
			fmt.Fprintln(stderr, "Run receipts show --help for usage.")
			return 2
		default:
			if id != "" {
				fmt.Fprintln(stderr, "too many arguments: expected at most one ID")
				return 2
			}
			id = a
		}
	}

	switch format {
	case "terminal", "md", "markdown", "json", "html", "svg":
	default:
		fmt.Fprintf(stderr, "unknown format: %s (expected terminal|md|json|html|svg)\n", format)
		return 2
	}

	st := store.New(filepath.Join(workdir, ".receipts"))
	loaded, err := loadShowReceipt(st, id)
	if err != nil {
		fmt.Fprintf(stderr, "show: %v\n", err)
		return 4
	}

	var data []byte
	switch format {
	case "terminal":
		b, rerr := render.TerminalRenderer{}.Render(loaded, render.Options{Color: false, IncludeRaw: raw})
		if rerr != nil {
			fmt.Fprintf(stderr, "show: %v\n", rerr)
			return 4
		}
		data = b
	case "md", "markdown":
		b, rerr := render.MarkdownRenderer{}.Render(loaded, render.Options{IncludeRaw: raw})
		if rerr != nil {
			fmt.Fprintf(stderr, "show: %v\n", rerr)
			return 4
		}
		data = b
	case "html":
		b, rerr := render.HTMLRenderer{}.Render(loaded, render.Options{})
		if rerr != nil {
			fmt.Fprintf(stderr, "show: %v\n", rerr)
			return 4
		}
		data = b
	case "svg":
		b, rerr := render.SVGRenderer{}.Render(loaded, render.Options{})
		if rerr != nil {
			fmt.Fprintf(stderr, "show: %v\n", rerr)
			return 4
		}
		data = b
	case "json":
		b, rerr := store.MarshalCanonical(loaded)
		if rerr != nil {
			fmt.Fprintf(stderr, "show: %v\n", rerr)
			return 4
		}
		data = append(b, '\n')
	}

	if raw && format != "json" {
		fmt.Fprintln(stderr, "warning: --raw output may contain sensitive content")
	}

	if out != "" {
		if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
			fmt.Fprintf(stderr, "show: %v\n", err)
			return 4
		}
		if err := os.WriteFile(out, data, 0644); err != nil {
			fmt.Fprintf(stderr, "show: %v\n", err)
			return 4
		}
		fmt.Fprintf(stdout, "Wrote %s\n", out)
		return 0
	}
	if _, err := stdout.Write(data); err != nil {
		fmt.Fprintf(stderr, "show: %v\n", err)
		return 4
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		fmt.Fprintln(stdout)
	}
	return 0
}
