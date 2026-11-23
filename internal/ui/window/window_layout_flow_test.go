package window

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"testing"
)

const dividerY = 400 // anywhere along the tall dividers

func TestDraggingTheSidebarDividerResizesTheSidebar(t *testing.T) {
	a := testApp(t)
	drag(a, 259, 400, dividerY)
	if a.sidebarSplit.Size < 390 {
		t.Fatalf("sidebar size = %v", a.sidebarSplit.Size)
	}
	click(a, 350, treeRowY(4)) // x=350 was the main column before; it is the tree now
	if a.cur().Req.Name != "Repository Info" {
		t.Fatalf("a click at x=350 opened %q: the sidebar did not grow", a.cur().Req.Name)
	}
}

func TestSidebarKeepsAMinimumWidthAndDoubleClickResets(t *testing.T) {
	a := testApp(t)
	drag(a, 259, 5, dividerY)
	click(a, 150, treeRowY(4)) // the minimum is 160px wide
	if a.cur().Req.Name != "Repository Info" {
		t.Fatal("the sidebar must not shrink below its minimum")
	}
	click(a, 200, treeRowY(5))
	if a.cur().Req.Name == "Repository Tags" {
		t.Fatal("x=200 is outside a minimum-width sidebar")
	}

	a.Step(core.Input{X: 159, Y: dividerY, Click: true, Held: true, Clicks: 2})
	a.Step(core.Input{})
	if a.sidebarSplit.Size != 0 {
		t.Fatalf("double click must reset the width, Size = %v", a.sidebarSplit.Size)
	}
	click(a, 250, treeRowY(5))
	if a.cur().Req.Name != "Repository Tags" {
		t.Fatal("back at the default width x=250 is in the tree again")
	}
}

func TestDraggingThePaneDividerChangesTheShare(t *testing.T) {
	a := testApp(t)
	drag(a, 769, 600, dividerY)
	if s := a.paneSplit.Size; s <= 0.25 || s >= 0.4 {
		t.Fatalf("share = %v, want about 0.33 (the request pane is now narrower)", s)
	}
}

func TestResizeCursorOverTheDividers(t *testing.T) {
	a := testApp(t)
	for _, tt := range []struct {
		name string
		x, y int
		want core.Cursor
	}{
		{"sidebar divider", 259, dividerY, core.CursorResizeH},
		{"pane divider", 769, dividerY, core.CursorResizeH},
		{"elsewhere", 500, dividerY, core.CursorDefault},
	} {
		a.Step(core.Input{X: tt.x, Y: tt.y})
		if a.ctx.Cursor != tt.want {
			t.Errorf("%s: cursor %v, want %v", tt.name, a.ctx.Cursor, tt.want)
		}
	}
}

// Layout chooser at the bottom of the settings menu (below the text size row).
const ()

func TestLayoutSettingStacksThePanesAndIsSaved(t *testing.T) {
	a := testApp(t)
	path := a.SettingsPath()
	click(a, menuBtnX, menuBtnY)
	a.over.Settings.Shell.Search = widget.NewTextField("request and response")
	clickText(t, a, "Stacked", 0)
	if a.settings.Layout != "stacked" {
		t.Fatalf("Layout = %q", a.settings.Layout)
	}
	if saved, _ := settingsDisk.Load(path); saved.Layout != "stacked" {
		t.Fatalf("saved %+v", saved)
	}
	a.Step(core.Input{Escape: true}) // close the dialog; a click outside leaves it open
	clip := &fakeClip{}
	a.ctx.Clipboard = clip
	tab := a.cur()
	srv := jsonServer(t)
	sendAndWait(t, a, srv.URL)

	// stacked: the response pane is the lower half, so its toolbar is at y=522
	click(a, copyBtnX, 510+12)
	if clip.text != `{"ok":true}` {
		t.Fatalf("the Copy button was not where a stacked layout puts it, clipboard %q", clip.text)
	}
	_ = tab
	click(a, menuBtnX, menuBtnY)
	a.over.Settings.Shell.Search = widget.NewTextField("request and response") // the reopened dialog is fresh
	clickText(t, a, "Side by side", 0)
	if a.settings.Layout != "side" {
		t.Fatalf("Layout = %q", a.settings.Layout)
	}
}
