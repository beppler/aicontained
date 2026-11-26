// Command aicontained runs containerized AI coding CLIs, Busybox-style: one
// binary, one command per tool via argv[0] (symlinked as claude, codex, ...).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/beppler/aicontained/internal/build"
	"github.com/beppler/aicontained/internal/install"
	"github.com/beppler/aicontained/internal/launcher"
	"github.com/beppler/aicontained/internal/tools"
)

func main() {
	if tool, ok := toolFor(os.Args[0]); ok {
		launch(tool, os.Args[1:], launcher.Run)
	}

	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}
	switch args[0] {
	case "build":
		os.Exit(build.Build(args[1:]))
	case "install":
		os.Exit(install.Install(args[1:]))
	case "shell":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: aicontained shell <tool> [bash args...] (available: %s)\n", tools.Names(tools.All()))
			os.Exit(1)
		}
		launch(lookup(args[1]), args[2:], launcher.Shell)
	case "-h", "--help", "help":
		usage()
	case "--list", "list":
		for _, t := range tools.All() {
			fmt.Println(t.Name)
		}
	default:
		launch(lookup(args[0]), args[1:], launcher.Run)
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

// launch runs tool via fn (launcher.Run or launcher.Shell) and exits with
// its code.
func launch(tool tools.Tool, args []string, fn func(tools.Tool, []string) (int, error)) {
	rc, err := fn(tool, args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "aicontained:", err)
	}
	os.Exit(rc)
}

// lookup finds the named tool, or exits with the list of known ones.
func lookup(name string) tools.Tool {
	tool, ok := tools.Lookup(name)
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown tool: %s (available: %s)\n", name, tools.Names(tools.All()))
		os.Exit(1)
	}
	return tool
}

func usage() {
	fmt.Fprintf(os.Stderr, `Runs AI coding CLIs in isolated containers.

Usage:
  claude|codex|copilot|agy|opencode|pi [args...]
  aicontained <tool> [args...]
  aicontained shell <tool> [bash args...]
  aicontained build [tool...] [--prune-images] [--no-cache]
  aicontained install [tool...]
  aicontained --list
`)
}
