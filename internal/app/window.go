package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Sizes of the window chrome, in logical pixels.
const (
	sidebarW = 260           // the collections tree's default width
	headerH  = sizes.HeaderH // the banner: title, environment chip, settings
	tabsH    = sizes.TabsH   // the request tab row: a tab's height plus the room above it
	urlH     = sizes.URLBarH // the URL bar: its controls plus padding above and below

	// Smallest sizes the dividers allow.
	minSidebarW = 160
	minMainW    = 420
	minPaneSize = 220
)

// sidebarWidth is the current width of the sidebar in pixels.
func (a *App) sidebarWidth() int {
	if a.sidebarSplit.Size > 0 {
		return int(a.sidebarSplit.Size)
	}
	return sidebarW - 1
}

var windowTemplate = markup.Must("window.vml", windowSource)

// view is the whole window as a function of the app state (window.vml):
// the collections tree beside the main column (header, request tabs, URL
// bar, request | response) above the status bar, and the overlays over
// them. The popup menu and the tooltip are layers that markup has no
// element for, so they come in as slots like the views built here.
func (a *App) view() Node {
	t := a.cur()
	scope := markup.Scope{
		"sidebarSplit":   &a.sidebarSplit,
		"sidebarDefault": sidebarW - 1,
		"minSidebarW":    minSidebarW,
		"minMainW":       minMainW,
		"sidebar":        a.side.View(a.ctx),
		"header":         a.header.View(a.ctx),
		"requestTabs":    a.strip.View(a.ctx),
		"urlBar":         a.editor.URLBar.View(a.ctx),
		"paneSplit":      &a.paneSplit,
		"minPaneSize":    minPaneSize,
		"stacked":        a.settings.Layout == "stacked",
		"requestPane":    a.editor.RequestPane(a.ctx, t.Tab),
		"responsePane":   a.responsePane(t, a.responsePanesFor(t)),
		"statusBar":      a.statusBar(),
	}
	// The overlays are built after the panes, as before, one node per slot of
	// the template; the tooltip layer is last so it is on top.
	over := a.over.Views(a.ctx)
	scope["menu"], scope["picker"], scope["browser"], scope["popupMenu"] = over.Settings, over.Picker, over.Browser, over.Popup
	scope["dialog"], scope["callLog"], scope["toasts"] = over.Dialog, over.CallLog, over.Toasts
	scope["tooltips"] = TooltipLayer(&a.tips)
	return windowTemplate.MustBuild(scope)
}

// ── Main column ────────────────────────────────────────────────────────

var paneTemplate = markup.Must("pane.vml", paneSource)

// responsePane is the response area (pane.vml): a tab strip, with the
// status at its right end, above the body of the active tab. The request
// pane is the editor's twin of it.
func (a *App) responsePane(tab *reqTab, panes []pane) Node {
	selectTab := func(index int) { tab.respTab, tab.respScroll.Y = index, 0 }
	tab.respTab = min(tab.respTab, len(panes)-1) // a socket has fewer response panes
	return paneTemplate.MustBuild(markup.Scope{
		"tabsX":     &tab.respTabsX,
		"tabs":      a.tabItems(panes, tab),
		"active":    tab.respTab,
		"selectTab": selectTab,
		"extra":     a.resp.StatusView(a.ctx, tab.Tab),
		"body":      a.paneBody(panes[tab.respTab], tab, &tab.respScroll),
	})
}

// Templates, parsed once above.
const (
	windowSource = `<stack>
  <!-- the frame: collections tree | main column, above the status bar -->
  <col bg="frame">
    <split state="sidebarSplit" default="{{sidebarDefault}}" grow="1" min-first="{{minSidebarW}}" min-second="{{minMainW}}">
      <slot name="sidebar"/>
      <!-- the main column: header, request tabs, URL bar, then request | response -->
      <col grow="1">
        <slot name="header"/>
        <slot name="requestTabs"/>
        <slot name="urlBar"/>
        <split state="paneSplit" default="0.5" as-ratio grow="1" min-first="{{minPaneSize}}" min-second="{{minPaneSize}}" vertical="{{stacked}}">
          <slot name="requestPane"/>
          <slot name="responsePane"/>
        </split>
      </col>
    </split>
    <slot name="statusBar"/>
  </col>
  <!-- the overlays, each only while open, in z-order: the tooltip last so it is on top -->
  <slot if="menu" name="menu"/>
  <slot if="picker" name="picker"/>
  <slot if="browser" name="browser"/>
  <slot if="popupMenu" name="popupMenu"/>
  <slot if="dialog" name="dialog"/>
  <slot if="callLog" name="callLog"/>
  <slot if="toasts" name="toasts"/>
  <slot name="tooltips"/>
</stack>
`

	paneSource = `<col grow="1" bg="bg">
  <tab-bar>
    <scroll-tabs state="tabsX" items="tabs" active="{{active}}" on-select="selectTab"/>
    <slot if="extra" name="extra"/>
  </tab-bar>
  <slot name="body"/>
</col>
`
)

// scope is the scope of a template: the sizes of the faces the context draws
// and the fixed widths, then own.
func (a *App) scope(own markup.Scope) markup.Scope { return sizes.Scope(a.ctx, own) }

// statusBar is the line under the window.
func (a *App) statusBar() Node { return a.bar.View(a.ctx) }
