//go:build windows

package launcher

import (
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
	return exitCode(cmd.ProcessState), err
}

func exitCode(ps *os.ProcessState) int {
	if rc := ps.ExitCode(); rc >= 0 {
		return rc
	}
	return 1
}
