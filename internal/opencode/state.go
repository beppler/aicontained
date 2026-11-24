// Package opencode manages opencode's per-run container state: opencode
// keeps a single service registration (service.json) in its state dir, so
// sharing one dir between containers makes each instance kill the other's
// server. Every run therefore gets a private state dir, seeded from — and
// saved back to — the shared one.
package opencode

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// Files seeded into the run dir and saved back on exit. service.json and the
// rest of latest/ and locks/ stay per-run: they can't be shared between
// running instances.
var keep = []string{"model.json", "prompt-history.jsonl", "latest/tui/highlights.json"}

const historyFile = "prompt-history.jsonl"

// Run is one launcher process's private opencode state dir.
type Run struct {
	dir    string // per-run, mounted into the container
	shared string // persistent, under opencode-home/.local/state/opencode
}

// Seed creates the per-run state dir (named after this process's PID so
// orphaned dirs are easy to spot) and copies the persistent files into it.
func Seed(homeDir string) (*Run, error) {
	stateParent := filepath.Join(homeDir, ".local", "state")
	shared := filepath.Join(stateParent, "opencode")
	dir := filepath.Join(stateParent, fmt.Sprintf("opencode-%d", os.Getpid()))
	if err := os.RemoveAll(dir); err != nil { // stale leftover from a reused PID
		return nil, err
	}
	// 0700: the state sits under the tool's credential home.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(shared, 0o700); err != nil {
		return nil, err
	}
	for _, f := range keep {
		src := filepath.Join(shared, filepath.FromSlash(f))
		if _, err := os.Stat(src); err != nil {
			continue // nothing saved yet
		}
		dst := filepath.Join(dir, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return nil, err
		}
		if err := copyFile(src, dst); err != nil {
			return nil, err
		}
	}
	return &Run{dir: dir, shared: shared}, nil
}

// Dir is the per-run state directory to mount over /root/.local/state/opencode.
func (r *Run) Dir() string { return r.dir }

// Save copies changed files back to the shared dir and removes the run dir.
// The prompt history is appended, not overwritten, so instances that exited
// in the meantime don't lose their entries (last-exit-wins would wipe them).
// Safe to call after the container exits for any reason.
func (r *Run) Save() error {
	var firstErr error
	check := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	for _, f := range keep {
		if f == historyFile {
			check(r.mergeHistory())
			continue
		}
		check(saveChanged(r.dir, r.shared, f))
	}
	check(os.RemoveAll(r.dir))
	return firstErr
}

// saveChanged copies dir/file over shared/file when the run copy exists and
// differs (the shared copy may not exist yet, which counts as differing).
func saveChanged(dir, shared, file string) error {
	src := filepath.Join(dir, filepath.FromSlash(file))
	data, err := os.ReadFile(src)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	dst := filepath.Join(shared, filepath.FromSlash(file))
	if cur, err := os.ReadFile(dst); err == nil && bytes.Equal(cur, data) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// mergeHistory appends the run's prompt-history lines that the shared file
// doesn't have yet. The run file starts as a copy of the shared one and
// opencode later rewrites it whole when trimming (it keeps at most 50
// entries), so "new" is judged per line.
func (r *Run) mergeHistory() error {
	data, err := os.ReadFile(filepath.Join(r.dir, historyFile))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	sharedPath := filepath.Join(r.shared, historyFile)
	existing, err := os.ReadFile(sharedPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	known := map[string]bool{}
	for _, line := range splitLines(existing) {
		known[line] = true
	}
	var add bytes.Buffer
	for _, line := range splitLines(data) {
		if line == "" || known[line] {
			continue
		}
		known[line] = true
		add.WriteString(line)
		add.WriteByte('\n')
	}
	if add.Len() == 0 {
		return nil
	}
	if err := os.MkdirAll(r.shared, 0o755); err != nil {
		return err
	}
	// Bytes are already UTF-8 without BOM; append so concurrent instances merge.
	f, err := os.OpenFile(sharedPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	// Two JSON objects must never end up on one line: repair a missing
	// trailing newline on disk before appending.
	if len(existing) > 0 && !bytes.HasSuffix(existing, []byte("\n")) {
		if _, err := f.Write([]byte("\n")); err != nil {
			return err
		}
	}
	_, err = f.Write(add.Bytes())
	return err
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

func splitLines(data []byte) []string {
	lines := bytes.Split(data, []byte("\n"))
	if n := len(lines); n > 0 && len(lines[n-1]) == 0 {
		lines = lines[:n-1] // trailing newline, not an empty line
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = string(l)
	}
	return out
}
