package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"reflect"
	"strings"
	"testing"
)

func TestNewRequestAndSwitchingTabs(t *testing.T) {
	a := testApp(t)
	a.newRequest()
	if len(a.list.Tabs) != 2 || a.list.Active != 1 || a.cur().Req.Name != "New Request" {
		t.Fatalf("tabs %d active %d", len(a.list.Tabs), a.list.Active)
	}
	click(a, 280, headerH+tabsH/2) // the first tab sits at the left edge of the strip
	if a.list.Active != 0 {
		t.Fatalf("active = %d, want 0", a.list.Active)
	}
}

func TestCloseButtonClosesOnlyThatTab(t *testing.T) {
	a := twoTabs(t) // active: the second
	left, width := a.tabX(0)
	click(a, left+width-12, tabStripY) // the × of the first tab
	if len(a.list.Tabs) != 1 || a.cur().Req.Name != "Repository Info" {
		t.Fatalf("%d tabs, current %q", len(a.list.Tabs), a.cur().Req.Name)
	}
}

func TestClickingTheLabelSelectsAndDoesNotClose(t *testing.T) {
	a := twoTabs(t)
	left, _ := a.tabX(0)
	click(a, left+30, tabStripY)
	if len(a.list.Tabs) != 2 || a.list.Active != 0 {
		t.Fatalf("%d tabs, active %d", len(a.list.Tabs), a.list.Active)
	}
}

func TestMiddleButtonDragReordersTabsAndDoesNotCloseThem(t *testing.T) {
	a := twoTabs(t)
	first := a.openTabs()[0]
	l0, _ := a.tabX(0)
	l1, w1 := a.tabX(1)
	a.step(core.Input{X: l0 + 20, Y: tabStripY, MiddleClick: true, MiddleHeld: true}) // press the first tab
	if len(a.list.Tabs) != 2 {
		t.Fatal("a middle press must not close the tab any more")
	}
	a.step(core.Input{X: l1 + w1/2, Y: tabStripY, MiddleHeld: true}) // drag over the second
	if a.openTabs()[1] != first {
		t.Fatal("the dragged tab must move to the second place")
	}
	a.step(core.Input{X: l1 + w1/2, Y: tabStripY}) // release
	if a.list.Drag != -1 {
		t.Fatalf("dragTab = %d after the release", a.list.Drag)
	}
	a.step(core.Input{X: l0 + 20, Y: tabStripY, MiddleHeld: true}) // held again, but not pressed on a tab
	if a.openTabs()[1] != first {
		t.Fatal("after the release, moving must not drag")
	}
}

func TestMiddleClickWithoutMovingLeavesTheTabsAlone(t *testing.T) {
	a := twoTabs(t)
	left, width := a.tabX(1)
	a.step(core.Input{X: left + width/2, Y: tabStripY, MiddleClick: true, MiddleHeld: true})
	a.step(core.Input{X: left + width/2, Y: tabStripY})
	if len(a.list.Tabs) != 2 || a.cur().Req.Name != "Repository Info" {
		t.Fatalf("%d tabs, current %q", len(a.list.Tabs), a.cur().Req.Name)
	}
}

