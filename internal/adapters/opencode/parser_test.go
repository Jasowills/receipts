package opencode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/receipts/receipts/internal/adapters"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("fixtures", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func TestSelectLatestSession(t *testing.T) {
	sessions, err := parseSessionRows(loadFixture(t, "session_rows.json"))
	if err != nil {
		t.Fatalf("parseSessionRows: %v", err)
	}
	latest, err := selectLatest(sessions)
	if err != nil {
		t.Fatalf("selectLatest: %v", err)
	}
	if latest.ID != "ses_new000000000000000002" {
		t.Errorf("latest = %q, want ses_new", latest.ID)
	}
	if _, err := selectLatest(nil); err == nil {
		t.Errorf("expected error for empty sessions, got nil")
	}
}

func TestParseMessageRows(t *testing.T) {
	msgs, err := parseMessageRows(loadFixture(t, "message_rows.json"))
	if err != nil {
		t.Fatalf("parseMessageRows: %v", err)
	}
	if len(msgs) != 3 {
		t.Fatalf("got %d messages, want 3", len(msgs))
	}
	u := msgs[0]
	if u.Role != adapters.RoleUser || u.Text != "What is the status?" {
		t.Errorf("user msg = %+v, want role user with single text (ignored part skipped)", u)
	}
	if !u.Timestamp.Equal(time.UnixMilli(1760100001000).UTC()) {
		t.Errorf("user ts = %v", u.Timestamp)
	}
	a := msgs[1]
	if a.Role != adapters.RoleAssistant || a.Text != "All tests pass" {
		t.Errorf("assistant msg = %+v", a)
	}
	if len(a.ToolCalls) != 2 {
		t.Fatalf("tool calls = %d, want 2 (completed + error; running skipped)", len(a.ToolCalls))
	}
	bash := a.ToolCalls[0]
	if bash.Name != "bash" || bash.Result != "ok example 0.1s" {
		t.Errorf("bash call = %+v", bash)
	}
	if cmd, _ := bash.Arguments["command"].(string); cmd != "go test ./..." {
		t.Errorf("bash args = %v", bash.Arguments)
	}
	if bash.ExitCode != nil {
		t.Errorf("ExitCode should be nil for transcripts")
	}
	rd := a.ToolCalls[1]
	if rd.Name != "read" || !strings.Contains(rd.Result, "file not found") {
		t.Errorf("error call = %+v", rd)
	}
	if msgs[2].Text != "" || msgs[2].Role != adapters.RoleUser {
		t.Errorf("partless message = %+v, want empty user text", msgs[2])
	}
}

func TestParseMessageRowsRejectsHuge(t *testing.T) {
	rows := make([]messageRow, adapters.MaxMessages+1)
	for i := range rows {
		rows[i] = messageRow{MID: "m-" + itoa(i), MTime: 1, MData: `{"role":"user"}`}
	}
	if _, err := messagesFromRows(rows); err == nil {
		t.Errorf("expected error past MaxMessages, got nil")
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

func TestEscapeSQL(t *testing.T) {
	if escapeSQL("a'b") != "a''b" {
		t.Errorf("quotes not doubled")
	}
}

func TestResolveDBPath(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg")
	t.Setenv("OPENCODE_DB", "")
	home := "/tmp/home"
	got := resolveDBPath(func(k string) string {
		if k == "XDG_DATA_HOME" {
			return "/tmp/xdg"
		}
		if k == "OPENCODE_DB" {
			return ""
		}
		return ""
	}, home)
	if got != "/tmp/xdg/opencode/opencode.db" {
		t.Errorf("got %q", got)
	}
}

func TestTimestampsUTC(t *testing.T) {
	msgs, err := parseMessageRows(loadFixture(t, "message_rows.json"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, m := range msgs {
		if m.Timestamp.IsZero() {
			t.Errorf("msg %s missing timestamp", m.ID)
			continue
		}
		if m.Timestamp.Location().String() != "UTC" {
			t.Errorf("msg %s timestamp not UTC: %v", m.ID, m.Timestamp)
		}
	}
}
