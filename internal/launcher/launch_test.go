package launcher

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/beppler/aicontained/internal/tools"
	"github.com/beppler/aicontained/internal/workdir"
)

// Golden test pinning the exact engine flags per platform.
func TestAssemble(t *testing.T) {
	host, work, home := "/home/u/proj", "/workspace/HASH/proj", "/home/u/.aicontained"
	cases := []struct {
		name     string
		tool     string
		goos     string
		ctrName  string
		stateDir string
		sshAgent string
		homeSub  string
		entry    string
		want     []string
	}{
		{
			name: "claude/darwin", tool: "claude", goos: "darwin", ctrName: "claude_cli_42", homeSub: "claude-home",
			want: []string{
				"run", "--rm", "-it", "--progress", "none",
				"-v", host + ":" + work,
				"-v", home + "/claude-home:/root",
				"-w", work,
				"-e", "TERM", "-e", "COLORTERM",
				"-e", "ANTHROPIC_API_KEY",
				"--name", "claude_cli_42",
				"aicontained-claude", "--version",
			},
		},
		{
			name: "claude/linux", tool: "claude", goos: "linux", ctrName: "claude_cli_42", homeSub: "claude-home",
			want: []string{
				"run", "--rm", "-it",
				"-v", host + ":" + work,
				"-v", home + "/claude-home:/root",
				"-w", work,
				"-e", "TERM", "-e", "COLORTERM",
				"-e", "ANTHROPIC_API_KEY",
				"--security-opt", "no-new-privileges",
				"--cap-drop=ALL",
				"--cap-add=CHOWN,DAC_OVERRIDE,FOWNER,FSETID,SETUID,SETGID",
				"--name", "claude_cli_42",
				"aicontained-claude", "--version",
			},
		},
		{
			name: "claude/windows", tool: "claude", goos: "windows", ctrName: "claude_cli_42", homeSub: "claude-home",
			want: []string{
				"run", "--rm", "-it",
				"-v", host + ":" + work,
				"-v", home + "/claude-home:/root",
				"-w", work,
				"-e", "TERM=xterm-256color", "-e", "COLORTERM=truecolor",
				"-e", "GIT_CONFIG_COUNT=2",
				"-e", "GIT_CONFIG_KEY_0=core.autocrlf", "-e", "GIT_CONFIG_VALUE_0=true",
				"-e", "GIT_CONFIG_KEY_1=core.filemode", "-e", "GIT_CONFIG_VALUE_1=false",
				"-e", "ANTHROPIC_API_KEY",
				"--name", "claude_cli_42",
				"aicontained-claude", "--version",
			},
		},
		{
			name: "codex/linux", tool: "codex", goos: "linux", ctrName: "codex_cli_42", homeSub: "codex-home",
			want: []string{
				"run", "--rm", "-it",
				"-v", host + ":" + work,
				"-v", home + "/codex-home:/root",
				"-w", work,
				"-e", "TERM", "-e", "COLORTERM",
				"-e", "OPENAI_API_KEY",
				"--security-opt", "no-new-privileges",
				"--cap-drop=ALL",
				"--cap-add=CHOWN,DAC_OVERRIDE,FOWNER,FSETID,SETUID,SETGID",
				"--name", "codex_cli_42",
				"aicontained-codex", "--version",
			},
		},
		{
			name: "copilot/darwin", tool: "copilot", goos: "darwin", ctrName: "copilot_cli_42", homeSub: "copilot-home",
			want: []string{
				"run", "--rm", "-it", "--progress", "none",
				"-v", host + ":" + work,
				"-v", home + "/copilot-home:/root",
				"-w", work,
				"-e", "TERM", "-e", "COLORTERM",
				"--name", "copilot_cli_42",
				"aicontained-copilot", "--version",
			},
		},
		{
			name: "agy/windows", tool: "agy", goos: "windows", ctrName: "agy_cli_7", homeSub: "antigravity-home",
			want: []string{
				"run", "--rm", "-it",
				"-v", host + ":" + work,
				"-v", home + "/antigravity-home:/root",
				"-w", work,
				"-e", "TERM=xterm-256color", "-e", "COLORTERM=truecolor",
				"-e", "GIT_CONFIG_COUNT=2",
				"-e", "GIT_CONFIG_KEY_0=core.autocrlf", "-e", "GIT_CONFIG_VALUE_0=true",
				"-e", "GIT_CONFIG_KEY_1=core.filemode", "-e", "GIT_CONFIG_VALUE_1=false",
				"-e", "GEMINI_API_KEY", "-e", "GOOGLE_API_KEY", "-e", "GOOGLE_GENAI_USE_VERTEXAI",
				"--name", "agy_cli_7",
				"aicontained-agy", "--version",
			},
		},
		{
			// opencode is the only tool with the per-run state dir mount.
			name: "opencode/linux", tool: "opencode", goos: "linux", ctrName: "opencode_cli_42", stateDir: home + "/opencode-home/.local/state/opencode-1234", homeSub: "opencode-home",
			want: []string{
				"run", "--rm", "-it",
				"-v", host + ":" + work,
				"-v", home + "/opencode-home:/root",
				"-v", home + "/opencode-home/.local/state/opencode-1234:/root/.local/state/opencode",
				"-w", work,
				"-e", "TERM", "-e", "COLORTERM",
				"-e", "ANTHROPIC_API_KEY", "-e", "OPENAI_API_KEY",
				"-e", "GEMINI_API_KEY", "-e", "GOOGLE_API_KEY", "-e", "GOOGLE_GENAI_USE_VERTEXAI",
				"--security-opt", "no-new-privileges",
				"--cap-drop=ALL",
				"--cap-add=CHOWN,DAC_OVERRIDE,FOWNER,FSETID,SETUID,SETGID",
				"--name", "opencode_cli_42",
				"aicontained-opencode", "--version",
			},
		},
		{
			// pi passes only the keys its provider docs read and, like every
			// tool but opencode, has no per-run state dir mount.
			name: "pi/linux", tool: "pi", goos: "linux", ctrName: "pi_cli_42", homeSub: "pi-home",
			want: []string{
				"run", "--rm", "-it",
				"-v", host + ":" + work,
				"-v", home + "/pi-home:/root",
				"-w", work,
				"-e", "TERM", "-e", "COLORTERM",
				"-e", "ANTHROPIC_API_KEY", "-e", "OPENAI_API_KEY",
				"-e", "GEMINI_API_KEY",
				"--security-opt", "no-new-privileges",
				"--cap-drop=ALL",
				"--cap-add=CHOWN,DAC_OVERRIDE,FOWNER,FSETID,SETUID,SETGID",
				"--name", "pi_cli_42",
				"aicontained-pi", "--version",
			},
		},
		{
			// `aicontained shell`: same run, bash overrides the entrypoint.
			name: "claude-shell/linux", tool: "claude", goos: "linux", ctrName: "claude_shell_42", homeSub: "claude-home", entry: "bash",
			want: []string{
				"run", "--rm", "-it",
				"-v", host + ":" + work,
				"-v", home + "/claude-home:/root",
				"-w", work,
				"-e", "TERM", "-e", "COLORTERM",
				"-e", "ANTHROPIC_API_KEY",
				"--security-opt", "no-new-privileges",
				"--cap-drop=ALL",
				"--cap-add=CHOWN,DAC_OVERRIDE,FOWNER,FSETID,SETUID,SETGID",
				"--entrypoint", "bash",
				"--name", "claude_shell_42",
				"aicontained-claude", "--version",
			},
		},
		{
			name: "opencode-shell/darwin", tool: "opencode", goos: "darwin", ctrName: "opencode_shell_42", stateDir: home + "/opencode-home/.local/state/opencode-1234", homeSub: "opencode-home", entry: "bash",
			want: []string{
				"run", "--rm", "-it", "--progress", "none",
				"-v", host + ":" + work,
				"-v", home + "/opencode-home:/root",
				"-v", home + "/opencode-home/.local/state/opencode-1234:/root/.local/state/opencode",
				"-w", work,
				"-e", "TERM", "-e", "COLORTERM",
				"-e", "ANTHROPIC_API_KEY", "-e", "OPENAI_API_KEY",
				"-e", "GEMINI_API_KEY", "-e", "GOOGLE_API_KEY", "-e", "GOOGLE_GENAI_USE_VERTEXAI",
				"--entrypoint", "bash",
				"--name", "opencode_shell_42",
				"aicontained-opencode", "--version",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tool, ok := tools.Lookup(c.tool)
			if !ok {
				t.Fatalf("unknown tool %q", c.tool)
			}
			got, err := assemble(tool, c.goos, host, work, work, home+"/"+c.homeSub, c.stateDir, c.sshAgent, c.ctrName, c.entry, []string{"--version"})
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(c.want) {
				t.Fatalf("got:\n  %q\nwant:\n  %q", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("arg %d: got %q, want %q\nfull got: %q", i, got[i], c.want[i], got)
					break
				}
			}
		})
	}
}

