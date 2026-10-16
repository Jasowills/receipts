package opencode

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func requireSQLite(t *testing.T) string {
	t.Helper()
	p, err := exec.LookPath("sqlite3")
	if err != nil {
		t.Skip("sqlite3 CLI required for adapter integration test")
	}
	return p
}

func buildTestDB(t *testing.T, workdir string) string {
	t.Helper()
	bin := requireSQLite(t)
	db := filepath.Join(t.TempDir(), "opencode.db")
	script := `
CREATE TABLE session (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, directory TEXT NOT NULL, title TEXT NOT NULL, time_created INTEGER NOT NULL, time_updated INTEGER NOT NULL);
CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, time_created INTEGER NOT NULL, time_updated INTEGER NOT NULL, data TEXT NOT NULL);
CREATE TABLE part (id TEXT PRIMARY KEY, message_id TEXT NOT NULL, session_id TEXT NOT NULL, time_created INTEGER NOT NULL, time_updated INTEGER NOT NULL, data TEXT NOT NULL);
CREATE TABLE project_directory (project_id TEXT NOT NULL, directory TEXT NOT NULL);
INSERT INTO session VALUES ('ses_test0000000000000001','prj_1','` + workdir + `','t',1760100001000,1760100009000);
INSERT INTO message VALUES ('msg_a','ses_test0000000000000001',1760100001000,1760100001000,'{"role":"assistant","time":{"created":1760100001000}}');
INSERT INTO part VALUES ('prt_a','msg_a','ses_test0000000000000001',1760100001000,1760100001000,'{"type":"text","text":"All tests pass"}');
`
	cmd := exec.Command(bin, db, script)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build db: %v\n%s", err, out)
	}
	return db
}

func TestDetect(t *testing.T) {
	ctx := context.Background()
	a := Adapter{DBPath: filepath.Join(t.TempDir(), "missing.db")}
	ok, err := a.Detect(ctx, "/tmp")
	if err != nil || ok {
		t.Errorf("Detect = %v, %v; want false, nil", ok, err)
	}
	a.DBPath = buildTestDB(t, "/work/proj")
	ok, err = a.Detect(ctx, "/work/proj")
	if err != nil || !ok {
		t.Errorf("Detect = %v, %v; want true, nil", ok, err)
	}
}

func TestLatestAndMessagesIntegration(t *testing.T) {
	requireSQLite(t)
	ctx := context.Background()
	workdir := "/work/proj"
	a := Adapter{DBPath: buildTestDB(t, workdir)}
	ref, err := a.LatestSession(ctx, workdir)
	if err != nil {
		t.Fatalf("LatestSession: %v", err)
	}
	if ref.ID != "ses_test0000000000000001" || ref.Agent != "opencode" {
		t.Errorf("ref = %+v", ref)
	}
	msgs, err := a.Messages(ctx, ref)
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Text != "All tests pass" {
		t.Fatalf("msgs = %+v", msgs)
	}
}

func TestLatestSessionNoMatch(t *testing.T) {
	requireSQLite(t)
	ctx := context.Background()
	a := Adapter{DBPath: buildTestDB(t, "/work/proj")}
	if _, err := a.LatestSession(ctx, "/somewhere/else"); err == nil {
		t.Errorf("expected error for unmatched directory, got nil")
	}
}

func TestMissingCLIClearError(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	a := Adapter{DBPath: filepath.Join(t.TempDir(), "x.db")}
	if _, err := a.query(context.Background(), "SELECT 1;"); err == nil {
		t.Errorf("expected clear error without sqlite3, got nil")
	} else if !strings.Contains(err.Error(), "sqlite3") {
		t.Errorf("error should mention sqlite3, got %q", err.Error())
	}
}

func TestReadOnlyDB(t *testing.T) {
	requireSQLite(t)
	db := buildTestDB(t, "/work/proj")
	if err := os.Chmod(db, 0444); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	a := Adapter{DBPath: db}
	ref, err := a.LatestSession(context.Background(), "/work/proj")
	if err != nil {
		t.Fatalf("LatestSession on read only db: %v", err)
	}
	if _, err := a.Messages(context.Background(), ref); err != nil {
		t.Fatalf("Messages on read only db: %v", err)
	}
}

func TestErrorsCarryNoSelfPrefix(t *testing.T) {
	a := Adapter{DBPath: filepath.Join(t.TempDir(), "missing.db")}
	_, err := a.LatestSession(context.Background(), "/work/proj")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if strings.Count(err.Error(), "opencode") > 0 {
		t.Errorf("adapter errors must not self prefix; callers add context. got %q", err.Error())
	}
}
