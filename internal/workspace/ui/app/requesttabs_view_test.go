package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/tabs"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Widths the strip of request tabs is pinned at: the main column of the
// test window, and the narrowest main column the window allows.
const (
	requestTabsWideW   = winW - sidebarW
	requestTabsNarrowW = minMainW
)

// requestTabsState is a named look of the strip of open requests and how
// to give the app that look.
type requestTabsState struct {
	name  string
	setUp func(t *testing.T, a *App)
}

// requestTabsStates are the looks of the strip: one tab, several with the
// active one between others, a changed tab with a long name and other
// badges, and more tabs than the strip has room for.
var requestTabsStates = []requestTabsState{
	{"one", func(t *testing.T, a *App) {}},
	{"several", func(t *testing.T, a *App) {
		openNamed(t, a, "Repository Info", "Echo", "List Tools")
		a.updateTabs(tabs.Activate{Index: 1})
	}},
	{"dirty-long", func(t *testing.T, a *App) {
		openNamed(t, a, "Echo")
		long := a.openTabs()[0]
		long.Req.Name = "Search repositories by stars, newest first, then by name"
		long.Req.Method = "DELETE"
		a.updateTabs(tabs.Activate{Index: 0})
		click(a, urlFieldX, urlBarY)
		a.typeText("&page=2")
		if !a.tabDirty(long) {
			t.Fatal("test setup: the tab must be marked changed")
		}
	}},
	{"overflow", func(t *testing.T, a *App) {
		for range 14 {
			a.newRequest()
		}
	}},
}

// openNamed opens the sample's requests called names, each in its tab.
func openNamed(t *testing.T, a *App, names ...string) {
	t.Helper()
	for _, name := range names {
		request := a.findRequestByName(name)
		if request == nil {
			t.Fatalf("no request %q in the sample", name)
		}
		a.openRequest(request)
	}
}

// The strip of request tabs is pinned as text (element kinds, options,
// handlers, rects) at two widths in each of its looks, so moving it to
// markup cannot change it unseen.
func TestTheRequestTabsLayOutAsPinned(t *testing.T) {
	sizes := []struct {
		name  string
		width int
	}{
		{"wide", requestTabsWideW},
		{"narrow", requestTabsNarrowW},
	}
	for _, state := range requestTabsStates {
		for _, size := range sizes {
			name := "requesttabs-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				a, _ := newTestApp(t)
				t.Cleanup(a.waitWrites)
				state.setUp(t, a)
				root := a.requestTabs()
				Layout(a.ctx, root, size.width, tabsH)
				want, err := os.ReadFile(filepath.Join("testdata", name+".dump"))
				if err != nil {
					t.Fatal(err)
				}
				if got := Dump(root); got != string(want) {
					t.Fatalf("got:\n%s\nwant:\n%s", got, want)
				}
			})
		}
	}
}

// newRequestX is the middle of the "+" at the right end of the strip.
const newRequestX = winW - 20

// The "+" and each tab's cross say what they do, with their shortcuts.
func TestTheRequestTabsButtonsHaveTooltips(t *testing.T) {
	a := twoTabs(t)
	t.Cleanup(a.waitWrites)
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

// Clicking the "+" opens a new request in a new, active tab.
func TestClickingThePlusOpensANewRequestTab(t *testing.T) {
	a, _ := newTestApp(t)
	t.Cleanup(a.waitWrites)
	before := len(a.list.Tabs)
	click(a, newRequestX, tabStripY)
	if len(a.list.Tabs) != before+1 || a.list.Active != before {
		t.Fatalf("%d tabs, active %d; want %d tabs with the new one active", len(a.list.Tabs), a.list.Active, before+1)
	}
}

// Right-clicking a tab opens its menu where the pointer is.
func TestRightClickingATabOpensItsMenuAtThePointer(t *testing.T) {
	a := twoTabs(t)
	t.Cleanup(a.waitWrites)
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
