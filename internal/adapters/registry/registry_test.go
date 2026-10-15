package registry

import (
	"context"
	"strings"
	"testing"
)

func TestDefaultOrder(t *testing.T) {
	got := []string{}
	for _, a := range Default() {
		got = append(got, a.Name())
	}
	if len(got) != 2 || got[0] != "opencode" || got[1] != "claude" {
		t.Errorf("default order = %v, want [opencode claude]", got)
	}
}

func TestByName(t *testing.T) {
	for _, n := range []string{"opencode", "claude"} {
		a, err := ByName(n)
		if err != nil || a.Name() != n {
			t.Errorf("ByName(%q) = %v, %v", n, a, err)
		}
	}
	if _, err := ByName("codex"); err == nil {
		t.Errorf("codex should be unknown until Phase G verification")
	}
}

func TestAutoUsesDefaults(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	_, err := Auto(context.Background(), "/work/proj")
	if err == nil {
		t.Fatalf("expected error with empty HOME, got nil")
	}
	for _, n := range []string{"opencode", "claude"} {
		if !strings.Contains(err.Error(), n) {
			t.Errorf("auto error should list %q, got %q", n, err.Error())
		}
	}
}
