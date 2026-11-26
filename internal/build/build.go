// Package build implements the `aicontained build` subcommand: build the
// container images from the embedded definitions.
package build

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/beppler/aicontained/internal/engine"
	"github.com/beppler/aicontained/internal/images"
	"github.com/beppler/aicontained/internal/tools"
)

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
			fmt.Println("usage: aicontained build [tool...] [--prune-images] [--no-cache]")
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
		fmt.Fprintln(os.Stderr, "aicontained: extracting image definitions:", err)
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
			fmt.Fprintf(os.Stderr, "aicontained: building %s failed\n", image.tag)
			return exitCode(code, err)
		}
	}

	if pruneImages {
		pruneArgs := []string{"image", "prune"}
		if runtime.GOOS == "windows" { // wslc prompts without it
			pruneArgs = append(pruneArgs, "-f")
		}
		if code, err := eng.Run(pruneArgs...); err != nil || code != 0 {
			return exitCode(code, err)
		}
	}

	fmt.Printf("Built images (%s) for: %s\n", eng.Name, tools.Names(list))
	return 0
}

type imageSpec struct{ tag, dir string }

func buildList(list []tools.Tool) []imageSpec {
	out := []imageSpec{{"aicontained-base", "base"}}
	for _, t := range list {
		out = append(out, imageSpec{t.Image(), t.Name})
	}
	return out
}

func exitCode(code int, err error) int {
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
	root, err := os.MkdirTemp("", "aicontained-containers-")
	if err != nil {
		return "", err
	}
	err = fs.WalkDir(images.Containers, "containers", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(root, p)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := images.Containers.ReadFile(p)
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
