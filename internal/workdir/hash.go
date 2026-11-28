// Package workdir maps the host project directory to its unique
// in-container path, so per-project tool state (sessions, trust, memory)
// never mixes projects.
package workdir

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// HostPath returns the canonical absolute path of the current directory
// (pwd -P equivalent, symlinks resolved).
func HostPath() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}
	return dir, nil
}

// GitRoot returns the git work-tree root containing dir, or "" when dir is
// not inside a repo (or git is unavailable/misbehaves).
func GitRoot(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	root := strings.TrimSpace(string(out))
	if root == "" || !filepath.IsAbs(root) {
		return ""
	}
	// Git prints forward slashes even on Windows; use native separators.
	return filepath.FromSlash(root)
}

// ContainerPath returns the in-container mount point for hostDir:
// /workspace/<sha256(hostDir), first 12 hex>/<sanitized folder name>.
// Unified across OSes (Windows launchers used to lowercase the path first)
// — existing per-project state resets once.
func ContainerPath(hostDir string) string {
	sum := sha256.Sum256([]byte(hostDir))
	name := sanitize(filepath.Base(hostDir))
	return "/workspace/" + hex.EncodeToString(sum[:])[:12] + "/" + name
}

func sanitize(name string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9',
			r == '.', r == '_', r == '-':
			return r
		default:
			return '_'
		}
	}, name)
}
