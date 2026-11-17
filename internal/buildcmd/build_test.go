package buildcmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/beppler/cli-tools/internal/images"
	"github.com/beppler/cli-tools/internal/tools"
)

// The binary builds images from the embedded definitions, not a repo
// checkout — they must actually be in the binary.
func TestEmbeddedContainers(t *testing.T) {
	for _, f := range []string{"containers/base/Dockerfile", "containers/claude/Dockerfile", "containers/opencode/Dockerfile"} {
		if _, err := images.Containers.ReadFile(f); err != nil {
			t.Errorf("embedded %s: %v", f, err)
		}
	}
}

// install must copy the binary and create per-tool links that argv[0]
// dispatch relies on. Runs the platform binDir path (Linux: $HOME/.local/bin,
// Windows: %LOCALAPPDATA%\Programs\bin).
func TestInstall(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))

	want := filepath.Join(home, ".local", "bin")
	if runtime.GOOS == "windows" {
		want = filepath.Join(home, "AppData", "Local", "Programs", "bin")
	}

	dest, err := install(tools.All())
	if err != nil {
		t.Fatal(err)
	}
	if dest != want {
		t.Fatalf("dest = %q, want %q", dest, want)
	}

	bin := filepath.Join(dest, binaryName())
	info, err := os.Stat(bin)
	if err != nil {
		t.Fatal(err)
	}
	// Windows has no exec bits; os.Stat always reports non-executable.
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o100 == 0 {
		t.Error("installed binary is not executable")
	}
	for _, tool := range tools.All() {
		link := filepath.Join(dest, linkName(tool.Name))
		got, err := os.Readlink(link)
		if err != nil && runtime.GOOS == "windows" {
			if sameFile(link, bin) { // hardlink fallback
				continue
			}
		}
		if err != nil || got != filepath.Base(bin) {
			t.Errorf("%s: link to %q, %v (want relative %s)", tool.Name, got, err, filepath.Base(bin))
		}
	}
}

// Install rejects unknown tool names with exit code 1, like Build does.
func TestInstallUnknownTool(t *testing.T) {
	if code := Install([]string{"nope"}); code != 1 {
		t.Errorf("Install(nope) = %d, want 1", code)
	}
	if code := Install([]string{"--help"}); code != 0 {
		t.Errorf("Install(--help) = %d, want 0", code)
	}
}
