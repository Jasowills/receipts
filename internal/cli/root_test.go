package cli

import (
	"bytes"
	"strings"
	"testing"
)

func runWith(args []string) (stdout, stderr string, code int) {
	var outBuf, errBuf bytes.Buffer
	code = Run(args, &outBuf, &errBuf)
	return outBuf.String(), errBuf.String(), code
}

func TestRootHelpListsCommands(t *testing.T) {
	stdout, _, code := runWith([]string{"--help"})
	if code != 0 {
		t.Fatalf("expected exit 0 for --help, got %d", code)
	}
	for _, cmd := range []string{"audit", "show", "demo", "init"} {
		if !strings.Contains(stdout, cmd) {
			t.Errorf("help output missing command %q, got:\n%s", cmd, stdout)
		}
	}
}

func TestRootNoArgsPrintsHelp(t *testing.T) {
	stdout, _, code := runWith([]string{})
	if code != 0 {
		t.Fatalf("expected exit 0 for no args, got %d", code)
	}
	if !strings.Contains(stdout, "Usage") {
		t.Errorf("expected Usage in output, got:\n%s", stdout)
	}
}

func TestVersionFlag(t *testing.T) {
	stdout, _, code := runWith([]string{"--version"})
	if code != 0 {
		t.Fatalf("expected exit 0 for --version, got %d", code)
	}
	if !strings.Contains(stdout, Version) {
		t.Errorf("expected version %q in output, got:\n%s", Version, stdout)
	}
}

func TestEveryCommandSupportsHelp(t *testing.T) {
	commands := []string{"audit", "show", "demo", "init", "run", "check", "ci"}
	for _, cmd := range commands {
		stdout, _, code := runWith([]string{cmd, "--help"})
		if code != 0 {
			t.Errorf("command %q --help: expected exit 0, got %d", cmd, code)
			continue
		}
		if !strings.Contains(stdout, "Usage") {
			t.Errorf("command %q --help: expected Usage in output, got:\n%s", cmd, stdout)
		}
	}
}

func TestUnknownCommandFailsWithUsageError(t *testing.T) {
	_, _, code := runWith([]string{"does-not-exist"})
	if code != 2 {
		t.Fatalf("expected exit 2 for unknown command, got %d", code)
	}
}

func TestHelpHasNoEmDash(t *testing.T) {
	stdout, _, _ := runWith([]string{"--help"})
	if strings.Contains(stdout, "—") {
		t.Errorf("help output must not contain em dash character")
	}
}
