package collectors

import (
	"time"

	"github.com/receipts/receipts/internal/collectors/shell"
	"github.com/receipts/receipts/internal/model"
)

// EvidenceForEvent converts one shim record into command evidence plus,
// when output parsing is conclusive, a test_result evidence item. IDs
// come from the caller to keep receipt numbering deterministic.
func EvidenceForEvent(e shell.CommandEvent, cmdID, resultID string) []model.Evidence {
	started, _ := time.Parse(time.RFC3339Nano, e.Started)
	ended, _ := time.Parse(time.RFC3339Nano, e.Ended)
	out := []model.Evidence{{
		ID: cmdID, Kind: model.EvidenceCommand, Timestamp: ended.UTC(),
		Data: map[string]any{
			"argv": e.Argv, "executable": e.Executable, "cwd": e.Cwd,
			"started":     started.UTC().Format(time.RFC3339Nano),
			"duration_ms": e.DurationMS, "exit_code": float64(e.ExitCode),
			"stdout_hash": e.StdoutHash, "stderr_hash": e.StderrHash,
			"stdout_tail": e.StdoutTail, "stderr_tail": e.StderrTail,
			"capture": e.Capture, "complete": e.Complete,
		},
	}}
	if passed := ParseTestOutput(e.Argv, e.StdoutTail, e.StderrTail); passed != nil {
		out = append(out, model.Evidence{
			ID: resultID, Kind: model.EvidenceTestResult, Timestamp: ended.UTC(),
			Data: map[string]any{"passed": *passed},
		})
	}
	return out
}
