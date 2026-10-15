package cli

import (
	"fmt"
	"io"
)

// Version is the CLI version. Override at build time with:
// go build -ldflags "-X github.com/receipts/receipts/internal/cli.Version=x.y.z"
var Version = "0.0.1-dev"

type commandInfo struct {
	name        string
	short       string
	use         string
	implemented bool
}

var commands = []commandInfo{
	{name: "audit", short: "Audit the latest agent session against Git evidence", use: auditUse, implemented: true},
	{name: "show", short: "Show a stored receipt", use: showUse, implemented: true},
	{name: "demo", short: "Run a deterministic offline demo scenario", use: demoUse, implemented: true},
	{name: "init", short: "Write starter config and gitignore entry", use: initUse, implemented: true},
	{name: "run", short: "Wrap an agent command with command capture", use: runUse, implemented: true},
	{name: "check", short: "Re-evaluate a stored receipt", use: checkUse, implemented: true},
	{name: "ci", short: "Print PR comment body from a stored receipt", use: ciUse, implemented: true},
}

func findCommand(name string) *commandInfo {
	for i := range commands {
		if commands[i].name == name {
			return &commands[i]
		}
	}
	return nil
}

func printRootHelp(w io.Writer) {
	fmt.Fprintln(w, "receipts - evidence based check of coding agent claims")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  receipts [--help] [--version] <command> [--help]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	for _, c := range commands {
		fmt.Fprintf(w, "  %-8s %s\n", c.name, c.short)
	}
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --help     Show help for receipts or a subcommand")
	fmt.Fprintln(w, "  --version  Print version information")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Status: all commands are implemented.")
	fmt.Fprintln(w, "Run `receipts <command> --help` for details.")
}

func printCommandHelp(w io.Writer, c *commandInfo) {
	if c.name == "show" {
		printShowHelp(w)
		return
	}
	if c.name == "audit" {
		printAuditHelp(w)
		return
	}
	if c.name == "init" {
		printInitHelp(w)
		return
	}
	if c.name == "demo" {
		printDemoHelp(w)
		return
	}
	if c.name == "run" {
		printRunHelp(w)
		return
	}
	if c.name == "check" {
		printCheckHelp(w)
		return
	}
	if c.name == "ci" {
		printCiHelp(w)
		return
	}
	fmt.Fprintf(w, "%s - %s\n", c.name, c.short)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintf(w, "  %s\n", c.use)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --help     Show help for this command")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Status: not yet implemented in this preview build.")
}

func printVersion(w io.Writer) {
	fmt.Fprintf(w, "receipts %s\n", Version)
}

// Run executes the CLI with the given args (excluding program name).
// It writes help and version output to stdout, errors to stderr,
// and returns the process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printRootHelp(stdout)
		return 0
	}

	first := args[0]

	switch first {
	case "--help", "-h", "help":
		if len(args) > 1 {
			if c := findCommand(args[1]); c != nil {
				printCommandHelp(stdout, c)
				return 0
			}
			fmt.Fprintf(stderr, "unknown command: %s\n", args[1])
			return 2
		}
		printRootHelp(stdout)
		return 0
	case "--version", "-v", "version":
		printVersion(stdout)
		return 0
	}

	if len(first) > 0 && first[0] == '-' {
		fmt.Fprintf(stderr, "unknown flag: %s\n", first)
		fmt.Fprintln(stderr, "Run receipts --help for usage.")
		return 2
	}

	if c := findCommand(first); c != nil {
		if c.name == "show" {
			for _, a := range args[1:] {
				if a == "--help" || a == "-h" {
					printShowHelp(stdout)
					return 0
				}
			}
			return runShow(args[1:], stdout, stderr)
		}
		if c.name == "audit" {
			for _, a := range args[1:] {
				if a == "--help" || a == "-h" {
					printAuditHelp(stdout)
					return 0
				}
			}
			return runAudit(args[1:], stdout, stderr)
		}
		if c.name == "init" {
			for _, a := range args[1:] {
				if a == "--help" || a == "-h" {
					printInitHelp(stdout)
					return 0
				}
			}
			return runInit(args[1:], stdout, stderr)
		}
		if c.name == "demo" {
			for _, a := range args[1:] {
				if a == "--help" || a == "-h" {
					printDemoHelp(stdout)
					return 0
				}
			}
			return runDemo(args[1:], stdout, stderr)
		}
		if c.name == "run" {
			for _, a := range args[1:] {
				if a == "--" {
					break
				}
				if a == "--help" || a == "-h" {
					printRunHelp(stdout)
					return 0
				}
			}
			return runRun(args[1:], stdout, stderr)
		}
		if c.name == "check" {
			for _, a := range args[1:] {
				if a == "--help" || a == "-h" {
					printCheckHelp(stdout)
					return 0
				}
			}
			return runCheck(args[1:], stdout, stderr)
		}
		if c.name == "ci" {
			for _, a := range args[1:] {
				if a == "--help" || a == "-h" {
					printCiHelp(stdout)
					return 0
				}
			}
			return runCi(args[1:], stdout, stderr)
		}
		for _, a := range args[1:] {
			if a == "--help" || a == "-h" {
				printCommandHelp(stdout, c)
				return 0
			}
		}
		printCommandHelp(stdout, c)
		return 0
	}

	fmt.Fprintf(stderr, "unknown command: %s\n", first)
	fmt.Fprintln(stderr, "Run receipts --help for usage.")
	return 2
}
