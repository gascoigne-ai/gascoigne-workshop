package app

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/collection"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/model/textfmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Sizes of the window chrome, in logical pixels.
const (
	sidebarW = 260
	headerH  = 44
	tabsH    = 42
	urlH     = 56
	statusH  = 28

	// Smallest sizes the dividers allow.
	minSidebarW = 160
	minMainW    = 420
	minPaneSize = 220
	treeIndent  = 16
	badgeW      = 40

	// A request tab has fixed slots for the method badge and the unsaved
	// dot, so editing a request never changes the tab's width.
	tabBadgeW = 48
	tabDirtyW = 12
)

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
		"sidebar":        a.sidebar(),
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
		"browser":        a.fileBrowserView(),
		"popupMenu":      PopupMenuLayer(a.ctx),
		"dialog":         a.dialogView(),
		"callLog":        a.calls.View(a),
		"toasts":         a.toastsView(),
		"tooltips":       TooltipLayer(&a.tips),
	})
}

// ── Sidebar ────────────────────────────────────────────────────────────

var (
	sidebarTemplate     = markup.Must("sidebar.vml", sidebarSource)
	treeFolderTemplate  = markup.Must("treefolder.vml", treefolderSource)
	treeRequestTemplate = markup.Must("treerequest.vml", treerequestSource)
)

// sidebar is the header (the chip that switches between the collection
// tree and the log of sent requests, and the filter) above the one shown.
func (a *App) sidebar() Node {
	label, switchTo := "Collections", func() { a.showSent = true }
	if a.showSent {
		label, switchTo = "Sent", func() { a.showSent = false }
	}
	var sentList Node
	var rows []Node
	if a.showSent {
		sentList = a.sent.List(a, a.filter.Text)
	} else {
		for _, row := range a.visibleRows() {
			rows = append(rows, a.treeRow(row, row.Node() == a.selected))
		}
	}
	return sidebarTemplate.MustBuild(markup.Scope{
		"mode":           label,
		"switchMode":     switchTo,
		"collectionMenu": func(posX, posY int) { a.openMenu(posX, posY, a.collectionMenu()) },
		"filter":         &a.filter,
		"showSent":       a.showSent,
		"showTree":       !a.showSent && len(a.open.Entries) > 0,
		"noCollection":   !a.showSent && len(a.open.Entries) == 0,
		"openFolder":     a.promptWorkspace,
		"newCollection":  a.promptNewCollection,
		"sentList":       sentList,
		"treeScroll":     &a.treeScroll,
		"rows":           rows,
		"headerH":        headerH,
		"gutter":         w.Gutter,
	})
}

// treeItem is what a tree row's template shows of one row.
type treeItem struct {
	Name     string // the whole name, shown when the filter matched none of it
	Before   string // the name cut around the part the filter matched
	Match    string
	After    string
	Indent   int
	Selected bool
	// Root marks a collection's own row: its name is bold, so where one
	// collection ends and the next begins reads at a glance.
	Root     bool
	Triangle string  // a folder's disclosure icon
	Badge    string  // a request's method or protocol
	Color    w.Token // the badge's color
	// DropBorder outlines the row a drag would drop on; empty, no outline.
	DropBorder w.Token
	Stripe     w.Token // a collection root's color
	Press      func()
	Menu       func(pointerX, pointerY int)
	DragOver   func()
}

// treeRow is one row of the tree, a folder or a request: one template
// each, since their rows differ in padding and alignment.
func (a *App) treeRow(row collection.Row, selected bool) Node {
	item := a.treeItemOf(row)
	item.Selected = selected
	template := treeFolderTemplate
	if !row.Folder {
		template = treeRequestTemplate
	}
	return template.MustBuild(markup.Scope{"row": item, "rowH": w.RowH, "gutter": w.Gutter, "badgeW": badgeW})
}

