// Package launcher assembles the engine command for a tool and runs it as a
// child process that inherits the terminal.
package launcher

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"

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
	cmdArgs := assemble(tool, runtime.GOOS, host, workdir.ContainerPath(host), home, stateDir, name, args)
	return runChild(eng.Name, cmdArgs)
}

// assemble builds the full engine argument list. Pure so the golden test can
// pin the exact flags per platform.
func assemble(tool tools.Tool, goos, hostDir, workDir, homeDir, stateDir, containerName string, args []string) []string {
	out := []string{"run", "--rm", "-it"}

	// Apple `container` runs each container in its own VM; quiet its
	// progress output.
	if goos == "darwin" {
		out = append(out, "--progress", "none")
	}

	out = append(out,
		"-v", hostDir+":"+workDir,
		"-w", workDir,
		"-v", homeDir+":/root",
	)
	if stateDir != "" {
		// opencode's per-run state dir; the launcher saves it back on exit.
		out = append(out, "-v", stateDir+":/root/.local/state/opencode")
	}

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
	return append(out, args...)
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
