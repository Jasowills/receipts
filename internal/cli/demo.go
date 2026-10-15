package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/receipts/receipts/internal/demo"
	"github.com/receipts/receipts/internal/render"
	"github.com/receipts/receipts/internal/store"
)

const demoUse = "receipts demo [--scenario default|duplicate-claims|test-weakening|scope-drift] [--format terminal|md|json] [--out PATH] [--keep]"

func printDemoHelp(w io.Writer) {
	fmt.Fprintln(w, "demo - Run a deterministic offline demo scenario")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintf(w, "  %s\n", demoUse)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --scenario NAME          Demo scenario (default default)")
	fmt.Fprintln(w, "  --format terminal|md|json|html   Output format (default terminal)")
	fmt.Fprintln(w, "  --out PATH               Write output to file instead of stdout")
	fmt.Fprintln(w, "  --keep                   Keep the temp directory and print its path")
	fmt.Fprintln(w, "  --help                   Show help for this command")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Runs offline in a temp directory with no network or API key.")
	fmt.Fprintln(w, "Evidence is synthetic; verdicts illustrate the engine only.")
}

// runDemo implements receipts demo. Exit 0 on success, 2 on usage error,
// 4 when the scenario fails to build.
func runDemo(args []string, stdout, stderr io.Writer) int {
	scenario, format, out := "default", "terminal", ""
	keep := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--help" || a == "-h":
			printDemoHelp(stdout)
			return 0
		case a == "--keep":
			keep = true
		case a == "--scenario" && i+1 < len(args):
			i++
			scenario = args[i]
		case strings.HasPrefix(a, "--scenario="):
			scenario = strings.TrimPrefix(a, "--scenario=")
		case a == "--format" && i+1 < len(args):
			i++
			format = args[i]
		case strings.HasPrefix(a, "--format="):
			format = strings.TrimPrefix(a, "--format=")
		case a == "--out" && i+1 < len(args):
			i++
			out = args[i]
		case strings.HasPrefix(a, "--out="):
			out = strings.TrimPrefix(a, "--out=")
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(stderr, "unknown flag: %s\n", a)
			return 2
		default:
			fmt.Fprintf(stderr, "unexpected argument: %s\n", a)
			return 2
		}
	}
	switch format {
	case "terminal", "md", "markdown", "json", "html":
	default:
		fmt.Fprintf(stderr, "unknown format: %s (expected terminal|md|json)\n", format)
		return 2
	}
	rec, dir, cleanup, err := demo.Run(scenario, keep)
	if err != nil {
		fmt.Fprintf(stderr, "demo: %v\n", err)
		return 4
	}
	defer cleanup()
	if keep {
		fmt.Fprintf(stderr, "Kept: %s\n", dir)
	}
	var data []byte
	switch format {
	case "terminal":
		data, err = render.TerminalRenderer{}.Render(rec, render.Options{})
	case "md", "markdown":
		data, err = render.MarkdownRenderer{}.Render(rec, render.Options{})
	case "html":
		data, err = render.HTMLRenderer{}.Render(rec, render.Options{})
	case "json":
		var jb []byte
		jb, err = store.MarshalCanonical(rec)
		if err == nil {
			data = append(jb, '\n')
		}
	}
	if err != nil {
		fmt.Fprintf(stderr, "demo: render failed: %v\n", err)
		return 4
	}
	if out != "" {
		if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
			fmt.Fprintf(stderr, "demo: %v\n", err)
			return 4
		}
		if err := os.WriteFile(out, data, 0644); err != nil {
			fmt.Fprintf(stderr, "demo: %v\n", err)
			return 4
		}
		fmt.Fprintf(stdout, "Wrote %s\n", out)
		return 0
	}
	_, err = stdout.Write(data)
	if err != nil {
		fmt.Fprintf(stderr, "demo: %v\n", err)
		return 4
	}
	return 0
}