// treeItemOf is the view model of one tree row, except whether it is selected.
func (a *App) treeItemOf(row collection.Row) treeItem {
	node := row.Node()
	item := treeItem{
		Name:     row.Name,
		Indent:   w.Gutter + row.Depth*treeIndent,
		Press:    func() { a.pressTreeRow(row) },
		Menu:     func(pointerX, pointerY int) { a.openMenu(pointerX, pointerY, a.treeMenuItems(row)) },
		DragOver: func() { a.dropTarget = node },
	}
	item.Before, item.Match, item.After = collection.SplitMatch(row.Name, a.filter.Text)
	if row.Folder {
		item.Root = a.isRoot(row.Dir)
		if item.Root {
			item.Stripe = a.collectionToken(row.Dir)
		}
		item.Triangle = "triangle-right"
		if row.Open {
			item.Triangle = "triangle-down"
		}
	} else {
		item.Indent += 6
		item.Badge = requestBadge(row.Req.Protocol, row.Req.Method)
		item.Color = appearance.MethodToken(row.Req.Method)
	}
	if a.dragNode != nil && a.dropTarget == node && node != a.dragNode {
		item.DropBorder = w.AccentLine
	}
	return item
}

// pressTreeRow is a press on a row: it selects it, gives the tree the
// keyboard and starts a drag. A request opens at once; a folder toggles
// when the button is released on the same row (see finishDrag), so an
// open folder can still be dragged.
func (a *App) pressTreeRow(row collection.Row) {
	a.treeFocus = true
	a.dragNode = row.Node()
	if a.isRoot(row.Dir) {
		a.dragNode = nil // the collection itself cannot be moved
	}
	if row.Folder {
		a.selectNode(row.Node())
		a.pressedFolder = row.Dir
		return
	}
	a.clickTree(row)
}

// requestBadge is the short label shown before a request name.
func requestBadge(protocol model.Protocol, method string) string {
	if info := protocol.Info(); info.Label != "" && info.Method != model.MethodHTTP {
		return info.Label
	}
	if method == "DELETE" {
		return "DEL"
	}
	return method
}

// ── Main column ────────────────────────────────────────────────────────

var headerTemplate = markup.Must("header.vml", headerSource)

// header is the banner over the main column: the collection's name, the
// environment chip and the settings menu button.
func (a *App) header() Node {
	return headerTemplate.MustBuild(markup.Scope{
		"headerH":     headerH,
		"gutter":      w.Gutter,
		"title":       a.headerTitle(),
		"env":         a.envChip(),
		"toggleMenu":  a.toggleSettings,
		"settingsTip": a.hint("Settings", "settings"),
	})
}

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
	return requestTabsTemplate.MustBuild(markup.Scope{
		"tabsH":      tabsH,
		"tabGap":     w.TabGap,
		"tabsScroll": &a.tabsScroll,
		"tabs":       items,
		"tabBadgeW":  tabBadgeW,
		"tabDirtyW":  tabDirtyW,
		"dirtyMark":  dirtyMark,
		"closeTip":   a.hint("Close tab", "close-tab"),
		"newRequest": a.newRequest,
		"newTip":     a.hint("New request", "new-request"),
	})
}

