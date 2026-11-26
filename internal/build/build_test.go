package build

import (
	"testing"

	"github.com/beppler/aicontained/internal/images"
)

// The binary builds images from the embedded definitions, not a repo
// checkout — they must actually be in the binary.
func TestEmbeddedContainers(t *testing.T) {
	for _, f := range []string{"containers/base/Dockerfile", "containers/claude/Dockerfile", "containers/opencode/Dockerfile", "containers/pi/Dockerfile"} {
		if _, err := images.Containers.ReadFile(f); err != nil {
			t.Errorf("embedded %s: %v", f, err)
		}
	}
}
