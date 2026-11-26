package launcher

import (
	"net"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/beppler/aicontained/internal/tools"
)

// Golden test pinning the exact engine flags per platform.
func TestAssemble(t *testing.T) {
	host, work, home := "/home/u/proj", "/workspace/HASH/proj", "/home/u/.aicontained"
	cases := []struct {
		name     string
		tool      string
		goos      string
		ctrName   string
		stateDir  string
		sshAgent  string
		homeSub   string
		entry     string
		want      []string
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
			got, err := assemble(tool, c.goos, host, work, home+"/"+c.homeSub, c.stateDir, c.sshAgent, c.ctrName, c.entry, []string{"--version"})
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
	got, err := assemble(tool, "linux", "/home/u/proj:v2", work, "/home/u/.aicontained/claude-home", "", "", "", "", []string{"--version"})
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
	got, err = assemble(tool, "windows", "/home/u/proj:v2", work, "/home/u/.aicontained/claude-home", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got[3] != "-v" || got[4] != "/home/u/proj:v2:"+work {
		t.Errorf("windows colon path: got %q, want plain -v mount", got[3:5])
	}

	// Colon + comma: not mountable on any non-Windows engine; must error
	// clearly instead of failing inside podman.
	_, err = assemble(tool, "linux", "/home/u/proj:v2,final", work, "/home/u/.aicontained/claude-home", "", "", "", "", nil)
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
	got, err := assemble(tool, "linux", "/home/u/proj", work, "/home/u/.aicontained/claude-home", "", agentSock, "", "", nil)
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
	got, err = assemble(tool, "darwin", "/home/u/proj", work, "/home/u/.aicontained/claude-home", "", agentSock, "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range want {
		if !slices.Contains(got, w) {
			t.Errorf("darwin: missing %q\nfull got: %q", w, got)
		}
	}

	// Windows: never mounted, never set.
	got, err = assemble(tool, "windows", "/home/u/proj", work, "/home/u/.aicontained/claude-home", "", agentSock, "", "", nil)
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
	if got := sshAgentSock(); got != sock {
		t.Errorf("live socket: got %q, want %q", got, sock)
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
