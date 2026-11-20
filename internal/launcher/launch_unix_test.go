//go:build !windows

package launcher

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"
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
	if err := os.WriteFile(os.Getenv("OUT"), fmt.Appendf(nil, "%d", syscall.Getpgrp()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A non-zero exit is a result, not an error: main would print a bogus
// "aicontained: exit status N" line on every failing engine run otherwise.
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

// SIGHUP (closed terminal tab, dropped SSH) must not kill the launcher
// before the deferred opencode state save in Run runs: it is caught and
// forwarded to the child as SIGTERM.
func TestRunChildForwardsSIGHUP(t *testing.T) {
	out := filepath.Join(t.TempDir(), "hup")
	t.Setenv("OUT", out)
	t.Setenv("RUN_CHILD_HUP_HELPER", "1")
	type exit struct {
		rc  int
		err error
	}
	done := make(chan exit, 1)
	go func() {
		// Re-run this test binary as the "engine"; TestChildTrapsSIGTERM
		// acknowledges the forwarded SIGTERM by writing the marker file.
		rc, err := runChild(os.Args[0], []string{"-test.run=TestChildTrapsSIGTERM$"})
		done <- exit{rc, err}
	}()

	// Wait for the child to install its SIGTERM handler, then HUP only the
	// launcher (not the group, so the child does not receive it directly).
	ready := out + ".ready"
	for i := 0; ; i++ {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		if i == 100 {
			t.Fatal("fake engine never became ready")
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGHUP); err != nil {
		t.Fatal(err)
	}
	res := <-done
	if res.err != nil || res.rc != 0 {
		t.Fatalf("runChild: rc=%d err=%v, want rc=0 err=nil", res.rc, res.err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("child did not receive the forwarded signal: %v", err)
	}
}

// TestChildTrapsSIGTERM is re-executed as runChild's fake engine for
// TestRunChildForwardsSIGHUP: it traps SIGTERM, signals readiness, writes
// the marker on termination, and gives up non-zero so the parent's
// assertions fail instead of hanging.
func TestChildTrapsSIGTERM(t *testing.T) {
	if os.Getenv("RUN_CHILD_HUP_HELPER") != "1" {
		t.Skip("only runs as the fake engine child")
	}
	out := os.Getenv("OUT")
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM)
	if err := os.WriteFile(out+".ready", nil, 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ch:
		if err := os.WriteFile(out, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		os.Exit(1)
	}
}
