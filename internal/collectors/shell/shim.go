package shell

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/receipts/receipts/internal/integrity"
)

// Shim environment protocol. The run command sets these for wrapped
// children; shims never edit the user's shell profile.
const (
	EnvRunDir   = "RECEIPTS_RUN_DIR"
	EnvRealPath = "RECEIPTS_REAL_PATH"
	EnvShimDir  = "RECEIPTS_SHIM_DIR"
)

// TailBytes caps stored output per stream. Hashes cover all raw bytes.
const TailBytes = 8192

// Supported lists shimmed executable basenames. Bare `python -m pytest`
// still observes through the python shim with argv matching downstream.
var Supported = []string{
	"npm", "pnpm", "yarn", "bun", "pytest", "python", "python3",
	"go", "cargo", "make", "just", "eslint", "tsc", "vitest",
	"jest", "playwright", "dotnet",
}

// IsShimArgv0 reports whether argv0 names a shimmed executable.
func IsShimArgv0(argv0 string) bool {
	base := argv0
	if i := strings.LastIndexByte(base, '/'); i >= 0 {
		base = base[i+1:]
	}
	for _, n := range Supported {
		if base == n {
			return true
		}
	}
	return false
}

// ResolveReal finds the real executable using the pre shim PATH,
// skipping the shim directory to avoid recursion.
func ResolveReal(name string) (string, error) {
	realPath := os.Getenv(EnvRealPath)
	if realPath == "" {
		realPath = os.Getenv("PATH")
	}
	shimDir := os.Getenv(EnvShimDir)
	for _, dir := range strings.Split(realPath, ":") {
		if dir == "" || dir == shimDir {
			continue
		}
		candidate := filepath.Join(dir, name)
		fi, err := os.Stat(candidate)
		if err != nil || fi.IsDir() {
			continue
		}
		if fi.Mode().Perm()&0111 == 0 {
			continue
		}
		return candidate, nil
	}
	return "", fmt.Errorf("real executable %q not found", name)
}

// CommandEvent is one shim observed command record.
type CommandEvent struct {
	ID         string   `json:"id"`
	Executable string   `json:"executable"`
	Argv       []string `json:"argv"`
	Cwd        string   `json:"cwd"`
	Started    string   `json:"started"`
	Ended      string   `json:"ended"`
	DurationMS int64    `json:"duration_ms"`
	ExitCode   int      `json:"exit_code"`
	StdoutHash string   `json:"stdout_hash"`
	StderrHash string   `json:"stderr_hash"`
	StdoutTail string   `json:"stdout_tail"`
	StderrTail string   `json:"stderr_tail"`
	Capture    string   `json:"capture"`
	Complete   bool     `json:"complete"`
	Error      string   `json:"error,omitempty"`
}

// eventsPath returns the run commands file.
func eventsPath(runDir string) string { return filepath.Join(runDir, "commands.jsonl") }

// AppendEvent appends one event line under a run directory lock.
func AppendEvent(runDir string, ev CommandEvent) error {
	if runDir == "" {
		return fmt.Errorf("shell: empty run directory")
	}
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return err
	}
	lock := filepath.Join(runDir, ".lock")
	if err := withLockFile(lock, func() error {
		f, err := os.OpenFile(eventsPath(runDir), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		b, err := json.Marshal(ev)
		if err != nil {
			return err
		}
		if _, err := f.Write(append(b, '\n')); err != nil {
			return err
		}
		return f.Sync()
	}); err != nil {
		return err
	}
	return nil
}

// ReadEvents returns all command events in file order. Corrupt lines
// fail the read rather than silently dropping evidence.
func ReadEvents(runDir string) ([]CommandEvent, error) {
	data, err := os.ReadFile(eventsPath(runDir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []CommandEvent
	for i, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var e CommandEvent
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("shell: corrupt command event line %d: %w", i+1, err)
		}
		out = append(out, e)
	}
	return out, nil
}

