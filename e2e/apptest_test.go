//go:build !release

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/app/apptest"
)

// The whole window after the first frame is the golden the in-package
// tests pin for a plain test app, so the seam changes no dump byte.
func TestStartRunsAFrameWhoseDumpMatchesTheGolden(t *testing.T) {
	want, err := os.ReadFile(filepath.Join("..", "internal", "app", "testdata", "window-empty-wide.dump"))
	if err != nil {
		t.Fatal(err)
	}
	got := apptest.Dump(apptest.Start(t, realAdapters))
	if got != string(want) {
		t.Fatalf("dump differs from window-empty-wide.dump:\n%s", got)
	}
}

func TestClickOpensATabSeenInTheDump(t *testing.T) {
	a := apptest.Start(t, realAdapters)
	needle := `text="Repository Info"`
	before := len(apptest.Lines(a, needle))
	x, y := apptest.TextCenter(t, a, "Repository Info")
	apptest.Click(a, x, y)
	if after := len(apptest.Lines(a, needle)); after != before+1 {
		t.Fatalf("lines showing %s: %d before the click, %d after, want one more", needle, before, after)
	}
}

func TestStartSandboxesHome(t *testing.T) {
	apptest.Start(t, realAdapters)
	for _, name := range []string{"HOME", "XDG_CONFIG_HOME"} {
		if value := os.Getenv(name); !strings.HasPrefix(value, os.TempDir()) {
			t.Errorf("%s = %q, want a folder under the test temp dir", name, value)
		}
	}
}
