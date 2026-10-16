package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/receipts/receipts/internal/adapters"
)

func loadFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("fixtures", "session.jsonl"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return data
}

func TestParseFixture(t *testing.T) {
	msgs, err := parseJSONL(loadFixture(t))
	if err != nil {
		t.Fatalf("parseJSONL: %v", err)
	}
	if len(msgs) != 4 {
		t.Fatalf("got %d messages, want 4 (user, assistant, assistant-sidechain, system; orphan results and meta skipped)", len(msgs))
	}
	if msgs[0].Role != adapters.RoleUser || msgs[0].Text != "Please fix the login bug" {
		t.Errorf("msg0 = %+v", msgs[0])
	}
	a := msgs[1]
	if a.Role != adapters.RoleAssistant || a.Text != "All tests pass" {
		t.Errorf("assistant = %+v", a)
	}
	if len(a.ToolCalls) != 1 {
		t.Fatalf("tool calls = %d, want 1", len(a.ToolCalls))
	}
	tc := a.ToolCalls[0]
	if tc.Name != "Bash" || tc.Result != "ok example 0.1s" {
		t.Errorf("tool call = %+v", tc)
	}
	if cmd, _ := tc.Arguments["command"].(string); cmd != "go test ./..." {
		t.Errorf("args = %v", tc.Arguments)
	}
	if tc.ExitCode != nil {
		t.Errorf("ExitCode must be nil")
	}
	if a.ID != "u-000000000000000002" {
		t.Errorf("id = %q", a.ID)
	}
	wantTs := time.Date(2026, 8, 31, 21, 39, 40, 100*1e6, time.UTC)
	if !a.Timestamp.Equal(wantTs) {
		t.Errorf("ts = %v, want %v", a.Timestamp, wantTs)
	}
	if msgs[2].Metadata["sidechain"] != "true" {
		t.Errorf("sidechain tag missing: %+v", msgs[2].Metadata)
	}
	if msgs[3].Role != adapters.RoleSystem {
		t.Errorf("msg3 role = %q, want system", msgs[3].Role)
	}
}

func TestSlug(t *testing.T) {
	if slugForDir("/Users/jasonamadi/receipts") != "-Users-jasonamadi-receipts" {
		t.Errorf("got %q", slugForDir("/Users/jasonamadi/receipts"))
	}
}

func TestSizeBound(t *testing.T) {
	big := make([]byte, adapters.MaxTranscriptBytes+1)
	if _, err := parseJSONL(big); err == nil {
		t.Errorf("expected size bound error, got nil")
	}
}

func TestCorruptLineFails(t *testing.T) {
	if _, err := parseJSONL([]byte("{\"type\":\"user\"}\nnot json\n")); err == nil {
		t.Errorf("expected error for corrupt line, got nil")
	}
}

func TestTruncationCapReturnsPartial(t *testing.T) {
	var b strings.Builder
	for i := 0; i < adapters.MaxMessages+5; i++ {
		b.WriteString("{\"type\":\"user\",\"message\":{\"role\":\"user\",\"content\":\"hi\"},\"uuid\":\"u-")
		b.WriteString(itoa(i))
		b.WriteString("\",\"timestamp\":\"2026-08-31T21:39:38.950Z\"}\n")
	}
	msgs, err := parseJSONL([]byte(b.String()))
	if err == nil {
		t.Fatalf("expected cap error, got nil")
	}
	if len(msgs) != adapters.MaxMessages {
		t.Errorf("got %d partial messages, want %d", len(msgs), adapters.MaxMessages)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [32]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func TestLatestSessionSelection(t *testing.T) {
	dir := t.TempDir()
	proj := filepath.Join(dir, "projects", "-work-proj")
	if err := os.MkdirAll(proj, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	old := filepath.Join(proj, "old.jsonl")
	new := filepath.Join(proj, "new.jsonl")
	line := "{\"type\":\"user\",\"message\":{\"role\":\"user\",\"content\":\"hi\"},\"uuid\":\"u-1\",\"timestamp\":\"2026-08-31T21:39:38.950Z\",\"cwd\":\"/work/proj\"}\n"
	if err := os.WriteFile(old, []byte(line), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(new, []byte(line), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	past := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(old, past, past); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	a := Adapter{ProjectsRoot: filepath.Join(dir, "projects")}
	ref, err := a.LatestSession(t.Context(), "/work/proj")
	if err != nil {
		t.Fatalf("LatestSession: %v", err)
	}
	if !strings.HasSuffix(ref.Path, "new.jsonl") {
		t.Errorf("ref path = %q, want new.jsonl", ref.Path)
	}
	if ref.Agent != "claude" {
		t.Errorf("agent = %q", ref.Agent)
	}
}

func TestLatestSessionNoMatch(t *testing.T) {
	a := Adapter{ProjectsRoot: filepath.Join(t.TempDir(), "projects")}
	if _, err := a.LatestSession(t.Context(), "/work/proj"); err == nil {
		t.Errorf("expected error with no sessions, got nil")
	}
}

func TestDetect(t *testing.T) {
	a := Adapter{ProjectsRoot: filepath.Join(t.TempDir(), "none")}
	ok, err := a.Detect(t.Context(), "/work")
	if err != nil || ok {
		t.Errorf("Detect = %v, %v; want false, nil", ok, err)
	}
	if err := os.MkdirAll(a.ProjectsRoot, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	ok, err = a.Detect(t.Context(), "/work")
	if err != nil || !ok {
		t.Errorf("Detect = %v, %v; want true, nil", ok, err)
	}
}

func TestReadOnlyFile(t *testing.T) {
	dir := t.TempDir()
	dst := dir + "/ro.jsonl"
	data, err := loadFixtureBytes()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := os.WriteFile(dst, data, 0444); err != nil {
		t.Fatalf("write: %v", err)
	}
	a := Adapter{}
	msgs, err := a.Messages(t.Context(), adapters.SessionRef{Agent: "claude", ID: "x", Path: dst})
	if err != nil {
		t.Fatalf("read only file must still read, got %v", err)
	}
	if len(msgs) == 0 {
		t.Errorf("expected messages from read only file")
	}
}

func loadFixtureBytes() ([]byte, error) {
	return os.ReadFile("fixtures/session.jsonl")
}

func TestTimestampsUTC(t *testing.T) {
	msgs, err := parseJSONL(mustFixture(t))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, m := range msgs {
		if m.Timestamp.IsZero() {
			continue
		}
		if m.Timestamp.Location().String() != "UTC" {
			t.Errorf("msg %s timestamp not UTC: %v", m.ID, m.Timestamp)
		}
	}
}

func mustFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("fixtures/session.jsonl")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return data
}

func TestErrorsCarryNoSelfPrefix(t *testing.T) {
	a := Adapter{ProjectsRoot: filepath.Join(t.TempDir(), "none")}
	_, err := a.LatestSession(t.Context(), "/work/proj")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if strings.Count(err.Error(), "claude") > 0 {
		t.Errorf("adapter errors must not self prefix; callers add context. got %q", err.Error())
	}
}
