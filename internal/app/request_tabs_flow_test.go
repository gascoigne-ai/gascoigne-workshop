package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"reflect"
	"strings"
	"testing"
)

func TestNewRequestAndSwitchingTabs(t *testing.T) {
	a := testApp(t)
	a.newRequest()
	if len(a.tabs.Tabs()) != 2 || a.tabs.Active() != 1 || a.cur().Req.Name != "New Request" {
		t.Fatalf("tabs %d active %d", len(a.tabs.Tabs()), a.tabs.Active())
	}
	click(a, 280, headerH+tabsH/2) // the first tab sits at the left edge of the strip
	if a.tabs.Active() != 0 {
		t.Fatalf("active = %d, want 0", a.tabs.Active())
	}
}

func TestClosingATabBeforeTheActiveOneKeepsTheSameTabActive(t *testing.T) {
	a := twoTabs(t)
	a.closeTab(0)
	if a.tabs.Active() != 0 || a.cur().Req.Name != "Repository Info" {
		t.Fatalf("active %d current %q", a.tabs.Active(), a.cur().Req.Name)
	}
}

func TestDirtyFollowsEditsAndUndoingThemByHand(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	if dirty(tab) {
		t.Fatal("a freshly opened request is clean")
	}
	click(a, 1100, urlBarY)
	a.typeText("x")
	if !dirty(tab) {
		t.Fatal("editing the URL makes the tab dirty")
	}
	a.Step(core.Input{Edits: []core.Edit{core.EditBackspace}})
	if dirty(tab) {
		t.Fatal("restoring the original text makes it clean again")
	}
	click(a, 739, paramsRow0Y) // delete a param
	if !dirty(tab) {
		t.Fatal("deleting a param is a change")
	}
}

func TestTabMenu(t *testing.T) {
	a := twoTabs(t)
	left, _ := a.tabX(0)
	rightClick(a, left+20, tabStripY)
	want := []string{"Close", "Close others", "Close tabs to the right", "", "Duplicate", "Copy URL", "Cookies..."}
	if got := menuLabels(a); !reflect.DeepEqual(got, want) {
		t.Fatalf("tab menu = %q", got)
	}
	pickMenu(t, a, "Close tabs to the right")
	if len(a.tabs.Tabs()) != 1 || a.cur().Req.Name != "Search Repos" {
		t.Fatalf("%d tabs, current %q", len(a.tabs.Tabs()), a.cur().Req.Name)
	}
	rightClick(a, left+20, tabStripY)
	if !itemByLabel(t, a, "Close others").Disabled || !itemByLabel(t, a, "Close tabs to the right").Disabled {
		t.Fatal("with one tab there is nothing else to close")
	}
}

func TestCloseOthersKeepsOnlyThatTab(t *testing.T) {
	a := twoTabs(t)
	a.newRequest() // three tabs
	left, _ := a.tabX(1)
	keep := a.openTabs()[1]
	rightClick(a, left+20, tabStripY)
	pickMenu(t, a, "Close others")
	if len(a.tabs.Tabs()) != 1 || a.openTabs()[0] != keep || a.cur() != keep {
		t.Fatalf("%d tabs, current %q", len(a.tabs.Tabs()), a.cur().Req.Name)
	}
}

func TestDuplicateTabAddsACopyToTheTreeAndOpensIt(t *testing.T) {
	a := testApp(t)
	left, _ := a.tabX(0)
	rightClick(a, left+20, tabStripY)
	pickMenu(t, a, "Duplicate")
	if got := requestNames(a.col().Folders[1]); !reflect.DeepEqual(got, []string{"Search Repos", "Search Repos copy", "Repository Info", "Repository Tags"}) {
		t.Fatalf("Repository = %v", got)
	}
	if a.cur().Req.Name != "Search Repos copy" || len(a.tabs.Tabs()) != 2 {
		t.Fatalf("current %q, %d tabs", a.cur().Req.Name, len(a.tabs.Tabs()))
	}
}

func TestTabMenuCopyURL(t *testing.T) {
	a := testApp(t)
	clip := &fakeClip{}
	a.ctx.Clipboard = clip
	left, _ := a.tabX(0)
	rightClick(a, left+20, tabStripY)
	pickMenu(t, a, "Copy URL")
	if !strings.HasPrefix(clip.text, "https://api.github.com/search/repositories?q=gioui") {
		t.Fatalf("clipboard = %q", clip.text)
	}
}

// newRequestX is the middle of the "+" at the right end of the strip.
const newRequestX = winW - 20

// The "+" and each tab's cross say what they do, with their shortcuts.
func TestTheRequestTabsButtonsHaveTooltips(t *testing.T) {
	a := twoTabs(t)
	t.Cleanup(a.WaitWrites)
	left, width := a.tabX(0)
	cases := []struct {
		name string
		x    int
		tip  string
	}{
		{"new request", newRequestX, "New request (Ctrl+T)"},
		{"close", left + width - 12, "Close tab (Ctrl+W)"},
	}
	for _, tc := range cases {
		if got := tipAt(a, tc.x, tabStripY); got != tc.tip {
			t.Errorf("%s: tooltip %q, want %q", tc.name, got, tc.tip)
		}
	}
}

// Right-clicking a tab opens its menu where the pointer is.
func TestRightClickingATabOpensItsMenuAtThePointer(t *testing.T) {
	a := twoTabs(t)
	t.Cleanup(a.WaitWrites)
	left, _ := a.tabX(1)
	rightClick(a, left+20, tabStripY)
	menu := widget.PopupMenu(a.ctx)
	if menu == nil || menuLabels(a)[0] != "Close" {
		t.Fatalf("no tab menu after the right click: %v", menu)
	}
	if menu.X != left+20 || menu.Y != tabStripY {
		t.Errorf("menu at (%d, %d), want the pointer (%d, %d)", menu.X, menu.Y, left+20, tabStripY)
	}
}

// openNamed opens the sample's requests called names, each in its tab.
func openNamed(t *testing.T, app *App, names ...string) {
	t.Helper()
	for _, name := range names {
		request := app.findRequestByName(name)
		if request == nil {
			t.Fatalf("no request %q in the sample", name)
		}
		app.openRequest(request)
	}
}