// requestTab is the view model of tab i: badge, name, dirty mark and handlers.
func (a *App) requestTab(index int, tab *reqTab) requestTabItem {
	item := requestTabItem{
		Badge:       requestBadge(tab.Req.Protocol, tab.Req.Method),
		Color:       appearance.MethodToken(tab.Req.Method),
		Name:        tab.Req.Name,
		Dirty:       a.tabDirty(tab),
		Active:      index == a.list.Active,
		Stripe:      a.collectionToken(tab.Req),
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

var urlBarTemplate = markup.Must("urlbar.vml", urlbarSource)

// urlBar is [protocol | method | URL field ........] [Send]. The button
// is one of three: Send, Connect or Disconnect on a socket, and Cancel
// while a request runs.
func (a *App) urlBar(tab *reqTab) Node {
	info, socket := tab.Req.Protocol.Info(), isSocket(tab)
	httpMethod, rpcMethod := info.Method == model.MethodHTTP, info.Method == model.MethodRPC
	sending := !socket && tab.Sending
	connectionLabel, connectionHint := "Connect", "Open the connection"
	if tab.Active() {
		connectionLabel, connectionHint = "Disconnect", "Close the connection"
	}
	grpcMethod, grpcMethodColor := grpcMethodLabel(tab)
	return urlBarTemplate.MustBuild(markup.Scope{
		"urlH":             urlH,
		"gutter":           w.Gutter,
		"urlTextGap":       w.Gutter - 4,
		"urlTip":           a.urlTip(tab),
		"protocol":         info.Label,
		"divided":          httpMethod || rpcMethod,
		"httpMethod":       httpMethod,
		"method":           tab.Req.Method,
		"methodColor":      appearance.MethodToken(tab.Req.Method),
		"rpcMethod":        rpcMethod,
		"streamable":       info.Connection == model.OnStream,
		"grpcMethodW":      grpcMethodW,
		"grpcMethod":       grpcMethod,
		"grpcMethodColor":  grpcMethodColor,
		"streamColor":      streamColor(tab),
		"streamBg":         streamBg(tab),
		"url":              &tab.url,
		"segments":         a.varSegments,
		"idle":             !socket && !sending,
		"socket":           socket,
		"sending":          sending,
		"sendTip":          a.hint("Send the request", "send"),
		"connectionLabel":  connectionLabel,
		"connectionTip":    a.hint(connectionHint, "send"),
		"cycleProtocol":    func() { a.cycleProtocol(tab) },
		"cycleMethod":      func() { a.cycleMethod(tab) },
		"pickMethod":       func() { a.pickGRPCMethod(tab) },
		"toggleStream":     func() { a.toggleStream(tab) },
		"edited":           func() { a.urlEdited(tab) },
		"send":             func() { a.send(tab) },
		"toggleConnection": func() { a.toggleConnection(tab) },
		"cancel":           func() { a.cancelSend(tab) },
	})
}

// streamColor shows the Stream chip lit while the tab streams.
func streamColor(tab *reqTab) w.Token {
	if tab.grpcStream {
		return w.Accent
	}
	return w.Muted
}

// streamBg tints the Stream chip while the tab streams; an empty token
// applies no background (decision 4).
func streamBg(tab *reqTab) w.Token {
	if tab.grpcStream {
		return w.Selected
	}
	return ""
}

// hint is a tooltip with the action's shortcut, when it has one.
func (a *App) hint(text, shortcutID string) string {
	if key := a.shortcutHint(shortcutID); key != "" {
		return text + " (" + key + ")"
	}
	return text
}

// varSegments colors {{variables}} so they stand out from the literal
// URL: defined ones in the success color, unresolved ones as errors.
func (a *App) varSegments(string string) []Span {
	var segs []Span
	vars := a.varsFor(a.cur().Req)
	for _, seg := range splitVars(string) {
		style := w.Text
		if seg.isVar {
			style = w.OK
			if !model.Defined(strings.TrimSuffix(strings.TrimPrefix(seg.text, "{{"), "}}"), vars) {
				style = w.Error
			}
		}
		segs = append(segs, Span{Text: seg.text, Style: style})
	}
	return segs
}

type urlSeg struct {
	text  string
	isVar bool
}

// splitVars cuts s into literal runs and {{name}} variable runs.
func splitVars(string string) []urlSeg {
	var segs []urlSeg
	for string != "" {
		start := strings.Index(string, "{{")
		end := strings.Index(string, "}}")
		if start < 0 || end < start {
			return append(segs, urlSeg{text: string})
		}
		if start > 0 {
			segs = append(segs, urlSeg{text: string[:start]})
		}
		segs = append(segs, urlSeg{text: string[start : end+2], isVar: true})
		string = string[end+2:]
	}
	return segs
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

var responseStatusTemplate = markup.Must("responsestatus.vml", responsestatusSource)

// responseStatus is "200 OK  1.4s  156.67KB", dropped when the tabs need the room.
func (a *App) responseStatus(tab *reqTab) Node {
	if tab.Resp == nil || tab.Resp.Err != "" {
		return nil
	}
	fg, bg := statusPill(tab.Resp.Status)
	return responseStatusTemplate.MustBuild(markup.Scope{
		"gutter": w.Gutter,
		"pillFg": fg,
		"pillBg": bg,
		"status": fmt.Sprintf("%d %s", tab.Resp.Status, tab.Resp.StatusText),
		"timing": fmt.Sprintf("%s  %s", textfmt.Duration(tab.Resp.Duration), textfmt.Size(tab.Resp.Size)),
	})
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

// ── Status bar ─────────────────────────────────────────────────────────

var statusBarTemplate = markup.Must("statusbar.vml", statusbarSource)

// statusBar is the line under the window: the last status, the Log
// button and the version.
func (a *App) statusBar() Node {
	return statusBarTemplate.MustBuild(markup.Scope{
		"statusH":     statusH,
		"gutter":      w.Gutter,
		"status":      a.status,
		"version":     version,
		"openLog":     a.calls.Open,
		"openWelcome": a.openWelcome,
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
  <slot if="browser" name="browser"/>
  <slot if="popupMenu" name="popupMenu"/>
  <slot if="dialog" name="dialog"/>
  <slot if="callLog" name="callLog"/>
  <slot if="toasts" name="toasts"/>
  <slot name="tooltips"/>
</stack>
`

	sidebarSource = `<col>
  <row h="{{headerH}}" pad-x="{{gutter}}" gap="6" bg="bg" border-b="border">
    <button w="96" h="28" on-click="switchMode" on-right-click="collectionMenu" tooltip="Collections and sent requests. Right-click: new, open, rename, save, close, delete a collection">{{mode}}</button>
    <field state="filter" h="28" grow="1" bg="sunken" placeholder="Filter" text-menu="filter"/>
  </row>
  <slot if="sentList" name="sentList"/>
  <scroll if="showTree" state="treeScroll" bg="sunken">
    <slot for="row in rows" name="row"/>
  </scroll>
  <!-- closing the last collection leaves this; nothing is made on disk until asked -->
  <col if="noCollection" grow="1" bg="sunken" pad="{{gutter}}" gap="8">
    <text muted>No collection is open</text>
    <button h="28" on-click="openFolder">Open collection folder...</button>
    <button h="28" on-click="newCollection">New collection...</button>
  </col>
</col>
`

	treefolderSource = `<list-row selected="{{row.Selected}}" border="{{row.DropBorder}}" h="{{rowH}}" pad-l="{{row.Indent}}" gap="6" on-click="row.Press" on-right-click="row.Menu" on-drag-over="row.DragOver">
  <row if="row.Stripe" w="3" h="14" bg="{{row.Stripe}}"/>
  <icon name="{{row.Triangle}}" w="12"/>
  <!-- the name, with the part the sidebar filter matched in the accent color; treerequest.vml has the same -->
  <text if="!row.Match" bold="{{row.Root}}" ellipsis grow="1">{{row.Name}}</text>
  <row if="row.Match" grow="1">
    <text bold="{{row.Root}}">{{row.Before}}</text>
    <text bold="{{row.Root}}" color="accent-line">{{row.Match}}</text>
    <text bold="{{row.Root}}" ellipsis grow="1">{{row.After}}</text>
  </row>
</list-row>
`

	treerequestSource = `<list-row selected="{{row.Selected}}" border="{{row.DropBorder}}" h="{{rowH}}" pad-l="{{row.Indent}}" pad-r="{{gutter}}" align-items="baseline" on-click="row.Press" on-right-click="row.Menu" on-drag-over="row.DragOver">
  <text small color="{{row.Color}}" w="{{badgeW}}">{{row.Badge}}</text>
  <!-- the name, with the part the sidebar filter matched in the accent color; treefolder.vml has the same -->
  <text if="!row.Match" ellipsis grow="1">{{row.Name}}</text>
  <row if="row.Match" grow="1">
    <text>{{row.Before}}</text>
    <text color="accent-line">{{row.Match}}</text>
    <text ellipsis grow="1">{{row.After}}</text>
  </row>
</list-row>
`

	headerSource = `<!-- In a window too narrow for everything the title gives up room first,
     down to 48 px, then the chip, so the settings button stays inside. -->
<row h="{{headerH}}" pad-x="{{gutter}}" bg="hero" control="hero">
  <text bold ellipsis shrink min-w="48" color="hero-text">{{title}}</text>
  <spacer/>
  <slot name="env"/>
  <spacer w="{{gutter}}"/>
  <row w="{{headerH}}" on-click="toggleMenu" tooltip="{{settingsTip}}">
    <icon name="gear" grow="1" color="hero-muted" hover-color="hero-text"/>
  </row>
</row>
`

	requesttabsSource = `<row h="{{tabsH}}" pad-t="{{tabGap}}" bg="frame" control="tabstrip" border-b="border">
  <hscroll state="tabsScroll" grow="1">
    <!-- inactive tabs blend into the strip; the active one joins the pane. A middle drag reorders, like a left one -->
    <row for="tab in tabs" active-tab="{{tab.Active}}" separator="{{tab.Separator}}" on-middle-click="tab.MiddlePress" on-right-click="tab.Menu">
      <row w="3" align-self="stretch" bg="{{tab.Stripe}}"/>
      <row pad-l="11" gap="8" align-items="baseline" on-click="tab.Press" on-drag-over="tab.DragOver">
        <text small color="{{tab.Color}}" w="{{tabBadgeW}}">{{tab.Badge}}</text>
        <text>{{tab.Name}}</text>
        <row w="{{tabDirtyW}}" align-self="stretch">
          <text if="tab.Dirty" color="accent-line" grow="1">{{dirtyMark}}</text>
        </row>
      </row>
      <row w="24" on-click="tab.Close" tooltip="{{closeTip}}">
        <icon name="cross" grow="1"/>
      </row>
    </row>
  </hscroll>
  <row w="40" on-click="newRequest" tooltip="{{newTip}}">
    <icon name="plus" grow="1"/>
  </row>
</row>
`

	urlbarSource = `<row h="{{urlH}}" pad-x="{{gutter}}" pad-y="8" gap="8" bg="bg">
  <row grow="1" bg="sunken" control="field" tooltip="{{urlTip}}">
    <row w="64" on-click="cycleProtocol" tooltip="Protocol: click to change">
      <text text-center grow="1" color="muted">{{protocol}}</text>
    </row>
    <!-- a divider between the chips, none after the last one; markup has no vrule-inset, so view.VRuleInset's tree is spelled out -->
    <row if="divided" w="1" pad-y="8">
      <vrule/>
    </row>
    <row if="httpMethod" w="84" on-click="cycleMethod" tooltip="Method: click to change">
      <text text-center grow="1" color="{{methodColor}}">{{method}}</text>
    </row>
    <row if="rpcMethod" w="{{grpcMethodW}}" pad-x="10" on-click="pickMethod" tooltip="Method: click to list the server's methods">
      <text ellipsis grow="1" color="{{grpcMethodColor}}">{{grpcMethod}}</text>
    </row>
    <row if="streamable" w="1" pad-y="8">
      <vrule/>
    </row>
    <row if="streamable" w="56" bg="{{streamBg}}" on-click="toggleStream" tooltip="Stream: keep the call open and show every message in the Messages pane">
      <text text-center grow="1" color="{{streamColor}}">Stream</text>
    </row>
    <!-- breathing room between the divider and the URL text -->
    <spacer w="{{urlTextGap}}"/>
    <field state="url" grow="1" placeholder="Enter URL" segments="segments" on-change="edited" on-enter="send" text-menu="url"/>
  </row>
  <primary-button if="idle" min-w="80" on-click="send" tooltip="{{sendTip}}">Send</primary-button>
  <primary-button if="socket" min-w="96" on-click="toggleConnection" tooltip="{{connectionTip}}">{{connectionLabel}}</primary-button>
  <primary-button if="sending" w="80" on-click="cancel" tooltip="Cancel the running request">Cancel</primary-button>
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

	responsestatusSource = `<row gap="12" pad-r="{{gutter}}" optional align-items="center">
  <row pad-x="10" h="20" bg="{{pillBg}}" control="pill" align-items="center">
    <text small bold color="{{pillFg}}">{{status}}</text>
  </row>
  <text>{{timing}}</text>
</row>
`

	statusbarSource = `<row h="{{statusH}}" pad-x="{{gutter}}" gap="{{gutter}}" bg="frame" control="statusbar" border-t="border">
  <text small muted ellipsis grow="1">{{status}}</text>
  <button w="44" h="20" on-click="openLog" tooltip="Show the log of calls">Log</button>
  <row on-click="openWelcome" tooltip="What is in this version">
    <text small muted>{{version}}</text>
  </row>
</row>
`
)
