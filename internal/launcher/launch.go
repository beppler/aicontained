// Package launcher assembles the engine command for a tool and runs it as a
// child process that inherits the terminal.
package launcher

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/beppler/aicontained/internal/engine"
	"github.com/beppler/aicontained/internal/opencode"
	"github.com/beppler/aicontained/internal/tools"
	"github.com/beppler/aicontained/internal/workdir"
)

// Run detects the engine, prepares the tool's host directories, and runs the
// image in a child process, propagating its exit code.
func Run(tool tools.Tool, args []string) (int, error) {
	return run(tool, "", args)
}

// Shell runs the tool's image like Run, but with bash as the entrypoint, so
// args go to bash instead of the tool.
func Shell(tool tools.Tool, args []string) (int, error) {
	return run(tool, "bash", args)
}

// run is Run with an optional entrypoint override ("" keeps the image's).
func run(tool tools.Tool, entrypoint string, args []string) (int, error) {
	eng, err := engine.Detect()
	if err != nil {
		return 1, err
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return 1, err
	}
	host, err := workdir.HostPath()
	if err != nil {
		return 1, fmt.Errorf("cannot read current directory: %w", err)
	}
	mount, cwd := paths(host)
	home, err := homeDir(filepath.Join(userHome, ".aicontained"), tool)
	if err != nil {
		return 1, err
	}
	// 0700: the home holds the tool's login tokens. Chmod also tightens
	// folders created by older versions with 0755.
	if err := os.MkdirAll(home, 0o700); err != nil {
		return 1, err
	}
	if err := os.Chmod(home, 0o700); err != nil {
		return 1, err
	}

	// wslc requires a unique container name; the others just benefit from an
	// identifiable one. The pid is unique among live processes; a recycled pid
	// can hit a stale crashed container — rerun to recover.
	kind := "cli"
	if entrypoint != "" {
		kind = "shell"
	}
	name := fmt.Sprintf("%s_%s_%d", tool.Name, kind, os.Getpid())
	stateDir := ""
	if tool.StateMount {
		run, err := opencode.Seed(home)
		if err != nil {
			return 1, err
		}
		defer func() {
			if err := run.Save(); err != nil {
				fmt.Fprintln(os.Stderr, "aicontained: warning: saving opencode state:", err)
			}
		}()
		stateDir = run.Dir()
	}
	cmdArgs, err := assemble(tool, runtime.GOOS, mount, cwd, home, stateDir, sshAgentSock(), name, entrypoint, args)
	if err != nil {
		return 1, err
	}
	return runChild(eng.Name, cmdArgs)
}

// paths maps the host cwd to its mount point and the in-container work dir.
// Inside a git repo the repo root is mounted (per-project state is shared
// across the repo) and the tool opens in the caller's subdirectory.
func paths(host string) (mount, cwd string) {
	root := workdir.GitRoot(host)
	if root == "" {
		return host, workdir.ContainerPath(host)
	}
	cwd = workdir.ContainerPath(root)
	// Open the tool where the user is; fall back to the root if the resolved
	// cwd escapes the root (host symlink resolved outside it).
	if rel, err := filepath.Rel(root, host); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
		// Join re-introduces OS separators; container paths are always slash-separated.
		cwd = filepath.ToSlash(filepath.Join(cwd, rel))
	}
	return root, cwd
}

var profileRe = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

// homeDir returns the tool's host home under base, suffixed by its profile
// from base/aicontained.toml ([tool] profile = "name"), if any.
func homeDir(base string, tool tools.Tool) (string, error) {
	cfg := filepath.Join(base, "aicontained.toml")
	var conf map[string]struct{ Profile string }
	md, err := toml.DecodeFile(cfg, &conf)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("reading %s: %w", cfg, err)
	}
	profile := ""
	seen := map[string]bool{}
	for name, t := range conf {
		lower := strings.ToLower(name)
		// A typo like "profil" would silently fall back to the default folder.
		if !md.IsDefined(name, "profile") {
			return "", fmt.Errorf("%s: table [%s] has no profile", cfg, name)
		}
		// [Claude] and [claude] are distinct TOML keys; map order would pick one at random.
		if seen[lower] {
			return "", fmt.Errorf("%s: duplicate table for %s", cfg, lower)
		}
		seen[lower] = true
		if lower == tool.Name {
			profile = t.Profile
		}
	}
	p := strings.ToLower(profile) // folder names must not differ by case alone (Windows is case-insensitive)
	if p == "" {
		return filepath.Join(base, tool.HomeDir), nil
	}
	// The profile becomes a path component: reject anything that could escape
	// base, or that Windows would rename away (a trailing dot is stripped).
	if !profileRe.MatchString(p) || p == "." || p == ".." || strings.HasSuffix(p, ".") {
		return "", fmt.Errorf("%s: invalid profile %q for %s", cfg, profile, tool.Name)
	}
	return filepath.Join(base, tool.HomeDir+"-"+p), nil
}

// sshAgentSock returns the host SSH agent socket to mount, or "" when the
// passthrough is off (Windows, env var unset, or file missing).
func sshAgentSock() string {
	if runtime.GOOS == "windows" {
		return ""
	}
	v := os.Getenv("SSH_AUTH_SOCK")
	if v == "" {
		return ""
	}
	if _, err := os.Stat(v); err != nil {
		return ""
	}
	return v
}

// assemble builds the full engine argument list. Pure so the golden test can
// pin the exact flags per platform. sshAgent is the host socket to mount at
// /var/run/ssh-agent, "" for none.
func assemble(tool tools.Tool, goos, hostDir, workDir, homeDir, stateDir, sshAgent, containerName, entrypoint string, args []string) ([]string, error) {
	if goos == "windows" { // no unix sockets; sshAgent stays empty in practice too
		sshAgent = ""
	}
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
	if sshAgent != "" {
		mounts = append(mounts, [2]string{sshAgent, "/var/run/ssh-agent"})
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
	if sshAgent != "" {
		// Explicit value: a bare -e would forward the host socket path,
		// which does not exist in the container.
		out = append(out, "-e", "SSH_AUTH_SOCK=/var/run/ssh-agent")
	}

	// Linux hardening (Podman): keep just the capabilities needed for file
	// handling and apt-get.
	if goos == "linux" {
		out = append(out,
			"--security-opt", "no-new-privileges",
			"--cap-drop=ALL",
			"--cap-add=CHOWN,DAC_OVERRIDE,FOWNER,FSETID,SETUID,SETGID",
		)
	}

	if entrypoint != "" {
		out = append(out, "--entrypoint", entrypoint)
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
