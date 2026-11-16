package opencode

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestSeedAndSave(t *testing.T) {
	home := t.TempDir()
	shared := filepath.Join(home, ".local", "state", "opencode")
	mustWrite(t, filepath.Join(shared, "model.json"), `{"model":"a"}`)
	mustWrite(t, filepath.Join(shared, "prompt-history.jsonl"), "hello\n")
	mustWrite(t, filepath.Join(shared, filepath.FromSlash("latest/tui/highlights.json")), "seen")

	run, err := Seed(home)
	if err != nil {
		t.Fatal(err)
	}
	// Seeded into the run dir.
	got, err := os.ReadFile(filepath.Join(run.dir, "model.json"))
	if err != nil || string(got) != `{"model":"a"}` {
		t.Fatalf("seed model.json: %q, %v", got, err)
	}

	// Change model + add history entries in the run dir, then save back.
	mustWrite(t, filepath.Join(run.dir, "model.json"), `{"model":"b"}`)
	mustWrite(t, filepath.Join(run.dir, historyFile), "hello\nworld\nhéllo ☃\n")
	highlights := filepath.Join(run.dir, filepath.FromSlash("latest/tui/highlights.json"))
	if err := os.Remove(highlights); err != nil { // untouched file: shared copy stays
		t.Fatal(err)
	}
	if err := run.Save(); err != nil {
		t.Fatal(err)
	}

	assertFile(t, filepath.Join(shared, "model.json"), `{"model":"b"}`)
	assertFile(t, filepath.Join(shared, historyFile), "hello\nworld\nhéllo ☃\n")
	assertFile(t, filepath.Join(shared, filepath.FromSlash("latest/tui/highlights.json")), "seen")
	if _, err := os.Stat(run.dir); !os.IsNotExist(err) {
		t.Errorf("run dir not removed: %v", err)
	}
}

func TestSaveUnchangedIsNoop(t *testing.T) {
	home := t.TempDir()
	run, err := Seed(home)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(run.dir, "model.json"), `{"model":"a"}`)
	mustWrite(t, filepath.Join(run.dir, historyFile), "same\n")
	if err := run.Save(); err != nil {
		t.Fatal(err)
	}
	// Shared had no files: the run's model.json counts as "differing" so it
	// is saved back; the history line "same" is new so it is appended.
	// Re-seeding must round-trip that state.
	run2, err := Seed(home)
	if err != nil {
		t.Fatal(err)
	}
	assertFile(t, filepath.Join(run2.dir, "model.json"), `{"model":"a"}`)
	assertFile(t, filepath.Join(run2.dir, historyFile), "same\n")
}

func TestHistoryMerge(t *testing.T) {
	home := t.TempDir()
	shared := filepath.Join(home, ".local", "state", "opencode")
	// Shared history without trailing newline (a crash mid-write) must be
	// repaired on disk, not merged into one line.
	mustWrite(t, filepath.Join(shared, historyFile), "a")

	run, _ := Seed(home)
	mustWrite(t, filepath.Join(run.dir, historyFile), "a\na\nb\n\nb\n")
	if err := run.Save(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(shared, historyFile))
	if err != nil {
		t.Fatal(err)
	}
	if want := "a\nb\n"; string(data) != want {
		t.Errorf("merged history = %q, want %q", data, want)
	}
	if data[0] == 0xEF { // UTF-8 BOM would break JSONL consumers
		t.Error("history has a BOM")
	}
}

func TestSeedRemovesStaleRunDir(t *testing.T) {
	home := t.TempDir()
	stale := filepath.Join(home, ".local", "state", fmt.Sprintf("opencode-%d", os.Getpid()))
	mustWrite(t, filepath.Join(stale, "junk"), "leftover")
	if _, err := Seed(home); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(stale, "junk")); !os.IsNotExist(err) {
		t.Errorf("stale leftover survived: %v", err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertFile(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("%s = %q, want %q", path, got, want)
	}
}
