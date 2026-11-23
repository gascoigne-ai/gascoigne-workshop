package app

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Sizes the panes of the request editor are pinned at: the request pane of
// the test window and the narrowest main area.
const (
	settingsWideW   = winW - sidebarW
	settingsNarrowW = minMainW
	settingsPaneH   = 300
)

// Right-clicking the timeout field opens the text menu.
func TestTheTimeoutFieldHasTheTextMenu(t *testing.T) {
	a := openSettingsPane(t)
	t.Cleanup(a.WaitWrites)
	rightClick(a, timeoutFieldX, chipsY)
	if widget.PopupMenu(a.ctx) == nil || menuLabels(a)[0] != "Undo" {
		t.Fatalf("menu = %v", widget.PopupMenu(a.ctx))
	}
}

// The certificate row is one row and the pane's gap below the redirects row.
const verifyRowY = followRowY + 36

func TestTheVerifyCertificateRowTogglesInsecureTLS(t *testing.T) {
	a := openSettingsPane(t)
	t.Cleanup(a.WaitWrites)
	tab := a.cur()
	click(a, followRowX, verifyRowY)
	if !tab.Req.InsecureTLS {
		t.Fatal("the checkbox row must turn verification off")
	}
	click(a, followRowX, verifyRowY)
	if tab.Req.InsecureTLS {
		t.Fatal("a second click must turn it back on")
	}
}
