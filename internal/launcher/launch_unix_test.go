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
	// Field 5 of /proc/<pid>/stat is the process group id.
	engine := filepath.Join(t.TempDir(), "fakeengine")
	if err := os.WriteFile(engine, []byte("#!/bin/sh\ncut -d' ' -f5 /proc/self/stat > \"$OUT\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OUT", out)

	rc, err := runChild(engine, nil)
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
