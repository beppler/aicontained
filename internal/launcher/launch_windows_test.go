//go:build windows

package launcher

import (
	"testing"
)

// A non-zero exit is a result, not an error: main would print a bogus
// "cli-tools: exit status N" line on every failing engine run otherwise.
// Runs the engine as `cmd /c exit 3` — the Windows twin of the unix test.
func TestRunChildCmdExitIsNotAnError(t *testing.T) {
	rc, err := runChild("cmd", []string{"/c", "exit", "3"})
	if err != nil || rc != 3 {
		t.Fatalf("runChild: rc=%d err=%v, want rc=3 err=nil", rc, err)
	}
}

// Start failures surface as an error, not a nil ProcessState panic.
func TestRunChildStartFailureIsAnError(t *testing.T) {
	rc, err := runChild("cmd-does-not-exist", nil)
	if err == nil || rc != 1 {
		t.Fatalf("runChild: rc=%d err=%v, want rc=1 err non-nil", rc, err)
	}
}
