// Package install implements the `aicontained install` subcommand: copy the
// running binary to the user's private bin dir and create one link per tool,
// so argv[0] dispatch picks the right tool.
package install

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/beppler/aicontained/internal/tools"
)

// Install runs `aicontained install [tool...]` and returns the process exit
// code.
func Install(args []string) int {
	var names []string
	for _, a := range args {
		switch a {
		case "-h", "--help":
			fmt.Println("usage: aicontained install [tool...]")
			return 0
		default:
			names = append(names, a)
		}
	}
	list, err := tools.Resolve(names)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	dest, err := install(list)
	if err != nil {
		fmt.Fprintln(os.Stderr, "aicontained: install:", err)
		return 1
	}
	fmt.Printf("Done. Make sure %s is on your PATH, then just run: %s\n", dest, tools.Names(list))
	return 0
}

// install copies the running binary to the user's private bin dir and creates
// one link per tool, so argv[0] dispatch picks the right tool.
func install(list []tools.Tool) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dest, err := binDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return "", err
	}
	sweepStale(dest)
	dst := filepath.Join(dest, binaryName())
	if !sameFile(exe, dst) { // installing the running binary onto itself: skip
		if err := copyExecutable(exe, dst); err != nil {
			return "", err
		}
	}
	for _, t := range list {
		if err := linkTool(dst, filepath.Join(dest, linkName(t.Name))); err != nil {
			return "", err
		}
	}
	return dest, nil
}

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "aicontained.exe"
	}
	return "aicontained"
}

func linkName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func binDir() (string, error) {
	if runtime.GOOS == "windows" {
		// An empty LOCALAPPDATA would make the dest path relative and the
		// install would silently land in the current directory.
		if dir := os.Getenv("LOCALAPPDATA"); dir != "" {
			return filepath.Join(dir, "Programs", "bin"), nil
		}
		return "", errors.New("LOCALAPPDATA is not set: cannot determine the install directory")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		// No home: like the empty LOCALAPPDATA above, a "" would make the
		// path relative and the install would land in the current directory.
		return "", errors.New("cannot determine the user's home directory")
	}
	return filepath.Join(home, ".local", "bin"), nil
}

// linkTool creates a per-tool symlink to the binary. Symlinks are required:
// deleting one never touches the target executable, so a reinstall never hits
// a busy-file error.
func linkTool(target, link string) error {
	if err := removeOldLink(link); err != nil {
		return err
	}
	err := os.Symlink(filepath.Base(target), link)
	if err == nil {
		return nil
	}
	if runtime.GOOS == "windows" {
		// Symlink creation needs Developer Mode (or elevation); without it
		// there is no fallback.
		return fmt.Errorf("%w (Windows symlink creation needs Developer Mode or an elevated terminal)", err)
	}
	return err
}

// removeOldLink deletes the previous per-tool link. On Windows a hardlink to
// a running exe (from an install made before symlinks were required) can't be
// deleted, only renamed: move it aside and let the next install sweep it.
func removeOldLink(link string) error {
	err := os.Remove(link)
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if runtime.GOOS != "windows" {
		return err
	}
	old := fmt.Sprintf("%s.old-%d", link, os.Getpid())
	if rerr := os.Rename(link, old); rerr != nil {
		return rerr
	}
	os.Remove(old) // busy while a session still runs the hardlink; swept next install
	return nil
}

// sweepStale deletes leftovers from earlier installs: the ".new" staging file
// and ".old-<pid>" renames that were busy last time. Only our own file names
// match, so nothing else in the bin dir is touched.
func sweepStale(dest string) {
	prefixes := []string{binaryName() + ".old-", binaryName() + ".new"}
	for _, t := range tools.All() {
		prefixes = append(prefixes, linkName(t.Name)+".old-")
	}
	entries, err := os.ReadDir(dest)
	if err != nil {
		return
	}
	for _, e := range entries {
		for _, p := range prefixes {
			if strings.HasPrefix(e.Name(), p) {
				os.Remove(filepath.Join(dest, e.Name())) // busy or gone: try again next install
				break
			}
		}
	}
}

func sameFile(a, b string) bool {
	ai, erra := os.Stat(a)
	bi, errb := os.Stat(b)
	return erra == nil && errb == nil && os.SameFile(ai, bi)
}

// copyExecutable replaces dst with a copy of src, staged as dst+".new-<pid>"
// and swapped in with a rename so a failed copy can't truncate the installed
// binary. The pid keeps two overlapping installs from writing the same
// staging file. On Unix the rename replaces a running binary in place; on
// Windows a running exe can't be overwritten, only renamed, so the old file
// is moved aside first.
func copyExecutable(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := fmt.Sprintf("%s.new-%d", dst, os.Getpid())
	out, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	var old string // aside name, used only by the Windows rename-aside below
	swapped := false
	if runtime.GOOS == "windows" {
		old = fmt.Sprintf("%s.old-%d", dst, os.Getpid())
		if err := os.Rename(dst, old); err == nil {
			swapped = true
		} else if !errors.Is(err, fs.ErrNotExist) {
			os.Remove(tmp)
			return err
		}
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		if swapped {
			os.Rename(old, dst) // put the old binary back so the install stays usable
		}
		return err
	}
	if swapped {
		os.Remove(old) // busy while a session still runs it; swept next install
	}
	return nil
}
