package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

var requestTabsTemplate = markup.Must("requesttabs.vml", requesttabsSource)

// tabSeparatorInset keeps the line between two tabs off the strip's
// edges, so the tabs read as one strip and not as boxes.
const tabSeparatorInset = 12

// requestTabItem is what the strip's template shows of one open request.
type requestTabItem struct {
	Badge  string  // the request's method or protocol
	Color  w.Token // the badge's color
	Name   string
	Dirty  bool
	Active bool
	Stripe w.Token // the color of the tab's collection
	// Separator is the line's inset before the tab; 0, no line (the first tab).
	Separator   int
	Press       func()
	MiddlePress func()
	DragOver    func()
	Menu        func(pointerX, pointerY int)
	Close       func()
}

// requestTabs is the strip of open requests, with a "+" for a new one.
func (a *App) requestTabs() Node {
	var items []requestTabItem
	for i, tab := range a.openTabs() {
		items = append(items, a.requestTab(i, tab))
	}
	return requestTabsTemplate.MustBuild(a.scope(markup.Scope{
		"looseGap":   w.LooseGap,
		"tabsH":      tabsH,
		"tabGap":     w.TabGap,
		"tabsScroll": &a.tabsScroll,
		"tabs":       items,
		"tabBadgeW":  badgeSlot(a.ctx, tabBadgeW),
		"tabDirtyW":  tabDirtyW,
		"dirtyMark":  dirtyMark,
		"closeTip":   a.hint("Close tab", "close-tab"),
		"newRequest": a.newRequest,
		"newTip":     a.hint("New request", "new-request"),
	}))
}

// requestTab is the view model of tab i: badge, name, dirty mark and handlers.
func (a *App) requestTab(index int, tab *reqTab) requestTabItem {
	item := requestTabItem{
		Badge:       requestBadge(tab.Req.Protocol, tab.Req.Method),
		Color:       theme.MethodToken(tab.Req.Method),
		Name:        tab.Req.Name,
		Dirty:       a.tabDirty(tab),
		Active:      index == a.list.Active,
		Stripe:      a.collectionToken(request.Node{Req: tab.Req}),
		Press:       func() { a.pressTab(index) },
		MiddlePress: func() { a.updateTabs(tabs.DragBegin{Index: index}) },
		DragOver:    func() { a.updateTabs(tabs.DragOver{Index: index}) },
		Menu:        func(pointerX, pointerY int) { a.openMenu(pointerX, pointerY, a.tabMenuItems(index)) },
		Close:       func() { a.closeTab(index) },
	}
	if index > 0 {
		item.Separator = tabSeparatorInset
	}
	return item
}

// pressTab activates tab i and starts dragging it, unless the strip
// overflows: a left drag then pans the strip, and the middle button reorders.
func (a *App) pressTab(index int) {
	a.updateTabs(tabs.Activate{Index: index})
	if !a.tabsScroll.Overflowing {
		a.updateTabs(tabs.DragBegin{Index: index})
	}
}

// Templates, parsed once above.
const (
	requesttabsSource = `<row h="{{tabsH}}" pad-t="{{tabGap}}" bg="frame" control="tabstrip" border-b="border">
  <hscroll state="tabsScroll" grow="1">
    <!-- inactive tabs blend into the strip; the active one joins the pane. A middle drag reorders, like a left one -->
    <row for="tab in tabs" active-tab="{{tab.Active}}" separator="{{tab.Separator}}" on-middle-click="tab.MiddlePress" on-right-click="tab.Menu">
      <row w="{{tabStripeW}}" align-self="stretch" bg="{{tab.Stripe}}"/>
      <row pad-l="{{tabLabelInset}}" gap="{{looseGap}}" align-items="baseline" on-click="tab.Press" on-drag-over="tab.DragOver">
        <text small color="{{tab.Color}}" w="{{tabBadgeW}}">{{tab.Badge}}</text>
        <text>{{tab.Name}}</text>
        <row w="{{tabDirtyW}}" align-self="stretch">
          <text if="tab.Dirty" color="accent-line" grow="1">{{dirtyMark}}</text>
        </row>
      </row>
      <row w="{{tabCloseW}}" on-click="tab.Close" tooltip="{{closeTip}}">
        <icon name="cross" grow="1"/>
      </row>
    </row>
  </hscroll>
  <row w="{{newTabW}}" on-click="newRequest" tooltip="{{newTip}}">
    <icon name="plus" grow="1"/>
  </row>
</row>
`
)