func TestClosingATabBeforeTheActiveOneKeepsTheSameTabActive(t *testing.T) {
	a := twoTabs(t)
	a.closeTab(0)
	if a.list.Active != 0 || a.cur().Req.Name != "Repository Info" {
		t.Fatalf("active %d current %q", a.list.Active, a.cur().Req.Name)
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
	a.step(core.Input{Edits: []core.Edit{core.EditBackspace}})
	if dirty(tab) {
		t.Fatal("restoring the original text makes it clean again")
	}
	click(a, 739, paramsRow0Y) // delete a param
	if !dirty(tab) {
		t.Fatal("deleting a param is a change")
	}
}

func TestTabWidthDoesNotChangeWhenTheRequestChanges(t *testing.T) {
	a := twoTabs(t)
	left, width := a.tabX(0)
	req := a.openTabs()[0].Req
	req.Method = "DELETE"                                                 // a longer badge
	req.Headers = append(req.Headers, model.KV{Name: "X", Enabled: true}) // and an unsaved dot
	a.step(core.Input{})
	if !dirty(a.openTabs()[0]) {
		t.Fatal("test setup: the tab must be dirty")
	}

	// The × sits where it did before the edits; if the tab had grown it
	// would be somewhere else and the click would only select the tab.
	click(a, left+width-12, tabStripY)
	if len(a.list.Tabs) != 1 || a.cur().Req.Name != "Repository Info" {
		t.Fatalf("%d tabs: the tab changed size when its request changed", len(a.list.Tabs))
	}
}

func TestTheTabSlotsFitTheirContentAtEveryTextSize(t *testing.T) {
	for size := settings.MinFontSize; size <= settings.MaxFontSize; size++ {
		ctx, err := core.NewCtx(appearance.ThemeFor("dark", appearance.FontSizes{UI: size, Small: max(size-2, 9), Code: size - 1}))
		if err != nil {
			t.Fatal(err)
		}
		for _, badge := range badgeLabels {
			if got := ctx.Width(ctx.F.Small, badge); got-badgeOverhang > badgeSlot(ctx, tabBadgeW)+badgeOverhang {
				t.Errorf("%d: %q is %dpx wide, the tab badge slot is %dpx", size, badge, got, badgeSlot(ctx, tabBadgeW))
			}
			if got := ctx.Width(ctx.F.Small, badge); got-badgeOverhang > badgeSlot(ctx, badgeW)+badgeOverhang {
				t.Errorf("%d: %q is %dpx wide, the tree badge slot is %dpx", size, badge, got, badgeSlot(ctx, badgeW))
			}
		}
		if size == settings.DefaultFontSize && (badgeSlot(ctx, tabBadgeW) != tabBadgeW || badgeSlot(ctx, badgeW) != badgeW) {
			t.Errorf("at the default size the slots must stay %d and %d", tabBadgeW, badgeW)
		}
		if got := ctx.Width(ctx.F.UI, dirtyMark); got > tabDirtyW {
			t.Errorf("%d: the dirty mark is %dpx wide, the slot is %dpx", size, got, tabDirtyW)
		}
	}
}

func TestDraggingATabReordersIt(t *testing.T) {
	a := twoTabs(t)
	first := a.openTabs()[0]
	l0, w0 := a.tabX(0)
	l1, w1 := a.tabX(1)

	a.step(core.Input{X: l0 + 20, Y: tabStripY, Click: true, Held: true}) // press the first tab
	a.step(core.Input{X: l1 + w1/2, Y: tabStripY, Held: true})            // drag over the second
	if a.openTabs()[1] != first {
		t.Fatalf("the dragged tab must move to the second place")
	}
	a.step(core.Input{X: l1 + w1/2, Y: tabStripY}) // release
	a.step(core.Input{X: l0 + w0/2, Y: tabStripY, Held: true})
	if a.openTabs()[1] != first {
		t.Fatal("after release, moving over another tab must not drag")
	}
	if a.list.Drag != -1 {
		t.Fatalf("dragTab = %d after release", a.list.Drag)
	}
}

func TestDraggingKeepsTheActiveTabActive(t *testing.T) {
	a := twoTabs(t) // second tab active
	active := a.cur()
	l0, _ := a.tabX(0)
	_, w1 := a.tabX(1)
	l1, _ := a.tabX(1)
	a.step(core.Input{X: l0 + 20, Y: tabStripY, Click: true, Held: true})
	a.step(core.Input{X: l1 + w1/2, Y: tabStripY, Held: true})
	if a.cur() == active {
		t.Fatal("pressing the first tab activates it, so it is the active one")
	}
	if a.cur() != a.openTabs()[1] {
		t.Fatal("the active tab must travel with the drag")
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
	if len(a.list.Tabs) != 1 || a.cur().Req.Name != "Search Repos" {
		t.Fatalf("%d tabs, current %q", len(a.list.Tabs), a.cur().Req.Name)
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
	if len(a.list.Tabs) != 1 || a.openTabs()[0] != keep || a.cur() != keep {
		t.Fatalf("%d tabs, current %q", len(a.list.Tabs), a.cur().Req.Name)
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
	if a.cur().Req.Name != "Search Repos copy" || len(a.list.Tabs) != 2 {
		t.Fatalf("current %q, %d tabs", a.cur().Req.Name, len(a.list.Tabs))
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

func TestManyTabsScrollTheStripAndKeepTheActiveTabInView(t *testing.T) {
	a := testApp(t)
	for range 14 {
		a.newRequest()
	}
	a.step(core.Input{})
	if a.tabsScroll.X <= 0 {
		t.Fatalf("the new, active tab is past the right edge: X = %d", a.tabsScroll.X)
	}
	x := a.tabsScroll.X
	a.step(core.Input{X: 600, Y: tabStripY, WheelX: 1}) // two-finger swipe right: back toward the first tabs
	if a.tabsScroll.X >= x {
		t.Fatalf("a sideways swipe must scroll the strip back: %d -> %d", x, a.tabsScroll.X)
	}
	x = a.tabsScroll.X
	a.step(core.Input{X: 600, Y: tabStripY, Wheel: -1}) // the mouse wheel scrolls it too
	if a.tabsScroll.X <= x {
		t.Fatalf("the wheel must scroll the strip forward: %d -> %d", x, a.tabsScroll.X)
	}
	x = a.tabsScroll.X
	click(a, sidebarW+4, tabStripY) // the left arrow, at the strip's left edge
	if a.tabsScroll.X >= x {
		t.Fatalf("clicking the left arrow must scroll back: %d -> %d", x, a.tabsScroll.X)
	}
	if len(a.list.Tabs) != 15 || a.list.Active != 14 {
		t.Fatalf("an arrow click must not select or close a tab: %d tabs, active %d", len(a.list.Tabs), a.list.Active)
	}
}

func TestLeftDragOnAnOverflowingTabStripPansItInsteadOfReorderingTabs(t *testing.T) {
	a := testApp(t)
	for range 14 {
		a.newRequest()
	}
	a.step(core.Input{})
	order := append([]*reqTab(nil), a.openTabs()...)
	x0 := a.tabsScroll.X // the active (last) tab was revealed: the strip is at its right end
	a.step(core.Input{X: 600, Y: tabStripY, Click: true, Held: true})
	a.step(core.Input{X: 700, Y: tabStripY, Held: true}) // drag right: earlier tabs come into view
	a.step(core.Input{X: 800, Y: tabStripY, Held: true})
	if want := x0 - 200; a.tabsScroll.X != want {
		t.Fatalf("X = %d, want %d: the strip follows the pointer", a.tabsScroll.X, want)
	}
	for i := range order {
		if a.openTabs()[i] != order[i] {
			t.Fatalf("tab %d moved: a left drag on an overflowing strip must not reorder", i)
		}
	}
	a.step(core.Input{X: 800, Y: tabStripY}) // release
	if a.list.Drag != -1 {
		t.Fatalf("dragTab = %d", a.list.Drag)
	}
}
