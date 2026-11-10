package ui

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/collection"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/model/textfmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/component"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Sizes of the window chrome, in logical pixels.
const (
	sidebarW  = 260
	headerH   = 44
	tabsH     = 42
	urlH      = 56
	statusH   = 28
	paneTabsH = 40
	tabGap    = 6 // space above the tabs, so their outline never touches the banner or the bar above

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

// view is the whole window as a function of the app state. Its shape:
//
//	Stack
//	├─ Col                  the frame
//	│  ├─ Row
//	│  │  ├─ sidebar        collections tree
//	│  │  └─ mainColumn     header, request tabs, URL bar, request | response
//	│  └─ statusBar
//	└─ menuView             settings, floating above, only while open
func (a *App) view() Node {
	return Stack(
		Col(Bg(w.Frame),
			Split(&a.sidebarSplit, sidebarW-1, Grow(1), MinFirst(minSidebarW), MinSecond(minMainW),
				a.sidebar(),
				a.mainColumn(),
			),
			a.statusBar(),
		),
		a.menuView(),
		a.contextMenuView(),
		a.dialogView(),
		a.calls.View(a),
		a.toastsView(),
	)
}

// ── Sidebar ────────────────────────────────────────────────────────────

func (a *App) sidebar() Node {
	return Col(
		Row(H(headerH), PadX(w.Gutter), Gap(6), Bg(w.Bg), BorderB(w.Border),
			a.sidebarMode(),
			Field(&a.filter, H(28), Grow(1), Bg(w.Sunken), Placeholder("Filter"), a.textMenuOpt(&a.filter, nil)),
		),
		If(a.showSent, a.sent.List(a, a.filter.Text)),
		If(!a.showSent, Scroll(&a.treeScroll, Bg(w.Sunken),
			For(a.visibleRows(), func(_ int, row collection.Row) Node {
				return a.treeRow(row, row.Node() == a.selected)
			}),
		)),
	)
}

// sidebarMode is the header chip that switches the sidebar between the
// collection tree and the log of sent requests.
func (a *App) sidebarMode() Node {
	label, switchTo := "Collections", func() { a.showSent = true }
	if a.showSent {
		label, switchTo = "Sent", func() { a.showSent = false }
	}
	return Button(label, W(96), H(28), OnClick(switchTo))
}

func (a *App) treeRow(row collection.Row, selected bool) Node {
	look := HoverBg(w.Hover)
	if selected {
		look = Bg(w.Selected)
	}
	node := row.Node()
	indent := w.Gutter + row.Depth*treeIndent
	events := []Item{
		OnClick(func() { a.pressTreeRow(row) }),
		OnRightClick(func(x, y int) { a.openMenu(x, y, a.treeMenuItems(row)) }),
		OnDragOver(func() { a.dropTarget = node }),
		Control("row"),
	}
	if a.dragNode != nil && a.dropTarget == node && node != a.dragNode {
		events = append(events, Border(w.AccentLine)) // where the drop would land
	}

	if row.Folder {
		return Row(append([]Item{H(w.RowH), PadL(indent), Gap(6), look,
			TriangleIcon(row.Open, W(12)),
			a.rowName(row.Name),
		}, events...)...)
	}
	req := row.Req
	return Row(append([]Item{H(w.RowH), PadL(indent + 6), PadR(w.Gutter), AlignItems(AlignBaseline), look,
		Text(requestBadge(req.Protocol, req.Method), Small(), Color(appearance.MethodToken(req.Method)), W(badgeW)),
		a.rowName(row.Name),
	}, events...)...)
}

// rowName is a row's name with the part the sidebar filter matched in
// the accent color.
func (a *App) rowName(name string) Node {
	before, match, after := collection.SplitMatch(name, a.filter.Text)
	if match == "" {
		return Text(name, Ellipsis(), Grow(1))
	}
	return Row(Grow(1),
		Text(before), Text(match, Color(w.AccentText)), Text(after, Ellipsis(), Grow(1)),
	)
}

// pressTreeRow is a press on a row: it selects it, gives the tree the
// keyboard and starts a drag. A request opens at once; a folder toggles
// when the button is released on the same row (see finishDrag), so an
// open folder can still be dragged.
func (a *App) pressTreeRow(row collection.Row) {
	a.treeFocus = true
	a.dragNode = row.Node()
	if row.Dir == &a.col.Folder {
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
func requestBadge(protocol, method string) string {
	if protocol != "" && protocol != "rest" {
		return protocolLabel[protocol]
	}
	if method == "DELETE" {
		return "DEL"
	}
	return method
}

// ── Main column ────────────────────────────────────────────────────────

func (a *App) mainColumn() Node {
	t := a.cur()
	return Col(Grow(1),
		a.header(),
		a.requestTabs(),
		a.urlBar(t),
		a.panes(t),
	)
}

// panes is the request pane and the response pane with a draggable
// divider, side by side or stacked, as the layout setting says.
func (a *App) panes(t *reqTab) Node {
	items := []Item{
		AsRatio(), Grow(1), MinFirst(minPaneSize), MinSecond(minPaneSize),
		a.pane(t, requestPanes, &t.paneTab, &t.reqScroll, &t.reqTabsX, nil),
		a.pane(t, responsePanesFor(t), &t.respTab, &t.respScroll, &t.respTabsX, a.responseStatus(t)),
	}
	if a.settings.Layout == "stacked" {
		items = append(items, Vertical())
	}
	return Split(&a.paneSplit, 0.5, items...)
}

func (a *App) header() Node {
	return component.View(`
		<row h="44" pad-x="12" bg="hero" control="hero">
		  <text bold color="hero-text">{{title}}</text>
		  <spacer/>
		  <slot name="env"/>
		  <spacer w="12"/>
		  <row w="44" onclick="toggleMenu" tip="{{settingsTip}}">
		    <icon name="dots" grow="1" color="hero-muted" hover-color="hero-text"/>
		  </row>
		</row>`,
		component.Scope{
			"title":       a.col.Name,
			"env":         a.envChip(),
			"toggleMenu":  a.toggleSettings,
			"settingsTip": a.hint("Settings", "settings"),
		})
}

// requestTabs is the strip of open requests, with a "+" for a new one.
func (a *App) requestTabs() Node {
	return Row(H(tabsH), PadT(tabGap), Bg(w.Frame), Control("tabstrip"), BorderB(w.Border),
		HScroll(&a.tabsScroll, Grow(1), For(a.openTabs(), a.requestTab)),
		Row(W(40), OnClick(a.newRequest), Tooltip(a.hint("New request", "new-request")),
			PlusIcon(Grow(1)),
		),
	)
}

// requestTab is tab i of the strip: badge, name, dirty mark and the cross.
func (a *App) requestTab(i int, t *reqTab) Node {
	look := []Item{} // inactive tabs blend into the strip; the active one joins the pane
	if i == a.list.Active {
		look = []Item{Bg(w.Bg), Edges(w.AccentLine, true, true, true), KeepInView()} // outlined top, left and right, open at the bottom
	}
	return Row(append(look, OnMiddleClick(func() { a.updateTabs(tabs.DragBegin{Index: i}) }), // a middle drag reorders, like a left one
		OnRightClick(func(x, y int) { a.openMenu(x, y, a.tabMenuItems(i)) }),
		When(i > 0, Separator(12)),
		Row(PadL(14), Gap(8), AlignItems(AlignBaseline), OnClick(func() { a.pressTab(i) }), OnDragOver(func() { a.updateTabs(tabs.DragOver{Index: i}) }),
			Text(requestBadge(t.Req.Protocol, t.Req.Method), Small(), Color(appearance.MethodToken(t.Req.Method)), W(tabBadgeW)),
			Text(t.Req.Name),
			Row(W(tabDirtyW), AlignSelf(AlignStretch), If(dirty(t), Text(dirtyMark, Color(w.AccentLine), Grow(1)))),
		),
		Row(W(24), OnClick(func() { a.closeTab(i) }), Tooltip(a.hint("Close tab", "close-tab")),
			CrossIcon(Grow(1)),
		),
	)...)
}

// pressTab activates tab i and starts dragging it, unless the strip
// overflows: a left drag then pans the strip, and the middle button reorders.
func (a *App) pressTab(i int) {
	a.updateTabs(tabs.Activate{Index: i})
	if !a.tabsScroll.Overflowing {
		a.updateTabs(tabs.DragBegin{Index: i})
	}
}

// urlBar is [protocol | method | URL field ........] [Send].
func (a *App) urlBar(t *reqTab) Node {
	isREST, isGRPC := t.Req.Protocol == "rest", t.Req.Protocol == "grpc"
	send := PrimaryButton("Send", W(80), OnClick(func() { a.send(t) }), Tooltip(a.hint("Send the request", "send")))
	if isSocket(t) {
		label, hint := "Connect", "Open the connection"
		if t.Active() {
			label, hint = "Disconnect", "Close the connection"
		}
		send = PrimaryButton(label, W(96), OnClick(func() { a.toggleConnection(t) }), Tooltip(a.hint(hint, "send")))
	} else if t.Sending {
		send = PrimaryButton("Cancel", W(80), OnClick(func() { a.cancelSend(t) }), Tooltip("Cancel the running request"))
	}
	return Row(H(urlH), PadX(w.Gutter), PadY(8), Gap(8), Bg(w.Bg),
		Row(Grow(1), Bg(w.Sunken), Control("field"), Tooltip(a.urlTip(t)),
			tip("Protocol: click to change", chip(protocolLabel[t.Req.Protocol], w.Muted, 64, func() { a.cycleProtocol(t) })),
			If(isREST || isGRPC, VRuleInset(8)), // a divider between the chips, none after the last one
			If(isREST, tip("Method: click to change", chip(t.Req.Method, appearance.MethodToken(t.Req.Method), 84, func() { a.cycleMethod(t) }))),
			If(isGRPC, a.grpcMethodChip(t)),
			If(isGRPC, tip("Stream: keep the call open and show every message in the Messages pane", chip("Stream", streamColor(t), 56, func() { a.toggleStream(t) }))),
			Spacer(W(w.Gutter-4)), // breathing room between the divider and the URL text
			Field(&t.url, Grow(1),
				Placeholder("Enter URL"),
				Segments(a.varSegments),
				OnChange(func() { a.urlEdited(t) }),
				OnEnter(func() { a.send(t) }),
				a.textMenuOpt(&t.url, func() { a.urlEdited(t) }),
			),
		),
		send,
	)
}

// streamColor shows the Stream chip lit while the tab streams.
func streamColor(t *reqTab) w.Token {
	if t.grpcStream {
		return w.Accent
	}
	return w.Muted
}

// tip gives n a tooltip.
func tip(text string, n *El) *El {
	Tooltip(text)(n)
	return n
}

// hint is a tooltip with the action's shortcut, when it has one.
func (a *App) hint(text, shortcutID string) string {
	if key := a.shortcutHint(shortcutID); key != "" {
		return text + " (" + key + ")"
	}
	return text
}

// chip is a clickable label of fixed width, used for protocol and method.
func chip(label string, color w.Token, width int, onClick func()) *El {
	return Row(W(width), OnClick(onClick),
		Text(label, TextCenter(), Grow(1), Color(color)),
	)
}

// varSegments colors {{variables}} so they stand out from the literal
// URL: defined ones in the success color, unresolved ones as errors.
func (a *App) varSegments(s string) []Span {
	var segs []Span
	vars := a.varsFor(a.cur().Req)
	for _, seg := range splitVars(s) {
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
func splitVars(s string) []urlSeg {
	var segs []urlSeg
	for s != "" {
		start := strings.Index(s, "{{")
		end := strings.Index(s, "}}")
		if start < 0 || end < start {
			return append(segs, urlSeg{text: s})
		}
		if start > 0 {
			segs = append(segs, urlSeg{text: s[:start]})
		}
		segs = append(segs, urlSeg{text: s[start : end+2], isVar: true})
		s = s[end+2:]
	}
	return segs
}

// pane is one tabbed area: a tab strip, with extra at its right end,
// above the body of the active tab.
func (a *App) pane(t *reqTab, panes []pane, active *int, scroll, tabsX *ScrollState, extra Node) Node {
	selectTab := func(i int) { *active, scroll.Y = i, 0 }
	*active = min(*active, len(panes)-1) // a socket has fewer response panes
	return Col(Grow(1), Bg(w.Bg),
		Row(H(paneTabsH), PadT(tabGap), BorderB(w.Border),
			ScrollTabs(tabsX, a.tabItems(panes, t), *active, OnSelect(selectTab)),
			extra,
		),
		a.paneBody(panes[*active], t, scroll),
	)
}

// responseStatus is "200 OK  1.4s  156.67KB", dropped when the tabs need the room.
func (a *App) responseStatus(t *reqTab) Node {
	if t.Resp == nil || t.Resp.Err != "" {
		return nil
	}
	fg, bg := statusPill(t.Resp.Status)
	return Row(Gap(12), PadR(w.Gutter), Optional(), AlignItems(AlignCenter),
		Row(PadX(10), H(20), Bg(bg), Control("pill"), AlignItems(AlignCenter),
			Text(fmt.Sprintf("%d %s", t.Resp.Status, t.Resp.StatusText), Small(), Bold(), Color(fg)),
		),
		Text(fmt.Sprintf("%s  %s", textfmt.Duration(t.Resp.Duration), textfmt.Size(t.Resp.Size))),
	)
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

func (a *App) statusBar() Node {
	return component.View(`
		<row h="28" pad="12 0" gap="12" bg="frame" control="statusbar" border-t="border">
		  <text small muted ellipsis grow="1">{{status}}</text>
		  <button w="44" h="20" onclick="openLog" tip="Show the log of calls">Log</button>
		  <text small muted>{{version}}</text>
		</row>`,
		component.Scope{"status": a.status, "version": version, "openLog": a.calls.Open})
}
