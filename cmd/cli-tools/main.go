// Command cli-tools runs containerized AI coding CLIs, Busybox-style: one
// binary, one command per tool via argv[0] (symlinked as claude, codex, ...).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/beppler/cli-tools/internal/launcher"
	"github.com/beppler/cli-tools/internal/tools"
)

func main() {
	name := strings.ToLower(filepath.Base(os.Args[0]))
	name = strings.TrimSuffix(name, ".exe")

	if name != "cli-tools" {
		runTool(name, os.Args[1:])
	}

	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}
	switch args[0] {
	case "-h", "--help", "help":
		usage()
	case "--list", "list":
		for _, t := range tools.All() {
			fmt.Println(t.Name)
		}
	default:
		runTool(args[0], args[1:])
	}
}

func runTool(name string, args []string) {
	tool, ok := tools.Lookup(name)
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown tool: %s (available: %s)\n", name, strings.Join(names(), " "))
		os.Exit(1)
	}
	rc, err := launcher.Run(tool, args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cli-tools:", err)
	}
	os.Exit(rc)
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
  claude|codex|copilot|agy|opencode [args...]
  cli-tools <tool> [args...]
  cli-tools --list
`)
}
