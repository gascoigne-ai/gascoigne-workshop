package tabstrip

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

var template = markup.Must("requesttabs.vml", source)

// tabSeparatorInset keeps the line between two tabs off the strip's
// edges, so the tabs read as one strip and not as boxes.
const tabSeparatorInset = 12

// tabItem is what the strip's template shows of one open request.
type tabItem struct {
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

// View is the strip of open requests, with a "+" for a new one.
func (c *Component) View(ctx *w.Ctx) Node {
	var items []tabItem
	for i, tab := range c.deps.Tabs.Tabs() {
		items = append(items, c.item(i, tab))
	}
	return template.MustBuild(sizes.Scope(ctx, markup.Scope{
		"looseGap":   w.LooseGap,
		"tabsH":      sizes.TabsH,
		"tabGap":     w.TabGap,
		"tabsScroll": &c.scroll,
		"tabs":       items,
		"tabBadgeW":  sizes.BadgeSlot(ctx, sizes.TabBadgeW),
		"tabDirtyW":  sizes.TabDirtyW,
		"dirtyMark":  DirtyMark,
		"closeTip":   c.deps.Hints.Hint("Close tab", "close-tab"),
		"newRequest": c.deps.Requests.NewRequest,
		"newTip":     c.deps.Hints.Hint("New request", "new-request"),
	}))
}

// item is the view model of tab index: badge, name, dirty mark and handlers.
func (c *Component) item(index int, tab *tabs.Tab) tabItem {
	item := tabItem{
		Badge:       theme.RequestBadge(tab.Req.Protocol, tab.Req.Method),
		Color:       theme.MethodToken(tab.Req.Method),
		Name:        tab.Req.Name,
		Dirty:       c.deps.Marks.Dirty(tab),
		Active:      index == c.deps.Tabs.Active(),
		Stripe:      c.deps.Marks.Stripe(tab.Req),
		Press:       func() { c.press(index) },
		MiddlePress: func() { c.deps.Tabs.Update(tabs.DragBegin{Index: index}) },
		DragOver:    func() { c.deps.Tabs.Update(tabs.DragOver{Index: index}) },
		Menu:        func(pointerX, pointerY int) { c.deps.Menus.OpenTabMenu(pointerX, pointerY, index) },
		Close:       func() { c.deps.Tabs.Update(tabs.Close{Index: index}) },
	}
	if index > 0 {
		item.Separator = tabSeparatorInset
	}
	return item
}

// press activates tab index and starts dragging it, unless the strip
// overflows: a left drag then pans the strip, and the middle button reorders.
func (c *Component) press(index int) {
	c.deps.Tabs.Update(tabs.Activate{Index: index})
	if !c.scroll.Overflowing {
		c.deps.Tabs.Update(tabs.DragBegin{Index: index})
	}
}

// Template, parsed once above.
const (
	source = `<row h="{{tabsH}}" pad-t="{{tabGap}}" bg="frame" control="tabstrip" border-b="border">
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
