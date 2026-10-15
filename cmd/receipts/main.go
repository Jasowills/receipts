package main

import (
	"os"
	"path/filepath"

	"github.com/receipts/receipts/internal/cli"
	"github.com/receipts/receipts/internal/collectors/shell"
)

func main() {
	// Shim mode: this binary was invoked under a wrapped command name
	// through a PATH symlink. Never run CLI parsing in that path.
	if shell.IsShimArgv0(filepath.Base(os.Args[0])) {
		args := []string{}
		if len(os.Args) > 1 {
			args = os.Args[1:]
		}
		os.Exit(shell.Main(os.Args[0], args, os.Stdin, os.Stdout, os.Stderr))
	}
	code := cli.Run(os.Args[1:], os.Stdout, os.Stderr)
	os.Exit(code)
}
