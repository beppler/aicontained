//go:build !windows

package launcher

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// The engine child must share the terminal's foreground process group: with
// Setpgid it runs in the background, its stdin reads get SIGTTIN, and
// `podman run -i` hangs with the container stuck in "Created".
func TestRunChildSharesForegroundGroup(t *testing.T) {
	out := filepath.Join(t.TempDir(), "pgid")
	t.Setenv("OUT", out)
	t.Setenv("RUN_CHILD_HELPER", "1")
	// Re-run this test binary as the "engine"; TestChildProcessGroup below
	// writes its process group id. Portable — no /proc, no external tools.
	rc, err := runChild(os.Args[0], []string{"-test.run=TestChildProcessGroup$"})
	if err != nil || rc != 0 {
		t.Fatalf("runChild: rc=%d err=%v", rc, err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var got int
	if _, err := fmt.Sscanf(string(data), "%d", &got); err != nil {
		t.Fatal(err)
	}
	if got != syscall.Getpgrp() {
		t.Errorf("child pgid = %d, launcher pgid = %d — child must share the foreground group", got, syscall.Getpgrp())
	}
}

// TestChildProcessGroup is re-executed as runChild's fake engine.
func TestChildProcessGroup(t *testing.T) {
	if os.Getenv("RUN_CHILD_HELPER") != "1" {
		t.Skip("only runs as the fake engine child")
	}
	if err := os.WriteFile(os.Getenv("OUT"), []byte(fmt.Sprint(syscall.Getpgrp())), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A non-zero exit is a result, not an error: main would print a bogus
// "cli-tools: exit status N" line on every failing engine run otherwise.
func TestRunChildNonZeroExitIsNotAnError(t *testing.T) {
	engine := filepath.Join(t.TempDir(), "fakeengine")
	if err := os.WriteFile(engine, []byte("#!/bin/sh\nexit 3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	rc, err := runChild(engine, nil)
	if err != nil || rc != 3 {
		t.Fatalf("runChild: rc=%d err=%v, want rc=3 err=nil", rc, err)
	}
}
