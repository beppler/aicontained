// Package buildcmd implements the `cli-tools build` subcommand: build the
// container images, then install the binary with per-tool links.
package buildcmd

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	clitools "github.com/beppler/cli-tools"
	"github.com/beppler/cli-tools/internal/engine"
	"github.com/beppler/cli-tools/internal/tools"
)

// Build runs `cli-tools build [tool...] [--prune-images] [--no-cache]` and
// returns the process exit code. It only builds images; use Install to put
// the binary and per-tool links on disk.
func Build(args []string) int {
	pruneImages, noCache := false, false
	var names []string
	for _, a := range args {
		switch a {
		case "--prune-images":
			pruneImages = true
		case "--no-cache":
			noCache = true
		case "-h", "--help":
			fmt.Println("usage: cli-tools build [tool...] [--prune-images] [--no-cache]")
			return 0
		default:
			names = append(names, a)
		}
	}
	list, code := resolveTools(names)
	if code != 0 {
		return code
	}

	eng, err := engine.Detect()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if runtime.GOOS == "darwin" {
		eng.Run("system", "start") // nudge a stopped daemon, like the legacy script
		if err := ensureRosetta(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	root, err := writeContainers()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cli-tools: extracting image definitions:", err)
		return 1
	}
	defer os.RemoveAll(root)

	for _, image := range buildList(list) {
		dir := filepath.Join(root, "containers", image.dir)
		buildArgs := []string{"build"}
		if noCache {
			buildArgs = append(buildArgs, "--no-cache")
		}
		buildArgs = append(buildArgs, "-t", image.tag, dir)
		if code, err := eng.Run(buildArgs...); err != nil || code != 0 {
			fmt.Fprintf(os.Stderr, "cli-tools: building %s failed\n", image.tag)
			return exitish(code, err)
		}
	}

	if pruneImages {
		pruneArgs := []string{"image", "prune"}
		if runtime.GOOS == "windows" { // wslc prompts without it
			pruneArgs = append(pruneArgs, "-f")
		}
		if code, err := eng.Run(pruneArgs...); err != nil || code != 0 {
			return exitish(code, err)
		}
	}

	fmt.Printf("Built images (%s) for: %s\n", eng.Name, toolNames(list))
	return 0
}

// Install runs `cli-tools install [tool...]` and returns the process exit
// code. It copies the running binary to the user's private bin dir and
// creates one link per tool, so argv[0] dispatch picks the right tool.
func Install(args []string) int {
	var names []string
	for _, a := range args {
		switch a {
		case "-h", "--help":
			fmt.Println("usage: cli-tools install [tool...]")
			return 0
		default:
			names = append(names, a)
		}
	}
	list, code := resolveTools(names)
	if code != 0 {
		return code
	}
	dest, err := install(list)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cli-tools: install:", err)
		return 1
	}
	fmt.Printf("Done. Make sure %s is on your PATH, then just run: %s\n", dest, toolNames(list))
	return 0
}

// resolveTools maps user-supplied names to tools; no names means all tools.
// Returns a nonzero exit code on the first unknown name.
func resolveTools(names []string) ([]tools.Tool, int) {
	if len(names) == 0 {
		return tools.All(), 0
	}
	var list []tools.Tool
	for _, n := range names {
		t, ok := tools.Lookup(n)
		if !ok {
			fmt.Fprintf(os.Stderr, "Unknown tool: %s (available: %s)\n", n, toolNames(tools.All()))
			return nil, 1
		}
		list = append(list, t)
	}
	return list, 0
}

type imageSpec struct{ tag, dir string }

func buildList(list []tools.Tool) []imageSpec {
	out := []imageSpec{{"cli-tools-base", "base"}}
	for _, t := range list {
		out = append(out, imageSpec{t.Image(), t.Name})
	}
	return out
}

func exitish(code int, err error) int {
	if err != nil {
		return 1
	}
	return code
}

// ensureRosetta ports the legacy check: `container build` runs its BuildKit
// builder under Rosetta regardless of target architecture, and fails with
// "Rosetta is not installed" if it's missing.
func ensureRosetta() error {
	if err := exec.Command("pkgutil", "--pkg-info", "com.apple.pkg.RosettaUpdateAuto").Run(); err == nil {
		return nil
	}
	fmt.Fprintln(os.Stderr, "Installing Rosetta 2 (required by container's build tool)...")
	cmd := exec.Command("softwareupdate", "--install-rosetta", "--agree-to-license")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

// writeContainers extracts the embedded image definitions to a temp dir.
func writeContainers() (string, error) {
	root, err := os.MkdirTemp("", "cli-tools-containers-")
	if err != nil {
		return "", err
	}
	err = fs.WalkDir(clitools.Containers, "containers", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(root, p)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := clitools.Containers.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		os.RemoveAll(root)
		return "", err
	}
	return root, nil
}

// install copies the running binary to the user's private bin dir and creates
// one link per tool, so argv[0] dispatch picks the right tool.
func install(list []tools.Tool) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dest := binDir()
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return "", err
	}
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
		return "cli-tools.exe"
	}
	return "cli-tools"
}

func linkName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func binDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "bin")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".local", "bin")
}

func linkTool(target, link string) error {
	_ = os.Remove(link)
	err := os.Symlink(filepath.Base(target), link)
	if err == nil {
		return nil
	}
	if runtime.GOOS == "windows" {
		// Symlink creation needs Developer Mode (or elevation); a hardlink
		// copy works without either.
		if lerr := os.Link(target, link); lerr == nil {
			fmt.Fprintln(os.Stderr, "warning: symlink creation failed, made a hardlink copy instead (enable Windows Developer Mode for real symlinks):", err)
			return nil
		}
	}
	return err
}

func sameFile(a, b string) bool {
	ai, erra := os.Stat(a)
	bi, errb := os.Stat(b)
	return erra == nil && errb == nil && os.SameFile(ai, bi)
}

func copyExecutable(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func toolNames(list []tools.Tool) string {
	names := make([]string, len(list))
	for i, t := range list {
		names[i] = t.Name
	}
	return strings.Join(names, " ")
}
