package buildcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/beppler/aicontained/internal/images"
	"github.com/beppler/aicontained/internal/tools"
)

// The binary builds images from the embedded definitions, not a repo
// checkout — they must actually be in the binary.
func TestEmbeddedContainers(t *testing.T) {
	for _, f := range []string{"containers/base/Dockerfile", "containers/claude/Dockerfile", "containers/opencode/Dockerfile", "containers/pi/Dockerfile"} {
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
		if err != nil || got != filepath.Base(bin) {
			t.Errorf("%s: link to %q, %v (want relative %s)", tool.Name, got, err, filepath.Base(bin))
		}
	}
}

// On Windows an unset LOCALAPPDATA must fail the install instead of
// silently writing into the current directory.
func TestBinDirMissingLocalAppData(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only")
	}
	t.Setenv("LOCALAPPDATA", "")
	if _, err := binDir(); err == nil {
		t.Error("binDir with empty LOCALAPPDATA: want an error")
	}
}

// Unix mirrors Windows: no home directory must fail the install instead of
// silently writing into the current directory.
func TestBinDirMissingHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only")
	}
	t.Setenv("HOME", "")
	if _, err := binDir(); err == nil {
		t.Error("binDir with empty HOME: want an error")
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

// A failed copy must leave the installed binary intact and no staging file,
// since a truncated binary breaks every tool until the next install.
func TestCopyExecutableFailureLeavesTargetIntact(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, binaryName())
	if err := os.WriteFile(dst, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyExecutable(dir, dst); err == nil { // src is a directory: reading it fails
		t.Fatal("copying a directory over the target succeeded")
	}
	got, err := os.ReadFile(dst)
	if err != nil || string(got) != "old" {
		t.Errorf("dst = %q, %v; want \"old\" untouched", got, err)
	}
	if _, err := os.Stat(fmt.Sprintf("%s.new-%d", dst, os.Getpid())); !os.IsNotExist(err) {
		t.Errorf("staging file left behind: %v", err)
	}
}

// A successful copy replaces the target and leaves no staging or .old files.
func TestCopyExecutableReplacesTarget(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.WriteFile(src, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, binaryName())
	if err := os.WriteFile(dst, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyExecutable(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil || string(got) != "new" {
		t.Errorf("dst = %q, %v; want \"new\"", got, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if name := e.Name(); name != "src" && name != binaryName() {
			t.Errorf("leftover file %q after copy", name)
		}
	}
}

// The next install sweeps stale staging and .old-<pid> files left by earlier
// installs, but touches nothing else in the bin dir.
func TestSweepStale(t *testing.T) {
	dest := t.TempDir()
	junk := []string{binaryName() + ".new", binaryName() + ".new-42", binaryName() + ".old-99", linkName("claude") + ".old-7"}
	keep := []string{linkName("claude"), "unrelated.txt"}
	for _, name := range append(junk, keep...) {
		if err := os.WriteFile(filepath.Join(dest, name), []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	sweepStale(dest)
	for _, name := range junk {
		if _, err := os.Stat(filepath.Join(dest, name)); !os.IsNotExist(err) {
			t.Errorf("%s not swept: %v", name, err)
		}
	}
	for _, name := range keep {
		if _, err := os.Stat(filepath.Join(dest, name)); err != nil {
			t.Errorf("%s wrongly swept: %v", name, err)
		}
	}
}

// Pre-symlink installs made hardlinks; install must migrate them to symlinks
// so a reinstall never hits a busy file via the link.
func TestInstallMigratesHardlinks(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))

	dest, err := install(tools.All())
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dest, binaryName())
	link := filepath.Join(dest, linkName("claude"))
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(bin, link); err != nil { // recreate as a hardlink, like an old install
		t.Skipf("hardlinks unavailable: %v", err)
	}
	if _, err := install(tools.All()); err != nil {
		t.Fatal(err)
	}
	got, err := os.Readlink(link)
	if err != nil || got != filepath.Base(bin) {
		t.Errorf("claude: link to %q, %v (want relative %s)", got, err, filepath.Base(bin))
	}
}
