package app

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/format"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"

	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Sizes of the window chrome, in logical pixels.
const (
	sidebarW = 260           // the collections tree's default width
	headerH  = sizes.HeaderH // the banner: title, environment chip, settings
	tabsH    = 42            // the request tab row: a tab's height plus the room above it
	urlH     = 56            // the URL bar: its controls plus padding above and below

	// Smallest sizes the dividers allow.
	minSidebarW = 160
	minMainW    = 420
	minPaneSize = 220

	// A request tab has fixed slots for the method badge and the unsaved
	// dot, so editing a request never changes the tab's width.
	tabBadgeW = 48
	tabDirtyW = 12
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
	return windowTemplate.MustBuild(markup.Scope{
		"sidebarSplit":   &a.sidebarSplit,
		"sidebarDefault": sidebarW - 1,
		"minSidebarW":    minSidebarW,
		"minMainW":       minMainW,
		"sidebar":        a.side.View(a.ctx),
		"header":         a.header(),
		"requestTabs":    a.requestTabs(),
		"urlBar":         a.urlBar(t),
		"paneSplit":      &a.paneSplit,
		"minPaneSize":    minPaneSize,
		"stacked":        a.settings.Layout == "stacked",
		"requestPane":    a.pane(t, requestPanesFor(t), &t.paneTab, &t.reqScroll, &t.reqTabsX, nil),
		"responsePane":   a.pane(t, responsePanesFor(t), &t.respTab, &t.respScroll, &t.respTabsX, a.responseStatus(t)),
		"statusBar":      a.statusBar(),
		"menu":           a.menuView(),
		"picker":         a.pickerView(),
		"browser":        a.fileBrowserView(),
		"popupMenu":      PopupMenuLayer(a.ctx),
		"dialog":         a.dialogView(),
		"callLog":        a.calls.View(a.ctx, a),
		"toasts":         a.toastsView(),
		"tooltips":       TooltipLayer(&a.tips),
	})
}

// ── Main column ────────────────────────────────────────────────────────

var headerTemplate = markup.Must("header.vml", headerSource)

// header is the banner over the main column: the collection's name, the
// environment chip and the settings menu button.
func (a *App) header() Node {
	return headerTemplate.MustBuild(a.scope(markup.Scope{
		"headerH":     headerH,
		"gutter":      w.Gutter,
		"title":       a.headerTitle(),
		"env":         a.envChip(),
		"toggleMenu":  a.toggleSettings,
		"settingsTip": a.hint("Settings", "settings"),
	}))
}

var paneTemplate = markup.Must("pane.vml", paneSource)

// pane is one tabbed area (pane.vml): a tab strip, with extra at its
// right end when there is one, above the body of the active tab.
func (a *App) pane(tab *reqTab, panes []pane, active *int, scroll, tabsX *ScrollState, extra Node) Node {
	selectTab := func(index int) { *active, scroll.Y = index, 0 }
	*active = min(*active, len(panes)-1) // a socket has fewer response panes
	return paneTemplate.MustBuild(markup.Scope{
		"tabsX":     tabsX,
		"tabs":      a.tabItems(panes, tab),
		"active":    *active,
		"selectTab": selectTab,
		"extra":     extra,
		"body":      a.paneBody(panes[*active], tab, scroll),
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

	headerSource = `<!-- In a window too narrow for everything the title gives up room first,
     down to 48 px, then the chip, so the settings button stays inside. -->
<row h="{{headerH}}" pad-x="{{gutter}}" bg="hero" control="hero">
  <text bold ellipsis shrink min-w="{{titleMinW}}" color="hero-text">{{title}}</text>
  <spacer/>
  <slot name="env"/>
  <spacer w="{{gutter}}"/>
  <row w="{{headerH}}" on-click="toggleMenu" tooltip="{{settingsTip}}">
    <icon name="gear" grow="1" color="hero-muted" hover-color="hero-text"/>
  </row>
</row>
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

var responseStatusTemplate = markup.Must("responsestatus.vml", responsestatusSource)

// scope is the scope of a template: the sizes of the faces the context draws
// and the fixed widths, then own.
func (a *App) scope(own markup.Scope) markup.Scope { return sizes.Scope(a.ctx, own) }

// statusBar is the line under the window.
func (a *App) statusBar() Node { return a.bar.View(a.ctx) }

// responseStatus is "200 OK  1.4s  156.67KB", dropped when the tabs need the room.
func (a *App) responseStatus(tab *reqTab) Node {
	if tab.Resp == nil || tab.Resp.Err != "" {
		return nil
	}
	fg, bg := statusPill(tab.Resp.Status)
	return responseStatusTemplate.MustBuild(a.scope(markup.Scope{
		"pillFg": fg,
		"pillBg": bg,
		"status": fmt.Sprintf("%d %s", tab.Resp.Status, tab.Resp.StatusText),
		"timing": fmt.Sprintf("%s  %s", format.Duration(tab.Resp.Duration), format.Size(tab.Resp.Size)),
	}))
}

// statusPill is the text and background colors of a response status:
// success, redirect or failure, as the dashboard's badges.
func statusPill(status int) (fg, bg w.Token) {
	switch {
	case status >= 400:
		return w.Error, w.ErrorSoft
	case status >= 300:
		return w.Warn, w.WarnSoft
	}
	return w.OK, w.OKSoft
}

const responsestatusSource = `<row gap="{{gutter}}" pad-r="{{gutter}}" optional align-items="center">
  <row pad-x="{{labelPad}}" h="{{slimH}}" bg="{{pillBg}}" control="pill" align-items="center">
    <text small bold color="{{pillFg}}">{{status}}</text>
  </row>
  <text>{{timing}}</text>
</row>
`
