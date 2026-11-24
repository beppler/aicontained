// Package engine autodetects the container engine for the current OS.
package engine

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// Engine is a container engine CLI (Apple `container`, Podman, `wslc`).
type Engine struct {
	Name string // command name
}

// Detect finds the engine for the current OS: Apple `container` on macOS,
// Podman on Linux, `wslc` on Windows.
func Detect() (*Engine, error) {
	return detect(runtime.GOOS, exec.LookPath)
}

func detect(goos string, lookPath func(string) (string, error)) (*Engine, error) {
	switch goos {
	case "darwin":
		if _, err := lookPath("container"); err == nil {
			return &Engine{Name: "container"}, nil
		}
	case "windows":
		if _, err := lookPath("wslc"); err == nil { // LookPath resolves .exe on Windows
			return &Engine{Name: "wslc"}, nil
		}
	case "linux":
		if _, err := lookPath("podman"); err == nil {
			return &Engine{Name: "podman"}, nil
		}
	default:
		return nil, fmt.Errorf("Unsupported OS: %s", goos)
	}
	return nil, fmt.Errorf("No supported container engine found. Supported: Apple container (macOS), Podman (Linux), WSL Containers (Windows). Docker is not supported.")
}

// Run executes the engine with args, inheriting stdio, and returns the exit
// code (0 when it succeeded).
func (e *Engine) Run(args ...string) (int, error) {
	cmd := exec.Command(e.Name, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		code := exitErr.ProcessState.ExitCode()
		if code < 0 { // killed by a signal
			code = 1
		}
		return code, nil
	}
	return 1, err
}
