//go:build !windows

package launcher

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// runChild starts the engine as a child process in its own process group,
// inherits the real terminal (no PTY plumbing), forwards signals, and
// returns the child's exit code.
func runChild(name string, args []string) (int, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	// Own process group: terminal-generated SIGINT reaches only this
	// launcher, which forwards it — otherwise the launcher would die and
	// orphan the container.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return 1, err
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(ch)
	go func() {
		for sig := range ch {
			// Negative PID = the child's whole process group.
			_ = syscall.Kill(-cmd.Process.Pid, sig.(syscall.Signal))
		}
	}()

	err := cmd.Wait()
	return exitCode(cmd.ProcessState), err
}

func exitCode(ps *os.ProcessState) int {
	if rc := ps.ExitCode(); rc >= 0 {
		return rc
	}
	if s, ok := ps.Sys().(syscall.WaitStatus); ok && s.Signaled() {
		return 128 + int(s.Signal())
	}
	return 130
}
