package shell

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeScript(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body), 0755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return p
}

func TestIsShimArgv0(t *testing.T) {
	for _, n := range []string{"npm", "go", "pytest", "eslint", "/tmp/bin/npm"} {
		if !IsShimArgv0(n) {
			t.Errorf("%q should be a shim name", n)
		}
	}
	for _, n := range []string{"receipts", "sh", "node", "python3.11"} {
		if IsShimArgv0(n) {
			t.Errorf("%q should not be a shim name", n)
		}
	}
}

func TestResolveSkipsShimDir(t *testing.T) {
	shimDir := t.TempDir()
	realDir := t.TempDir()
	writeScript(t, realDir, "npm", "echo real-npm\n")
	// Trap executable in the shim dir must be skipped to avoid recursion.
	writeScript(t, shimDir, "npm", "echo TRAP; exit 99\n")
	t.Setenv(EnvRealPath, realDir+":"+shimDir)
	t.Setenv(EnvShimDir, shimDir)
	got, err := ResolveReal("npm")
	if err != nil {
		t.Fatalf("ResolveReal: %v", err)
	}
	if got != filepath.Join(realDir, "npm") {
		t.Errorf("got %q, want real dir script", got)
	}
}

func TestResolveMissing(t *testing.T) {
	t.Setenv(EnvRealPath, t.TempDir())
	t.Setenv(EnvShimDir, t.TempDir())
	if _, err := ResolveReal("no-such-tool-xyz"); err == nil {
		t.Errorf("expected error for missing executable, got nil")
	}
}

func TestRunShimRecordsAndForwards(t *testing.T) {
	realDir := t.TempDir()
	runDir := t.TempDir()
	writeScript(t, realDir, "npm", "echo out-line; echo err-line >&2; echo \"args:$@\"\n")
	t.Setenv(EnvRealPath, realDir)
	t.Setenv(EnvShimDir, t.TempDir())
	t.Setenv(EnvRunDir, runDir)
	var stdout, stderr bytes.Buffer
	code := Main("npm", []string{"test", "--silent"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "out-line") || !strings.Contains(stdout.String(), "args:test --silent") {
		t.Errorf("stdout not forwarded: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "err-line") {
		t.Errorf("stderr not forwarded: %q", stderr.String())
	}
	events, err := ReadEvents(runDir)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	e := events[0]
	if len(e.Argv) != 3 || e.Argv[1] != "test" {
		t.Errorf("argv = %v", e.Argv)
	}
	if e.ExitCode != 0 || !e.Complete || e.Capture != "shim" {
		t.Errorf("event = %+v", e)
	}
	if e.StdoutHash == "" || e.StderrHash == "" {
		t.Errorf("hashes must be set")
	}
	if !strings.Contains(e.StdoutTail, "out-line") {
		t.Errorf("tail missing output: %q", e.StdoutTail)
	}
}

func TestRunShimExitCode(t *testing.T) {
	realDir := t.TempDir()
	runDir := t.TempDir()
	writeScript(t, realDir, "go", "exit 3\n")
	t.Setenv(EnvRealPath, realDir)
	t.Setenv(EnvShimDir, t.TempDir())
	t.Setenv(EnvRunDir, runDir)
	var stdout, stderr bytes.Buffer
	if code := Main("go", []string{"test"}, strings.NewReader(""), &stdout, &stderr); code != 3 {
		t.Errorf("exit = %d, want 3", code)
	}
	events, err := ReadEvents(runDir)
	if err != nil || len(events) != 1 || events[0].ExitCode != 3 {
		t.Errorf("events = %+v err = %v", events, err)
	}
}

func TestRunShimMissingRealIs127(t *testing.T) {
	runDir := t.TempDir()
	t.Setenv(EnvRealPath, t.TempDir())
	t.Setenv(EnvShimDir, t.TempDir())
	t.Setenv(EnvRunDir, runDir)
	var stdout, stderr bytes.Buffer
	if code := Main("npm", []string{"test"}, strings.NewReader(""), &stdout, &stderr); code != 127 {
		t.Errorf("exit = %d, want 127", code)
	}
	events, err := ReadEvents(runDir)
	if err != nil || len(events) != 1 || events[0].ExitCode != 127 {
		t.Errorf("resolution failure must still record an event: %+v %v", events, err)
	}
}

func TestRunShimNoRunDirStillRuns(t *testing.T) {
	realDir := t.TempDir()
	writeScript(t, realDir, "npm", "echo hi\n")
	t.Setenv(EnvRealPath, realDir)
	t.Setenv(EnvShimDir, t.TempDir())
	t.Setenv(EnvRunDir, filepath.Join(t.TempDir(), "missing", "dir"))
	var stdout, stderr bytes.Buffer
	if code := Main("npm", []string{"x"}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Errorf("shim must not break the command when collection is down, exit = %d", code)
	}
	if !strings.Contains(stdout.String(), "hi") {
		t.Errorf("output must forward: %q", stdout.String())
	}
}

func TestTailBoundedAndHashed(t *testing.T) {
	realDir := t.TempDir()
	runDir := t.TempDir()
	// 100 KiB of output: tail must cap at 8 KiB, hash must cover it all.
	writeScript(t, realDir, "npm", "yes x | head -c 102400\n")
	t.Setenv(EnvRealPath, realDir)
	t.Setenv(EnvShimDir, t.TempDir())
	t.Setenv(EnvRunDir, runDir)
	var stdout, stderr bytes.Buffer
	ctx := context.Background()
	_ = ctx
	if code := Main("npm", []string{"x"}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	events, err := ReadEvents(runDir)
	if err != nil || len(events) != 1 {
		t.Fatalf("events: %+v %v", events, err)
	}
	if len(events[0].StdoutTail) > TailBytes {
		t.Errorf("tail len = %d, want <= %d", len(events[0].StdoutTail), TailBytes)
	}
	if events[0].StdoutHash == "" {
		t.Errorf("hash must cover full output")
	}
	if stdout.Len() < 100000 {
		t.Errorf("forwarded output truncated: %d bytes", stdout.Len())
	}
}
