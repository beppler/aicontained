//go:build !windows

package launcher

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// runChild starts the engine as a child process that shares this process
// group and inherits the real terminal (no PTY plumbing), exactly like the
// legacy bash scripts did.
//
// No Setpgid: a child in a background process group that reads the TTY —
// which podman -i does — is suspended by SIGTTIN and hangs with the
// container stuck in "Created". Terminal Ctrl+C is delivered by the kernel
// to the whole foreground group, launcher included, so SIGINT is swallowed
// here while the engine tears the container down. A SIGTERM aimed only at
// this process is forwarded so the container is still cleaned up. SIGHUP
// (closed terminal tab, dropped SSH) is caught too: the default action
// would kill the launcher before the deferred state save in Run runs.
func runChild(name string, args []string) (int, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		return 1, err
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(ch)
	go func() {
		for sig := range ch {
			if sig == syscall.SIGINT {
				continue // kernel already delivered it to the whole group
			}
			if cmd.Process != nil {
				_ = cmd.Process.Signal(syscall.SIGTERM)
			}
		}
	}()

	err := cmd.Wait()
	if _, ok := errors.AsType[*exec.ExitError](err); !ok {
		return exitCode(cmd.ProcessState), err
	}
	// The child started and ran to completion: its exit code is the
	// result, not an error (Go reports every non-zero exit as ExitError).
	return exitCode(cmd.ProcessState), nil
}

func exitCode(ps *os.ProcessState) int {
	if ps == nil { // Wait failed before reaping; matches the Windows guard
		return 1
	}
	if rc := ps.ExitCode(); rc >= 0 {
		return rc
	}
	if s, ok := ps.Sys().(syscall.WaitStatus); ok && s.Signaled() {
		return 128 + int(s.Signal())
	}
	return 130
}
