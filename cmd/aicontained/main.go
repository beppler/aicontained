// Command aicontained runs containerized AI coding CLIs, Busybox-style: one
// binary, one command per tool via argv[0] (symlinked as claude, codex, ...).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/beppler/aicontained/internal/buildcmd"
	"github.com/beppler/aicontained/internal/launcher"
	"github.com/beppler/aicontained/internal/tools"
)

func main() {
	if tool, ok := toolFor(os.Args[0]); ok {
		runTool(tool, os.Args[1:])
	}

	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}
	switch args[0] {
	case "build":
		os.Exit(buildcmd.Build(args[1:]))
	case "install":
		os.Exit(buildcmd.Install(args[1:]))
	case "-h", "--help", "help":
		usage()
	case "--list", "list":
		for _, t := range tools.All() {
			fmt.Println(t.Name)
		}
	default:
		runFor(args[0], args[1:])
	}
}

// toolFor maps argv0 to a known tool command, or reports false when the
// invocation should be handled as the aicontained command itself. Known tool
// names win over the binary name, so a cross-built copy
// (aicontained-linux-amd64) still reaches the aicontained subcommands.
func toolFor(argv0 string) (tools.Tool, bool) {
	return tools.Lookup(toolName(argv0))
}

// toolName extracts the tool command name from argv0: the lowercased
// basename without a Windows .exe suffix.
func toolName(argv0 string) string {
	name := strings.ToLower(filepath.Base(argv0))
	return strings.TrimSuffix(name, ".exe")
}

func runTool(tool tools.Tool, args []string) {
	rc, err := launcher.Run(tool, args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "aicontained:", err)
	}
	os.Exit(rc)
}

func runFor(name string, args []string) {
	tool, ok := tools.Lookup(name)
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown tool: %s (available: %s)\n", name, strings.Join(names(), " "))
		os.Exit(1)
	}
	runTool(tool, args)
}

func names() []string {
	all := tools.All()
	out := make([]string, len(all))
	for i, t := range all {
		out[i] = t.Name
	}
	return out
}

func usage() {
	fmt.Fprintf(os.Stderr, `Runs AI coding CLIs in isolated containers.

Usage:
  claude|codex|copilot|agy|opencode|pi [args...]
  aicontained <tool> [args...]
  aicontained build [tool...] [--prune-images] [--no-cache]
  aicontained install [tool...]
  aicontained --list
`)
}
