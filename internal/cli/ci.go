package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/receipts/receipts/internal/render"
	"github.com/receipts/receipts/internal/store"
)

const ciUse = "receipts ci [ID] [--format comment|md|json] [--workdir PATH] [--out PATH] [--base-url URL]"

func printCiHelp(w io.Writer) {
	fmt.Fprintln(w, "ci - Print a PR comment body from a stored receipt")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintf(w, "  %s\n", ciUse)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --format comment|md|json   Output format (default comment)")
	fmt.Fprintln(w, "  --workdir PATH             Project directory (default .)")
	fmt.Fprintln(w, "  --out PATH                 Write output to file instead of stdout")
	fmt.Fprintln(w, "  --base-url URL             Explicit base URL for the receipt link (else no link)")
	fmt.Fprintln(w, "  --help                     Show help for this command")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "No network access. The GitHub Action posts the body.")
}

// runCi implements receipts ci. Exit 0 on success, 2 usage error,
// 4 store failure.
func runCi(args []string, stdout, stderr io.Writer) int {
	var id, format, workdir, out, baseURL string
	format, workdir = "comment", "."
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--help" || a == "-h":
			printCiHelp(stdout)
			return 0
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
		case a == "--base-url" && i+1 < len(args):
			i++
			baseURL = args[i]
		case strings.HasPrefix(a, "--base-url="):
			baseURL = strings.TrimPrefix(a, "--base-url=")
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(stderr, "unknown flag: %s\n", a)
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
	case "comment", "md", "markdown", "json":
	default:
		fmt.Fprintf(stderr, "unknown format: %s (expected comment|md|json)\n", format)
		return 2
	}
	abs, err := filepath.Abs(workdir)
	if err != nil {
		fmt.Fprintf(stderr, "ci: %v\n", err)
		return 4
	}
	ctx := context.Background()
	st := store.New(filepath.Join(abs, ".receipts"))
	var recID = id
	var rec, gerr = st.Get(ctx, recID)
	if id == "" {
		rec, gerr = st.Latest(ctx)
	}
	if gerr != nil {
		fmt.Fprintf(stderr, "ci: %v\n", gerr)
		return 4
	}
	opts := render.Options{BaseURL: baseURL}
	var data []byte
	switch format {
	case "comment":
		data, err = render.CommentRenderer{}.Render(rec, opts)
	case "md", "markdown":
		data, err = render.MarkdownRenderer{}.Render(rec, opts)
	case "json":
		var jb []byte
		jb, err = store.MarshalCanonical(rec)
		if err == nil {
			data = append(jb, '\n')
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "ci: %v\n", err)
		return 4
	}
	if out != "" {
		if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
			fmt.Fprintf(stderr, "ci: %v\n", err)
			return 4
		}
		if err := os.WriteFile(out, data, 0644); err != nil {
			fmt.Fprintf(stderr, "ci: %v\n", err)
			return 4
		}
		fmt.Fprintf(stdout, "Wrote %s\n", out)
		return 0
	}
	_, err = stdout.Write(data)
	if err != nil {
		fmt.Fprintf(stderr, "ci: %v\n", err)
		return 4
	}
	return 0
}
