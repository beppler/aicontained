package main

import "testing"

func TestToolName(t *testing.T) {
	for _, tt := range []struct {
		argv0 string
		want  string
	}{
		{"claude", "claude"},
		{"claude.exe", "claude"},
		{"CLAUDE", "claude"},
		{"CLAUDE.EXE", "claude"},
		{"/usr/local/bin/codex", "codex"},
		{"cli-tools", "cli-tools"},
		{"cli-tools-linux-amd64", "cli-tools-linux-amd64"},
	} {
		if got := toolName(tt.argv0); got != tt.want {
			t.Errorf("toolName(%q) = %q, want %q", tt.argv0, got, tt.want)
		}
	}
}

func TestToolFor(t *testing.T) {
	for _, tt := range []struct {
		argv0 string
		want  string
		ok    bool
	}{
		{"claude", "claude", true},
		{"claude.exe", "claude", true},
		{"CLAUDE", "claude", true},
		{"CLAUDE.EXE", "claude", true},
		{"/usr/local/bin/codex", "codex", true},
		{"agy", "agy", true},
		{"cli-tools", "", false},
		{"cli-tools-linux-amd64", "", false},
		{"/opt/cli-tools-darwin-arm64", "", false},
	} {
		tool, ok := toolFor(tt.argv0)
		if ok != tt.ok || tool.Name != tt.want {
			t.Errorf("toolFor(%q) = (%q, %v), want (%q, %v)", tt.argv0, tool.Name, ok, tt.want, tt.ok)
		}
	}
}
