package window

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// settingsDialogRects is the digest of view.Dump (kinds, attributes and
// every rect) of the settings dialog scene laid out at 1280x800. Layout
// work such as measure caching must not move a rect; a deliberate change
// of the scene updates the digest (the test prints the new one).
const settingsDialogRects = "4f911dd8871201b3ff45f164f929f671bd59035beffa8737d24219fa0382a43b"

func TestSettingsDialogLaysOutToTheSameRects(t *testing.T) {
	prefs := settings.Default()
	prefs.Layout = "side"
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := settingsDisk.Save(path, prefs); err != nil {
		t.Fatal(err)
	}
	a, err := newWindow(withAdapters(startOptions{SettingsPath: path}))
	if err != nil {
		t.Fatal(err)
	}
	a.w, a.h = 1280, 800
	a.Step(core.Input{})
	a.over.Settings.Show()
	a.Step(core.Input{})
	root := a.view()
	view.Layout(a.ctx, root, a.w, a.h)
	dump := strings.ReplaceAll(view.Dump(root), path, "/config/settings.json") // a temp dir differs every run
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(dump))); got != settingsDialogRects {
		t.Errorf("dialog scene digest = %s, want %s", got, settingsDialogRects)
	}
}
