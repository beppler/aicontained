package launcher

import (
	"testing"

	"github.com/beppler/cli-tools/internal/tools"
)

// Golden test pinning the exact engine flags per platform, ported from the
// legacy bin/* scripts.
func TestAssemble(t *testing.T) {
	host, work, home := "/home/u/proj", "/workspace/HASH/proj", "/home/u/.cli-tools"
	cases := []struct {
		name     string
		tool     string
		goos     string
		ctrName  string
		stateDir string
		homeSub  string
		want     []string
	}{
		{
			name: "claude/darwin", tool: "claude", goos: "darwin", homeSub: "claude-home",
			want: []string{
				"run", "--rm", "-it", "--progress", "none",
				"-v", host + ":" + work, "-w", work,
				"-v", home + "/claude-home:/root",
				"-e", "TERM", "-e", "COLORTERM",
				"-e", "ANTHROPIC_API_KEY",
				"cli-tools-claude", "--version",
			},
		},
		{
			name: "claude/linux", tool: "claude", goos: "linux", homeSub: "claude-home",
			want: []string{
				"run", "--rm", "-it",
				"-v", host + ":" + work, "-w", work,
				"-v", home + "/claude-home:/root",
				"-e", "TERM", "-e", "COLORTERM",
				"-e", "ANTHROPIC_API_KEY",
				"--security-opt", "no-new-privileges",
				"--cap-drop=ALL",
				"--cap-add=CHOWN,DAC_OVERRIDE,FOWNER,FSETID,SETUID,SETGID",
				"cli-tools-claude", "--version",
			},
		},
		{
			name: "claude/windows", tool: "claude", goos: "windows", ctrName: "claude_cli_42", homeSub: "claude-home",
			want: []string{
				"run", "--rm", "-it",
				"-v", host + ":" + work, "-w", work,
				"-v", home + "/claude-home:/root",
				"-e", "TERM=xterm-256color", "-e", "COLORTERM=truecolor",
				"-e", "GIT_CONFIG_COUNT=2",
				"-e", "GIT_CONFIG_KEY_0=core.autocrlf", "-e", "GIT_CONFIG_VALUE_0=true",
				"-e", "GIT_CONFIG_KEY_1=core.filemode", "-e", "GIT_CONFIG_VALUE_1=false",
				"-e", "ANTHROPIC_API_KEY",
				"--name", "claude_cli_42",
				"cli-tools-claude", "--version",
			},
		},
		{
			name: "codex/linux", tool: "codex", goos: "linux", homeSub: "codex-home",
			want: []string{
				"run", "--rm", "-it",
				"-v", host + ":" + work, "-w", work,
				"-v", home + "/codex-home:/root",
				"-e", "TERM", "-e", "COLORTERM",
				"-e", "OPENAI_API_KEY",
				"--security-opt", "no-new-privileges",
				"--cap-drop=ALL",
				"--cap-add=CHOWN,DAC_OVERRIDE,FOWNER,FSETID,SETUID,SETGID",
				"cli-tools-codex", "--version",
			},
		},
		{
			name: "copilot/darwin", tool: "copilot", goos: "darwin", homeSub: "copilot-home",
			want: []string{
				"run", "--rm", "-it", "--progress", "none",
				"-v", host + ":" + work, "-w", work,
				"-v", home + "/copilot-home:/root",
				"-e", "TERM", "-e", "COLORTERM",
				"cli-tools-copilot", "--version",
			},
		},
		{
			name: "agy/windows", tool: "agy", goos: "windows", ctrName: "agy_cli_7", homeSub: "antigravity-home",
			want: []string{
				"run", "--rm", "-it",
				"-v", host + ":" + work, "-w", work,
				"-v", home + "/antigravity-home:/root",
				"-e", "TERM=xterm-256color", "-e", "COLORTERM=truecolor",
				"-e", "GIT_CONFIG_COUNT=2",
				"-e", "GIT_CONFIG_KEY_0=core.autocrlf", "-e", "GIT_CONFIG_VALUE_0=true",
				"-e", "GIT_CONFIG_KEY_1=core.filemode", "-e", "GIT_CONFIG_VALUE_1=false",
				"-e", "GEMINI_API_KEY", "-e", "GOOGLE_API_KEY", "-e", "GOOGLE_GENAI_USE_VERTEXAI",
				"--name", "agy_cli_7",
				"cli-tools-agy", "--version",
			},
		},
		{
			// opencode is the only tool with the per-run state dir mount.
			name: "opencode/linux", tool: "opencode", goos: "linux", stateDir: home + "/opencode-home/.local/state/opencode-1234", homeSub: "opencode-home",
			want: []string{
				"run", "--rm", "-it",
				"-v", host + ":" + work, "-w", work,
				"-v", home + "/opencode-home:/root",
				"-v", home + "/opencode-home/.local/state/opencode-1234:/root/.local/state/opencode",
				"-e", "TERM", "-e", "COLORTERM",
				"-e", "ANTHROPIC_API_KEY", "-e", "OPENAI_API_KEY",
				"-e", "GEMINI_API_KEY", "-e", "GOOGLE_API_KEY", "-e", "GOOGLE_GENAI_USE_VERTEXAI",
				"--security-opt", "no-new-privileges",
				"--cap-drop=ALL",
				"--cap-add=CHOWN,DAC_OVERRIDE,FOWNER,FSETID,SETUID,SETGID",
				"cli-tools-opencode", "--version",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tool, ok := tools.Lookup(c.tool)
			if !ok {
				t.Fatalf("unknown tool %q", c.tool)
			}
			got := assemble(tool, c.goos, host, work, home+"/"+c.homeSub, c.stateDir, c.ctrName, []string{"--version"})
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