// paths mounts the repo root inside a git repo and opens the tool in the
// caller's subdirectory; outside a repo it maps the directory itself.
func TestPaths(t *testing.T) {
	// Git reports the symlink-resolved root (macOS: /var → /private/var),
	// so compare against the resolved path.
	repo, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "-C", repo, "init", "-q")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	sub := filepath.Join(repo, "cmd", "foo")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	// Repo root is the mount; the tool opens in the subdir.
	mount, mountDst, cwd, err := paths(sub)
	if err != nil {
		t.Fatal(err)
	}
	if mount != repo {
		t.Errorf("paths(sub) mount = %q, want %q", mount, repo)
	}
	if mountDst != workdir.ContainerPath(repo) {
		t.Errorf("paths(sub) mountDst = %q, want %q", mountDst, workdir.ContainerPath(repo))
	}
	wantSub := filepath.ToSlash(filepath.Join(workdir.ContainerPath(repo), "cmd", "foo"))
	if cwd != wantSub {
		t.Errorf("paths(sub) cwd = %q, want %q", cwd, wantSub)
	}

	// Repo root itself: cwd stays at the mount point.
	mount, mountDst, cwd, err = paths(repo)
	if err != nil {
		t.Fatal(err)
	}
	if mount != repo || mountDst != workdir.ContainerPath(repo) || cwd != workdir.ContainerPath(repo) {
		t.Errorf("paths(repo) = %q, %q, %q, want %q, %q, %q", mount, mountDst, cwd, repo, workdir.ContainerPath(repo), workdir.ContainerPath(repo))
	}

	// A subdirectory whose name starts with ".." is still inside the repo.
	dotdot := filepath.Join(repo, "..foo")
	if err := os.MkdirAll(dotdot, 0o755); err != nil {
		t.Fatal(err)
	}
	mount, mountDst, cwd, err = paths(dotdot)
	if err != nil {
		t.Fatal(err)
	}
	if mount != repo || mountDst != workdir.ContainerPath(repo) || cwd != filepath.ToSlash(filepath.Join(workdir.ContainerPath(repo), "..foo")) {
		t.Errorf("paths(..foo) = %q, %q, %q", mount, mountDst, cwd)
	}

	// Outside a repo: today's behavior.
	plain := t.TempDir()
	mount, mountDst, cwd, err = paths(plain)
	if err != nil {
		t.Fatal(err)
	}
	if mount != plain || mountDst != workdir.ContainerPath(plain) || cwd != workdir.ContainerPath(plain) {
		t.Errorf("paths(plain) = %q, %q, %q, want %q, %q, %q", mount, mountDst, cwd, plain, workdir.ContainerPath(plain), workdir.ContainerPath(plain))
	}

	// A symlink resolving outside any repo maps to the resolved directory
	// itself, not the repo.
	other := t.TempDir()
	link := filepath.Join(repo, "outside")
	if err := os.Symlink(other, link); err != nil && !strings.Contains(err.Error(), "not supported") {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(link)
	if err != nil {
		t.Skip("symlinks unsupported")
	}
	mount, mountDst, cwd, err = paths(resolved)
	if err != nil {
		t.Fatal(err)
	}
	if mount != resolved || mountDst != workdir.ContainerPath(resolved) || cwd != workdir.ContainerPath(resolved) {
		t.Errorf("paths(resolved outside) = %q, %q, %q, want %q, %q, %q", mount, mountDst, cwd, resolved, workdir.ContainerPath(resolved), workdir.ContainerPath(resolved))
	}

	// A symlinked path into the repo: git resolves the root to the real path,
	// the host cwd no longer sits under it — refuse instead of guessing.
	intoLink := filepath.Join(t.TempDir(), "repo-link")
	if err := os.Symlink(repo, intoLink); err != nil && !strings.Contains(err.Error(), "not supported") {
		t.Fatal(err)
	}
	linkSub := filepath.Join(intoLink, "cmd", "foo")
	if _, err := os.Stat(linkSub); err != nil {
		t.Skip("symlinks unsupported")
	}
	if _, _, _, err := paths(linkSub); err == nil {
		t.Errorf("paths(%q linked into repo) = nil error, want escape error", linkSub)
	}
}

