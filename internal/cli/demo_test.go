package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestDemoDefault(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := runDemo([]string{}, &out, &errBuf); code != 0 {
		t.Fatalf("exit = %d, stderr %q", code, errBuf.String())
	}
	s := out.String()
	for _, want := range []string{"VERIFIED", "CONTRADICTED", "UNVERIFIED", "RC-DE07"} {
		if !strings.Contains(s, want) {
			t.Errorf("demo output missing %q:\n%s", want, s)
		}
	}
}

func TestDemoBadScenario(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := runDemo([]string{"--scenario", "nope"}, &out, &errBuf); code != 4 {
		t.Errorf("exit = %d, want 4", code)
	}
}

func TestDemoViaRunDispatch(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := Run([]string{"demo", "--scenario", "scope-drift"}, &out, &errBuf); code != 0 {
		t.Fatalf("Run demo exit = %d, stderr %q", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "CONTRADICTED") {
		t.Errorf("scope drift should contradict, got:\n%s", out.String())
	}
}
