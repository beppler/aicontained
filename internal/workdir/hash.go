// Package workdir maps the host project directory to its unique
// in-container path, so per-project tool state (sessions, trust, memory)
// never mixes projects.
package workdir

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

// HostPath returns the canonical absolute path of the current directory
// (pwd -P equivalent, symlinks resolved).
func HostPath() string {
	dir, err := os.Getwd()
	if err != nil {
		return dir
	}
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}
	return dir
}

// ContainerPath returns the in-container mount point for hostDir:
// /workspace/<sha256(hostDir), first 12 hex>/<sanitized folder name>.
// Unified across OSes (the legacy Windows wrappers lowercased the path
// first) — existing per-project state resets once.
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
