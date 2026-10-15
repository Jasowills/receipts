//go:build unix

package shell

import (
	"os/exec"
	"syscall"
)

// exitStatus extracts the child exit code, mapping fatal signals to the
// conventional 128 plus signal number.
func exitStatus(ee *exec.ExitError) (int, bool) {
	if ws, ok := ee.Sys().(syscall.WaitStatus); ok {
		if ws.Signaled() {
			return 128 + int(ws.Signal()), true
		}
		return ws.ExitStatus(), true
	}
	return ee.ProcessState.ExitCode(), true
}
