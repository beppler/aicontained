package workdir

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestContainerPath(t *testing.T) {
	cases := []struct {
		host, want string
	}{
		// sha256("/home/user/proj")[:12] = 7d73bf4fdeae
		{"/home/user/proj", "/workspace/7d73bf4fdeae/proj"},
		// folder name is sanitized, hash covers the whole path
		{"/home/user/My Project!", "/workspace/148160982455/My_Project_"},
	}
	for _, c := range cases {
		if got := ContainerPath(c.host); got != c.want {
			t.Errorf("ContainerPath(%q) = %q, want %q", c.host, got, c.want)
		}
	}
	// Distinct projects must never share a container path.
	a, b := ContainerPath("/home/user/proj"), ContainerPath("/home/user/other")
	if a == b {
		t.Errorf("distinct paths hashed equal: %q", a)
	}
}

func TestGitRoot(t *testing.T) {
	// Git reports the symlink-resolved root (macOS: /var → /private/var),
	// so compare against the resolved path.
	repo, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", repo, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	sub := filepath.Join(repo, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := GitRoot(sub); got != repo {
		t.Errorf("GitRoot(%q) = %q, want %q", sub, got, repo)
	}
	if got := GitRoot(repo); got != repo {
		t.Errorf("GitRoot(root) = %q, want %q", got, repo)
	}
	if got := GitRoot(filepath.Dir(repo)); got != "" {
		t.Errorf("GitRoot(outside repo) = %q, want \"\"", got)
	}
}
