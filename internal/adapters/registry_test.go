package adapters

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeAdapter struct {
	name    string
	detect  bool
	derr    error
	ref     SessionRef
	lerr    error
	detectW string
}

func (f fakeAdapter) Name() string { return f.name }
func (f fakeAdapter) Detect(_ context.Context, _ string) (bool, error) {
	return f.detect, f.derr
}
func (f fakeAdapter) LatestSession(_ context.Context, _ string) (SessionRef, error) {
	if f.lerr != nil {
		return SessionRef{}, f.lerr
	}
	return f.ref, nil
}
func (f fakeAdapter) Messages(_ context.Context, _ SessionRef) ([]Message, error) {
	return nil, nil
}

func TestAutoPicksFirstSuccess(t *testing.T) {
	a := fakeAdapter{name: "opencode", detect: true, ref: SessionRef{Agent: "opencode", ID: "s-1"}}
	b := fakeAdapter{name: "claude", detect: true, ref: SessionRef{Agent: "claude", ID: "s-2"}}
	res, err := Auto(context.Background(), "/work", []Adapter{a, b})
	if err != nil {
		t.Fatalf("Auto: %v", err)
	}
	if res.Ref.ID != "s-1" || res.Adapter.Name() != "opencode" {
		t.Errorf("res = %+v", res.Ref)
	}
	if len(res.Attempts) != 1 || res.Attempts[0].SessionID != "s-1" {
		t.Errorf("attempts = %+v, want single success", res.Attempts)
	}
}

func TestAutoSkipsUndetectedAndFailed(t *testing.T) {
	a := fakeAdapter{name: "opencode", detect: false}
	b := fakeAdapter{name: "claude", detect: true, lerr: errors.New("no sessions here")}
	c := fakeAdapter{name: "codex", detect: true, ref: SessionRef{Agent: "codex", ID: "s-3"}}
	res, err := Auto(context.Background(), "/work", []Adapter{a, b, c})
	if err != nil {
		t.Fatalf("Auto: %v", err)
	}
	if res.Ref.ID != "s-3" {
		t.Errorf("ref = %+v, want s-3", res.Ref)
	}
	if len(res.Attempts) != 3 {
		t.Fatalf("attempts = %+v, want all three recorded", res.Attempts)
	}
	if res.Attempts[0].Detected || res.Attempts[1].SessionID != "" {
		t.Errorf("attempt flags wrong: %+v", res.Attempts)
	}
}

func TestAutoAllFailListsReasons(t *testing.T) {
	a := fakeAdapter{name: "opencode", detect: false}
	_, err := Auto(context.Background(), "/work", []Adapter{a})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "opencode") || !strings.Contains(got, "not detected") {
		t.Errorf("error should name adapter and reason, got %q", got)
	}
}

func TestAutoEmptyCandidates(t *testing.T) {
	if _, err := Auto(context.Background(), "/work", nil); err == nil {
		t.Errorf("expected error for no candidates, got nil")
	}
}

func TestAutoRespectsCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	a := fakeAdapter{name: "opencode", detect: true, ref: SessionRef{ID: "s-1"}}
	if _, err := Auto(ctx, "/work", []Adapter{a}); err == nil {
		t.Errorf("expected cancel error, got nil")
	}
}
