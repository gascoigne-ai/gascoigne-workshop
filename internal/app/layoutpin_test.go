package app

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// settingsDialogRects is the digest of view.Dump (kinds, attributes and
// every rect) of the settings dialog scene laid out at 1280x800. Layout
// work such as measure caching must not move a rect; a deliberate change
// of the scene updates the digest (the test prints the new one).
const settingsDialogRects = "ebccda590db8eb0f6dfc7acd3c9af401c581accd881afe3889701b5a5871ded9"

func TestSettingsDialogLaysOutToTheSameRects(t *testing.T) {
	prefs := settings.Default()
	prefs.Layout = "side"
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := settingsDisk.Save(path, prefs); err != nil {
		t.Fatal(err)
	}
	a, err := NewApp(withAdapters(Options{SettingsPath: path}))
	if err != nil {
		t.Fatal(err)
	}
	a.w, a.h = 1280, 800
	a.step(core.Input{})
	a.menu = true
	a.step(core.Input{})
	root := a.view()
	view.Layout(a.ctx, root, a.w, a.h)
	dump := strings.ReplaceAll(view.Dump(root), path, "/config/settings.json") // a temp dir differs every run
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(dump))); got != settingsDialogRects {
		t.Errorf("dialog scene digest = %s, want %s", got, settingsDialogRects)
	}
}
