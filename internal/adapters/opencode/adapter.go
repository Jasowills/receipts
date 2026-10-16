package opencode

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/receipts/receipts/internal/adapters"
)

// Adapter reads OpenCode transcripts from its SQLite store read only.
// It shells out to the sqlite3 CLI so the core keeps zero dependencies.
// The database is opened read only; uncommitted OpenCode writes stay
// invisible until they commit (WAL readers see committed state).
type Adapter struct {
	// DBPath overrides discovery. Empty means resolve from
	// OPENCODE_DB, XDG_DATA_HOME, or the home directory.
	DBPath string
}

// Name returns the adapter name.
func (Adapter) Name() string { return "opencode" }

func resolveDBPath(getenv func(string) string, home string) string {
	if v := getenv("OPENCODE_DB"); v != "" {
		if v == ":memory:" || filepath.IsAbs(v) {
			return v
		}
		return filepath.Join(dataDir(getenv, home), v)
	}
	return filepath.Join(dataDir(getenv, home), "opencode.db")
}

func dataDir(getenv func(string) string, home string) string {
	if v := getenv("XDG_DATA_HOME"); v != "" {
		return filepath.Join(v, "opencode")
	}
	return filepath.Join(home, ".local", "share", "opencode")
}

func (a Adapter) dbPath() string {
	if a.DBPath != "" {
		return a.DBPath
	}
	home, _ := os.UserHomeDir()
	return resolveDBPath(os.Getenv, home)
}

// Detect reports whether an OpenCode database exists.
func (a Adapter) Detect(_ context.Context, _ string) (bool, error) {
	fi, err := os.Stat(a.dbPath())
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return !fi.IsDir(), nil
}

func sqlite3bin() (string, error) {
	p, err := exec.LookPath("sqlite3")
	if err != nil {
		return "", fmt.Errorf("sqlite3 CLI not found in PATH (required to read the session database read only)")
	}
	return p, nil
}

// query runs one read only query and returns -json output.
func (a Adapter) query(ctx context.Context, sql string) ([]byte, error) {
	bin, err := sqlite3bin()
	if err != nil {
		return nil, err
	}
	uri := "file:" + a.dbPath() + "?mode=ro"
	var stdout, stderr bytes.Buffer
	// Retry briefly: the live database uses WAL with a single writer.
	var last error
	for i := 0; i < 5; i++ {
		stdout.Reset()
		stderr.Reset()
		cmd := exec.CommandContext(ctx, bin, "-json", uri, sql)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		last = cmd.Run()
		if last == nil {
			out := stdout.Bytes()
			if int64(len(out)) > adapters.MaxTranscriptBytes {
				return nil, fmt.Errorf("query output exceeds %d bytes", adapters.MaxTranscriptBytes)
			}
			return out, nil
		}
		if ctx.Err() != nil {
			return nil, fmt.Errorf("query canceled: %w", ctx.Err())
		}
		time.Sleep(50 * time.Millisecond)
	}
	msg := strings.TrimSpace(stderr.String())
	if len(msg) > 300 {
		msg = msg[:300]
	}
	if msg == "" {
		msg = last.Error()
	}
	return nil, fmt.Errorf("sqlite3 query failed: %s", msg)
}

// LatestSession returns the most recently updated session for workdir,
// matching session.directory directly or via project_directory.
func (a Adapter) LatestSession(ctx context.Context, workdir string) (adapters.SessionRef, error) {
	abs, err := filepath.Abs(workdir)
	if err != nil {
		return adapters.SessionRef{}, err
	}
	sql := fmt.Sprintf(`SELECT id, directory, title, time_updated FROM session WHERE directory = '%s' OR project_id IN (SELECT project_id FROM project_directory WHERE directory = '%s') ORDER BY time_updated DESC LIMIT 1;`,
		escapeSQL(abs), escapeSQL(abs))
	out, err := a.query(ctx, sql)
	if err != nil {
		return adapters.SessionRef{}, err
	}
	rows, err := parseSessionRows(out)
	if err != nil {
		return adapters.SessionRef{}, err
	}
	sel, err := selectLatest(rows)
	if err != nil {
		return adapters.SessionRef{}, fmt.Errorf("%w (no session recorded for %s)", err, abs)
	}
	fi, _ := os.Stat(a.dbPath())
	var mod time.Time
	if fi != nil {
		mod = fi.ModTime()
	}
	return adapters.SessionRef{Agent: "opencode", ID: sel.ID, Path: a.dbPath(), ModifiedAt: mod}, nil
}

// Messages returns normalized messages for one session.
func (a Adapter) Messages(ctx context.Context, session adapters.SessionRef) ([]adapters.Message, error) {
	sql := fmt.Sprintf(`SELECT m.id AS mid, m.time_created AS mtime, m.data AS mdata, p.time_created AS ptime, p.data AS pdata FROM message m LEFT JOIN part p ON p.message_id = m.id WHERE m.session_id = '%s' ORDER BY m.time_created ASC, m.id ASC, p.time_created ASC, p.id ASC;`,
		escapeSQL(session.ID))
	out, err := a.query(ctx, sql)
	if err != nil {
		return nil, err
	}
	msgs, err := parseMessageRows(out)
	if err != nil {
		return nil, err
	}
	for i := range msgs {
		if msgs[i].Metadata == nil {
			msgs[i].Metadata = map[string]string{}
		}
		msgs[i].Metadata["session_id"] = session.ID
	}
	return msgs, nil
}