func withLockFile(path string, fn func() error) error {
	const tries = 200
	for i := 0; i < tries; i++ {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err == nil {
			f.Close()
			defer os.Remove(path)
			return fn()
		}
		if !os.IsExist(err) {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("shell: lock busy %s", path)
}

// tailWriter keeps the last N bytes written.
type tailWriter struct {
	limit int
	buf   []byte
}

func (w *tailWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	if len(w.buf) > w.limit {
		w.buf = append([]byte{}, w.buf[len(w.buf)-w.limit:]...)
	}
	return len(p), nil
}

func (w *tailWriter) String() string { return string(w.buf) }

// Main runs shimmed execution: resolve, stream with capture, record,
// and exit with the real command's code (127 when unresolvable).
// Output always forwards; a broken collector never breaks the command.
func Main(argv0 string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	name := argv0
	if i := strings.LastIndexByte(name, '/'); i >= 0 {
		name = name[i+1:]
	}
	start := time.Now().UTC()
	cwd, _ := os.Getwd()
	ev := CommandEvent{
		ID:         fmt.Sprintf("cmd-%d-%d", os.Getpid(), start.UnixNano()),
		Executable: name,
		Argv:       append([]string{name}, args...),
		Cwd:        cwd,
		Started:    start.Format(time.RFC3339Nano),
		Capture:    "shim",
	}
	real, err := ResolveReal(name)
	if err != nil {
		finishEvent(&ev, start, 127, "", "", "", "", err.Error())
		_ = recordBestEffort(&ev)
		fmt.Fprintf(stderr, "receipts shim: %v\n", err)
		return 127
	}
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, real, args...)
	cmd.Stdin = stdin
	cmd.Env = os.Environ()
	var outHash, errHash = sha256.New(), sha256.New()
	var outTail, errTail = &tailWriter{limit: TailBytes}, &tailWriter{limit: TailBytes}
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return failEvent(&ev, start, fmt.Sprintf("stdout pipe: %v", err), stderr)
	}
	errPipe, err := cmd.StderrPipe()
	if err != nil {
		return failEvent(&ev, start, fmt.Sprintf("stderr pipe: %v", err), stderr)
	}
	if err := cmd.Start(); err != nil {
		return failEvent(&ev, start, fmt.Sprintf("start: %v", err), stderr)
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(io.MultiWriter(outHash, outTail, stdout), outPipe)
	}()
	go func() {
		defer wg.Done()
		io.Copy(io.MultiWriter(errHash, errTail, stderr), errPipe)
	}()
	wg.Wait()
	runErr := cmd.Wait()
	exit := exitCodeOf(runErr)
	finishEvent(&ev, start, exit,
		hexOf(outHash), hexOf(errHash),
		integrity.RedactSecrets(outTail.String()), integrity.RedactSecrets(errTail.String()), "")
	if rerr := recordBestEffort(&ev); rerr != nil {
		fmt.Fprintf(stderr, "receipts shim: capture degraded: %v\n", rerr)
	}
	return exit
}

func hexOf(h interface{ Sum([]byte) []byte }) string {
	return hex.EncodeToString(h.Sum(nil))
}

func exitCodeOf(err error) int {
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		if ws, ok := exitStatus(ee); ok {
			return ws
		}
		return 1
	}
	return 1
}

func finishEvent(ev *CommandEvent, start time.Time, exit int, outHash, errHash, outTail, errTail, errmsg string) {
	end := time.Now().UTC()
	ev.Ended = end.Format(time.RFC3339Nano)
	ev.DurationMS = end.Sub(start).Milliseconds()
	ev.ExitCode = exit
	ev.StdoutHash = outHash
	ev.StderrHash = errHash
	ev.StdoutTail = outTail
	ev.StderrTail = errTail
	ev.Complete = true
	ev.Error = errmsg
}

func failEvent(ev *CommandEvent, start time.Time, msg string, stderr io.Writer) int {
	finishEvent(ev, start, 1, "", "", "", "", msg)
	_ = recordBestEffort(ev)
	fmt.Fprintf(stderr, "receipts shim: %s\n", msg)
	return 1
}

// recordBestEffort appends the event unless collection is unavailable,
// in which case the command still runs to completion.
func recordBestEffort(ev *CommandEvent) error {
	runDir := os.Getenv(EnvRunDir)
	if runDir == "" {
		return fmt.Errorf("no run directory")
	}
	return AppendEvent(runDir, *ev)
}
