package buildcmd

import (
	"os"
	"path/filepath"
	"testing"

	clitools "github.com/beppler/cli-tools"
	"github.com/beppler/cli-tools/internal/tools"
)

// The binary builds images from the embedded definitions, not a repo
// checkout — they must actually be in the binary.
func TestEmbeddedContainers(t *testing.T) {
	for _, f := range []string{"containers/base/Dockerfile", "containers/claude/Dockerfile", "containers/opencode/Dockerfile"} {
		if _, err := clitools.Containers.ReadFile(f); err != nil {
			t.Errorf("embedded %s: %v", f, err)
		}
	}
}

// install must copy the binary and create per-tool links that argv[0]
// dispatch relies on. Runs the linux binDir path ($HOME/.local/bin).
func TestInstall(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	exe := filepath.Join(t.TempDir(), "cli-tools")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	dest, err := install(tools.All())
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".local", "bin"); dest != want {
		t.Fatalf("dest = %q, want %q", dest, want)
	}

	bin := filepath.Join(dest, "cli-tools")
	info, err := os.Stat(bin)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o100 == 0 {
		t.Error("installed binary is not executable")
	}
	for _, tool := range tools.All() {
		link := filepath.Join(dest, tool.Name)
		got, err := os.Readlink(link)
		if err != nil || got != "cli-tools" {
			t.Errorf("%s: link to %q, %v (want relative cli-tools)", tool.Name, got, err)
		}
	}
}
