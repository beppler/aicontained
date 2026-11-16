package workdir

import "testing"

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
