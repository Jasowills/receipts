package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const initUse = "receipts init [--workdir PATH] [--force]"

const starterConfig = `# Receipts configuration. All keys are optional.
# Defaults are conservative. TOML loading and validation land in a later
# release; this file documents the planned keys from the spec.
agent = "auto" # auto | opencode | claude | codex
strictness = "conservative" # conservative | balanced | strict

[test]
command = "npm test"
timeout_seconds = 900

[scope]
allowed_paths = ["src/**", "tests/**"]
forbidden_paths = [".github/**", "package.json", "package-lock.json"]

[checks]
test_files_changed = true
assertions_removed = true
skips_added = true
out_of_scope_changes = true
dependencies_changed = true
build_config_changed = true
secret_patterns = true

[store]
directory = ".receipts"
retain_output_tail_bytes = 8192

[llm]
enabled = false
provider = "local"
model = ""
endpoint = "http://127.0.0.1:11434"
`

func printInitHelp(w io.Writer) {
	fmt.Fprintln(w, "init - Write starter config and gitignore entry")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintf(w, "  %s\n", initUse)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --workdir PATH   Project directory (default .)")
	fmt.Fprintln(w, "  --force          Replace existing files after backing them up")
	fmt.Fprintln(w, "  --help           Show help for this command")
}

// runInit implements receipts init. Exit 0 on success, 2 on usage error
// or when existing files block the write without --force.
func runInit(args []string, stdout, stderr io.Writer) int {
	workdir := "."
	force := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--help" || a == "-h":
			printInitHelp(stdout)
			return 0
		case a == "--force":
			force = true
		case a == "--workdir" && i+1 < len(args):
			i++
			workdir = args[i]
		case strings.HasPrefix(a, "--workdir="):
			workdir = strings.TrimPrefix(a, "--workdir=")
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(stderr, "unknown flag: %s\n", a)
			return 2
		default:
			fmt.Fprintf(stderr, "unexpected argument: %s\n", a)
			return 2
		}
	}
	if _, err := os.Stat(workdir); err != nil {
		fmt.Fprintf(stderr, "init: %v\n", err)
		return 2
	}
	cfgPath := filepath.Join(workdir, "receipts.toml")
	if _, err := os.Stat(cfgPath); err == nil && !force {
		fmt.Fprintf(stderr, "init: %s exists (pass --force to replace with backup)\n", cfgPath)
		return 2
	}
	if _, err := os.Stat(cfgPath); err == nil {
		if err := backupFile(cfgPath); err != nil {
			fmt.Fprintf(stderr, "init: refusing to replace %s: %v\n", cfgPath, err)
			return 2
		}
	}
	if err := os.WriteFile(cfgPath, []byte(starterConfig), 0644); err != nil {
		fmt.Fprintf(stderr, "init: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "Wrote %s\n", cfgPath)

	ignorePath := filepath.Join(workdir, ".gitignore")
	data, err := os.ReadFile(ignorePath)
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(stderr, "init: %v\n", err)
		return 2
	}
	if hasIgnoreEntry(string(data)) {
		fmt.Fprintf(stdout, "%s already ignores .receipts/\n", ignorePath)
		return 0
	}
	if err == nil && force {
		if berr := backupFile(ignorePath); berr != nil {
			fmt.Fprintf(stderr, "init: refusing to modify %s: %v\n", ignorePath, berr)
			return 2
		}
	}
	entry := ".receipts/\n"
	if len(data) > 0 && !strings.HasSuffix(string(data), "\n") {
		entry = "\n" + entry
	}
	f, err := os.OpenFile(ignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(stderr, "init: %v\n", err)
		return 2
	}
	defer f.Close()
	if _, err := f.WriteString(entry); err != nil {
		fmt.Fprintf(stderr, "init: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "Added .receipts/ to %s\n", ignorePath)
	return 0
}

func hasIgnoreEntry(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		if t := strings.TrimSpace(line); t == ".receipts/" || t == ".receipts" {
			return true
		}
	}
	return false
}

func backupFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return os.WriteFile(path+".bak", data, 0644)
}
