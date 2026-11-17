// Package launcher assembles the engine command for a tool and runs it as a
// child process that inherits the terminal.
package launcher

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/beppler/cli-tools/internal/engine"
	"github.com/beppler/cli-tools/internal/opencode"
	"github.com/beppler/cli-tools/internal/tools"
	"github.com/beppler/cli-tools/internal/workdir"
)

// Run detects the engine, prepares the tool's host directories, and runs the
// image in a child process, propagating its exit code.
func Run(tool tools.Tool, args []string) (int, error) {
	eng, err := engine.Detect()
	if err != nil {
		return 1, err
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return 1, err
	}
	host := workdir.HostPath()
	home := filepath.Join(userHome, ".cli-tools", tool.HomeDir)
	if err := os.MkdirAll(home, 0o755); err != nil {
		return 1, err
	}

	// wslc requires a unique container name; the other engines don't care.
	name := ""
	if runtime.GOOS == "windows" {
		name = fmt.Sprintf("%s_cli_%d", tool.Name, rand.IntN(1<<31))
	}
	stateDir := ""
	if tool.StateMount {
		run, err := opencode.Seed(home)
		if err != nil {
			return 1, err
		}
		defer func() {
			if err := run.Save(); err != nil {
				fmt.Fprintln(os.Stderr, "cli-tools: warning: saving opencode state:", err)
			}
		}()
		stateDir = run.Dir()
	}
	cmdArgs, err := assemble(tool, runtime.GOOS, host, workdir.ContainerPath(host), home, stateDir, name, args)
	if err != nil {
		return 1, err
	}
	return runChild(eng.Name, cmdArgs)
}

// assemble builds the full engine argument list. Pure so the golden test can
// pin the exact flags per platform.
func assemble(tool tools.Tool, goos, hostDir, workDir, homeDir, stateDir, containerName string, args []string) ([]string, error) {
	out := []string{"run", "--rm", "-it"}

	// Apple `container` runs each container in its own VM; quiet its
	// progress output.
	if goos == "darwin" {
		out = append(out, "--progress", "none")
	}

	mounts := [][2]string{{hostDir, workDir}, {homeDir, "/root"}}
	if stateDir != "" {
		// opencode's per-run state dir; the launcher saves it back on exit.
		mounts = append(mounts, [2]string{stateDir, "/root/.local/state/opencode"})
	}
	for _, m := range mounts {
		mount, err := mountFlag(goos, m[0], m[1])
		if err != nil {
			return nil, err
		}
		out = append(out, mount...)
	}
	out = append(out, "-w", workDir)

	// The engine forwards the host's own value for a bare -e KEY and skips
	// unset ones, so secret values never appear in the command line.
	out = append(out, envArgs(goos, tool)...)

	// Linux hardening (Podman): keep just the capabilities needed for file
	// handling and apt-get.
	if goos == "linux" {
		out = append(out,
			"--security-opt", "no-new-privileges",
			"--cap-drop=ALL",
			"--cap-add=CHOWN,DAC_OVERRIDE,FOWNER,FSETID,SETUID,SETGID",
		)
	}

	if containerName != "" {
		out = append(out, "--name", containerName)
	}

	out = append(out, tool.Image())
	return append(out, args...), nil
}

// mountFlag builds the argument pair mounting src into dst. `-v src:dst` is
// the default everywhere, but a colon in the host path is parsed as the
// mount-options separator, so on non-Windows hosts fall back to --mount
// (where the colon is not a delimiter). Podman has no comma escaping in
// --mount either (containers/podman#11688), so both chars together is
// unsolvable — fail with a clear message instead. Windows keeps `-v` even
// for `C:\...` paths; wslc handles the drive-letter colon.
func mountFlag(goos, src, dst string) ([]string, error) {
	if goos == "windows" || !strings.ContainsRune(src, ':') {
		return []string{"-v", src + ":" + dst}, nil
	}
	if strings.ContainsRune(src, ',') {
		return nil, fmt.Errorf("cannot mount %q: contains both ':' and ',' — rename the directory", src)
	}
	return []string{"--mount", "type=bind,src=" + src + ",dst=" + dst}, nil
}

func envArgs(goos string, tool tools.Tool) []string {
	var keys []string
	switch goos {
	case "windows":
		// wslc is a native Windows process; TERM/COLORTERM are usually unset
		// in the shell, so pin the values the TUIs need. The GIT_CONFIG_*
		// settings paper over CRLF-checked-out repos (see README).
		keys = []string{
			"TERM=xterm-256color", "COLORTERM=truecolor",
			"GIT_CONFIG_COUNT=2",
			"GIT_CONFIG_KEY_0=core.autocrlf", "GIT_CONFIG_VALUE_0=true",
			"GIT_CONFIG_KEY_1=core.filemode", "GIT_CONFIG_VALUE_1=false",
		}
	default:
		keys = []string{"TERM", "COLORTERM"}
	}
	out := make([]string, 0, 2*len(keys)+2*len(tool.EnvKeys))
	for _, k := range keys {
		out = append(out, "-e", k)
	}
	for _, k := range tool.EnvKeys {
		out = append(out, "-e", k)
	}
	return out
}
