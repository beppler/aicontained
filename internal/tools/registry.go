// Package tools holds the per-tool table: what image to run, which host
// directories to mount, and which environment keys to pass through.
package tools

// Tool describes one containerized CLI.
type Tool struct {
	Name       string   // command name; image is cli-tools-<name>
	HomeDir    string   // persistent home dir under ~/.cli-tools
	EnvKeys    []string // extra env keys passed through from the host
	StateMount bool     // mount a per-run state dir (opencode only)
}

var all = []Tool{
	{Name: "claude", HomeDir: "claude-home", EnvKeys: []string{"ANTHROPIC_API_KEY"}},
	{Name: "codex", HomeDir: "codex-home", EnvKeys: []string{"OPENAI_API_KEY"}},
	{Name: "copilot", HomeDir: "copilot-home"},
	{Name: "agy", HomeDir: "antigravity-home", EnvKeys: []string{"GEMINI_API_KEY", "GOOGLE_API_KEY", "GOOGLE_GENAI_USE_VERTEXAI"}},
	// OPENAI_API_KEY is passed even though the opencode image itself only
	// needs a single provider key: the README already promises it and the
	// registry spec in issue #27 lists it.
	{Name: "opencode", HomeDir: "opencode-home", StateMount: true,
		EnvKeys: []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GEMINI_API_KEY", "GOOGLE_API_KEY", "GOOGLE_GENAI_USE_VERTEXAI"}},
}

// All returns every known tool.
func All() []Tool { return all }

// Lookup finds a tool by its command name.
func Lookup(name string) (Tool, bool) {
	for _, t := range all {
		if t.Name == name {
			return t, true
		}
	}
	return Tool{}, false
}

// Image is the container image name for the tool.
func (t Tool) Image() string { return "cli-tools-" + t.Name }