// A colon in the host path breaks `-v` (parsed as the mount-options
// separator); the fallback to --mount, the Windows exemption, and the
// unsolvable colon+comma combination are pinned here (issue #52).
func TestAssembleMountFallback(t *testing.T) {
	tool, ok := tools.Lookup("claude")
	if !ok {
		t.Fatal("unknown tool claude")
	}
	const work = "/workspace/HASH/proj"

	// Linux: --mount replaces -v.
	got, err := assemble(tool, "linux", "/home/u/proj:v2", work, work, "/home/u/.aicontained/claude-home", "", "", "", "", []string{"--version"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--mount", "type=bind,src=/home/u/proj:v2,dst=" + work, "-v", "/home/u/.aicontained/claude-home:/root", "-w", work}
	for i, w := range want {
		if got[3+i] != w {
			t.Errorf("arg %d: got %q, want %q\nfull got: %q", 3+i, got[3+i], w, got)
		}
	}

	// Windows keeps -v even with a colon (wslc handles drive letters).
	got, err = assemble(tool, "windows", "/home/u/proj:v2", work, work, "/home/u/.aicontained/claude-home", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got[3] != "-v" || got[4] != "/home/u/proj:v2:"+work {
		t.Errorf("windows colon path: got %q, want plain -v mount", got[3:5])
	}

	// Colon + comma: not mountable on any non-Windows engine; must error
	// clearly instead of failing inside podman.
	_, err = assemble(tool, "linux", "/home/u/proj:v2,final", work, work, "/home/u/.aicontained/claude-home", "", "", "", "", nil)
	if err == nil {
		t.Fatal("want an error for a path containing both ':' and ','")
	}
}

const agentSock = "/run/user/1000/agent.sock"

func TestAssembleSSHAgent(t *testing.T) {
	tool, ok := tools.Lookup("claude")
	if !ok {
		t.Fatal("unknown tool claude")
	}
	const work = "/workspace/HASH/proj"

	// Linux: socket mount plus the container-side env value.
	got, err := assemble(tool, "linux", "/home/u/proj", work, work, "/home/u/.aicontained/claude-home", "", agentSock, "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"-v", agentSock + ":/var/run/ssh-agent",
		"-e", "SSH_AUTH_SOCK=/var/run/ssh-agent",
	}
	for _, w := range want {
		if !slices.Contains(got, w) {
			t.Errorf("linux: missing %q\nfull got: %q", w, got)
		}
	}

	// macOS: same.
	got, err = assemble(tool, "darwin", "/home/u/proj", work, work, "/home/u/.aicontained/claude-home", "", agentSock, "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range want {
		if !slices.Contains(got, w) {
			t.Errorf("darwin: missing %q\nfull got: %q", w, got)
		}
	}

	// Windows: never mounted, never set.
	got, err = assemble(tool, "windows", "/home/u/proj", work, work, "/home/u/.aicontained/claude-home", "", agentSock, "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range got {
		if a == agentSock || a == "SSH_AUTH_SOCK=/var/run/ssh-agent" {
			t.Errorf("windows: agent leaked into args: %q", got)
		}
	}
}

func TestSSHAgentSock(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	if got := sshAgentSock(); got != "" {
		t.Errorf("unset var: got %q, want \"\"", got)
	}

	// A real unix socket passes; a missing path does not.
	dir := t.TempDir()
	sock := filepath.Join(dir, "agent.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	t.Setenv("SSH_AUTH_SOCK", sock)
	want := sock
	if runtime.GOOS == "windows" { // passthrough is off on Windows even with a live socket
		want = ""
	}
	if got := sshAgentSock(); got != want {
		t.Errorf("live socket: got %q, want %q", got, want)
	}

	missing := filepath.Join(dir, "missing.sock")
	t.Setenv("SSH_AUTH_SOCK", missing)
	if got := sshAgentSock(); got != "" {
		t.Errorf("missing socket: got %q, want \"\"", got)
	}
}

func TestHomeDir(t *testing.T) {
	claude := tools.Tool{Name: "claude", HomeDir: "claude-home"}
	cases := []struct {
		name, conf, want string // want "" means an error is expected
	}{
		{"no file", "", "claude-home"},
		{"no tool table", "[codex]\nprofile = \"x\"\n", "claude-home"},
		{"empty profile", "[claude]\nprofile = \"\"\n", "claude-home"},
		{"profile", "[claude]\nprofile = \"personal\"\n", "claude-home-personal"},
		{"invalid profile", "[claude]\nprofile = \"../x\"\n", ""},
		{"dot profile", "[claude]\nprofile = \"..\"\n", ""},
		{"trailing dot", "[claude]\nprofile = \"x.\"\n", ""},
		{"case-insensitive", "[claude]\nprofile = \"Personal\"\n", "claude-home-personal"},
		{"case-insensitive table", "[Claude]\nprofile = \"x\"\n", "claude-home-x"},
		{"duplicate table", "[claude]\nprofile = \"a\"\n[Claude]\nprofile = \"b\"\n", ""},
		{"typo key", "[claude]\nprofil = \"x\"\n", ""},
		{"unknown table ignored", "[claud]\nprofile = \"x\"\n", "claude-home"},
		{"malformed", "[claude\n", ""},
		{"top-level key", "profile = \"x\"\n", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			base := t.TempDir()
			if c.name != "no file" {
				if err := os.WriteFile(filepath.Join(base, "aicontained.toml"), []byte(c.conf), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got, err := homeDir(base, claude)
			if c.want == "" {
				if err == nil {
					t.Fatalf("want error, got %q", got)
				}
				return
			}
			if err != nil || got != filepath.Join(base, c.want) {
				t.Fatalf("got %q, %v; want %q", got, err, c.want)
			}
		})
	}
}
