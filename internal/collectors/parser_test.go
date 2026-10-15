package collectors

import (
	"testing"

	"github.com/receipts/receipts/internal/collectors/shell"
)

func shimEvent(argv []string, exit int, out, errOut string) shell.CommandEvent {
	return shell.CommandEvent{
		ID: "cmd-1", Executable: argv[0], Argv: argv, Cwd: "/work",
		ExitCode: exit, StdoutTail: out, StderrTail: errOut,
		Capture: "shim", Complete: true,
	}
}

func TestParseGoTest(t *testing.T) {
	if p := ParseTestOutput([]string{"go", "test", "./..."}, "ok example 0.1s\n", ""); p == nil || !*p {
		t.Errorf("go ok should parse passed, got %v", p)
	}
	if p := ParseTestOutput([]string{"go", "test", "./..."}, "FAIL example [build failed]\n", ""); p == nil || *p {
		t.Errorf("go FAIL should parse failed")
	}
	if p := ParseTestOutput([]string{"go", "test", "./..."}, "some noise\n", ""); p != nil {
		t.Errorf("inconclusive output should return nil, got %v", *p)
	}
}

func TestParsePytest(t *testing.T) {
	if p := ParseTestOutput([]string{"pytest"}, "3 passed in 0.1s\n", ""); p == nil || !*p {
		t.Errorf("pytest passed should parse")
	}
	if p := ParseTestOutput([]string{"pytest", "-x"}, "1 failed, 2 passed\n", ""); p == nil || *p {
		t.Errorf("pytest failed should parse failed")
	}
}

func TestParseNpm(t *testing.T) {
	if p := ParseTestOutput([]string{"npm", "test"}, "10 passing\n", ""); p == nil || !*p {
		t.Errorf("npm passing should parse")
	}
	if p := ParseTestOutput([]string{"npm", "test"}, "2 failing\n", ""); p == nil || *p {
		t.Errorf("npm failing should parse failed")
	}
	if p := ParseTestOutput([]string{"npm", "run", "build"}, "built\n", ""); p != nil {
		t.Errorf("non test output should be nil, got %v", *p)
	}
}

func TestEvidenceForEvent(t *testing.T) {
	got := EvidenceForEvent(shimEvent([]string{"npm", "test"}, 0, "10 passing\n", ""), "E-001", "E-002")
	if len(got) != 2 {
		t.Fatalf("want command + test_result, got %d", len(got))
	}
	if got[0].Kind != "command" || got[1].Kind != "test_result" {
		t.Errorf("kinds = %s %s", got[0].Kind, got[1].Kind)
	}
	if got[0].Data["exit_code"] != float64(0) {
		t.Errorf("exit = %v", got[0].Data["exit_code"])
	}
	got = EvidenceForEvent(shimEvent([]string{"npm", "test"}, 0, "noise\n", ""), "E-001", "E-002")
	if len(got) != 1 {
		t.Errorf("inconclusive output should yield command only, got %d", len(got))
	}
}

func TestEvidenceArgvPreserved(t *testing.T) {
	got := EvidenceForEvent(shimEvent([]string{"go", "test", "./..."}, 1, "FAIL\n", ""), "E-009", "E-010")
	if got[0].ID != "E-009" || got[1].ID != "E-010" {
		t.Errorf("ids not preserved: %s %s", got[0].ID, got[1].ID)
	}
}
