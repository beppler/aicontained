//go:build windows

package launcher

import (
	"errors"
	"os"
	"os/exec"
)

// runChild starts the engine as a child process inheriting the console.
// No process-group setup: the OS delivers Ctrl+C to every process in the
// shared console group, and the exit code comes back from the child.
func runChild(name string, args []string) (int, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	if _, ok := errors.AsType[*exec.ExitError](err); !ok {
		return exitCode(cmd.ProcessState), err
	}
	// The child started and ran to completion: its exit code is the
	// result, not an error (Go reports every non-zero exit as ExitError).
	return exitCode(cmd.ProcessState), nil
}

func exitCode(ps *os.ProcessState) int {
	if ps == nil { // Start failed; cmd.Run() never got to Wait
		return 1
	}
	if rc := ps.ExitCode(); rc >= 0 {
		return rc
	}
	return 1
}
