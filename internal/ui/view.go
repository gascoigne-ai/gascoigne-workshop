package ui

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/textfmt"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/component"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
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
		a.toastsView(),
	)
}

// ── Sidebar ────────────────────────────────────────────────────────────

func (a *App) sidebar() Node {
	return Col(
		Row(H(headerH), PadXY(w.Pad, 0), Gap(6), Bg(w.Bg), BorderBottom(w.Border),
			Caption("Collections", W(96)),
			Field(&a.filter, H(28), Grow(1), Bg(w.Sunken), Placeholder("Filter"), a.textMenuOpt(&a.filter, nil)),
		),
		Scroll(&a.treeScroll, Bg(w.Sunken),
			For(a.visibleRows(), func(_ int, row collection.Row) Node {
				return a.treeRow(row, row.Node() == a.selected)
			}),
		),
	)
}

func (a *App) treeRow(row collection.Row, selected bool) Node {
	look := HoverBg(w.Hover)
	if selected {
		look = Bg(w.Selected)
	}
	node := row.Node()
	indent := w.Pad + row.Depth*treeIndent
	events := []Item{
		OnClick(func() { a.pressTreeRow(row) }),
		OnRightClick(func(x, y int) { a.openMenu(x, y, a.treeMenuItems(row)) }),
		OnHeldOver(func() { a.dropTarget = node }),
		Control("row"),
	}
	if a.dragNode != nil && a.dropTarget == node && node != a.dragNode {
		events = append(events, Border(w.AccentLine)) // where the drop would land
	}

	if row.Folder {
		return Row(append([]Item{H(w.RowH), PadL(indent), Gap(6), look,
			Triangle(row.Open, W(12)),
			Text(row.Name),
		}, events...)...)
	}
	req := row.Req
	return Row(append([]Item{H(w.RowH), PadL(indent + 6), PadR(w.Pad), AlignItems(AlignBaseline), look,
		Text(requestBadge(req.Protocol, req.Method), Small(), Color(appearance.MethodToken(req.Method)), W(badgeW)),
		Text(row.Name, Ellipsis(), Grow(1)),
	}, events...)...)
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
	toggleMenu := OnClick(func() { a.menu = !a.menu })
	return Row(H(headerH), PadXY(w.Pad, 0), Bg(w.Hero), Control("hero"),
		Text(a.col.Name, Bold(), Color(w.HeroText)),
		Spacer(),
		a.envChip(),
		Spacer(W(w.Pad)),
		Row(W(headerH), toggleMenu, Tooltip(a.hint("Settings", "settings")),
			Dots(Grow(1), Color(w.HeroMuted)),
		),
	)
}

// requestTabs is the strip of open requests, with a "+" for a new one.
func (a *App) requestTabs() Node {
	return Row(H(tabsH), PadT(tabGap), Bg(w.Frame), Control("tabstrip"), BorderBottom(w.Border),
		HScroll(&a.tabsScroll, Grow(1), For(a.tabs, func(i int, t *reqTab) Node {
			look := []Item{} // inactive tabs blend into the strip; the active one joins the pane
			if i == a.active {
				look = []Item{Bg(w.Bg), Edges(w.AccentLine, true, true, true), Reveal()} // outlined top, left and right, open at the bottom
			}
			grab := OnClick(func() {
				a.active = i
				if !a.tabsScroll.Overflowing { // an overflowing strip is panned by a left drag; the middle button reorders
					a.dragTab = i
				}
			})
			return Row(append(look, OnMiddleClick(func() { a.active, a.dragTab = i, i }), // a middle drag reorders, like a left one
				OnRightClick(func(x, y int) { a.openMenu(x, y, a.tabMenuItems(i)) }),
				Separator(i > 0, 12),
				Row(PadL(14), Gap(8), AlignItems(AlignBaseline), grab, OnHeldOver(func() { a.dragOver(i) }),
					Text(requestBadge(t.req.Protocol, t.req.Method), Small(), Color(appearance.MethodToken(t.req.Method)), W(tabBadgeW)),
					Text(t.req.Name),
					Row(W(tabDirtyW), AlignSelf(AlignStretch), If(dirty(t), Text(dirtyMark, Color(w.AccentLine), Grow(1)))),
				),
				Row(W(24), OnClick(func() { a.closeTab(i) }), Tooltip(a.hint("Close tab", "close-tab")),
					Cross(Grow(1)),
				),
			)...)
		})),
		Row(W(40), OnClick(a.newRequest), Tooltip(a.hint("New request", "new-request")),
			Plus(Grow(1)),
		),
	)
}

// urlBar is [protocol | method | URL field ........] [Send].
func (a *App) urlBar(t *reqTab) Node {
	isREST := t.req.Protocol == "rest"
	send := PrimaryButton("Send", W(80), OnClick(func() { a.send(t) }), Tooltip(a.hint("Send the request", "send")))
	if isSocket(t) {
		label, hint := "Connect", "Open the connection"
		if t.sess != nil || t.connecting || t.wantOpen {
			label, hint = "Disconnect", "Close the connection"
		}
		send = PrimaryButton(label, W(96), OnClick(func() { a.toggleConnection(t) }), Tooltip(a.hint(hint, "send")))
	} else if t.sending {
		send = PrimaryButton("Cancel", W(80), OnClick(func() { a.cancelSend(t) }), Tooltip("Cancel the running request"))
	}
	return Row(H(urlH), PadXY(w.Pad, 8), Gap(8), Bg(w.Bg),
		Row(Grow(1), Bg(w.Sunken), Control("field"),
			tip("Protocol: click to change", chip(protocolLabel[t.req.Protocol], w.Muted, 64, func() { a.cycleProtocol(t) })),
			If(isREST, VRuleInset(8)), // a divider between the chips, none after the last one
			If(isREST, tip("Method: click to change", chip(t.req.Method, appearance.MethodToken(t.req.Method), 84, func() { a.cycleMethod(t) }))),
			Spacer(W(w.Pad-4)), // breathing room between the divider and the URL text
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
		Text(label, Center(), Grow(1), Color(color)),
	)
}

// varSegments colors {{variables}} so they stand out from the literal
// URL: defined ones in the success color, unresolved ones as errors.
func (a *App) varSegments(s string) []Seg {
	var segs []Seg
	vars := a.vars()
	for _, seg := range splitVars(s) {
		style := w.Text
		if seg.isVar {
			style = w.OK
			if !model.Defined(strings.TrimSuffix(strings.TrimPrefix(seg.text, "{{"), "}}"), vars) {
				style = w.Error
			}
		}
		segs = append(segs, Seg{Text: seg.text, Style: style})
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
		Row(H(paneTabsH), PadT(tabGap), BorderBottom(w.Border),
			ScrollTabs(tabsX, a.tabItems(panes, t), *active, selectTab),
			extra,
		),
		a.paneBody(panes[*active], t, scroll),
	)
}

// responseStatus is "200 OK  1.4s  156.67KB", dropped when the tabs need the room.
func (a *App) responseStatus(t *reqTab) Node {
	if t.resp == nil || t.resp.Err != "" {
		return nil
	}
	fg, bg := statusPill(t.resp.Status)
	return Row(Gap(12), PadR(w.Pad), Optional(), AlignItems(AlignCenter),
		Row(PadXY(10, 0), H(20), Bg(bg), Control("pill"), AlignItems(AlignCenter),
			Text(fmt.Sprintf("%d %s", t.resp.Status, t.resp.StatusText), Small(), Bold(), Color(fg)),
		),
		Text(fmt.Sprintf("%s  %s", textfmt.Duration(t.resp.Duration), textfmt.Size(t.resp.Size))),
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
		<row h="28" pad="12 0" gap="12" bg="frame" control="statusbar" border-top="border">
		  <text small muted ellipsis grow="1">{{status}}</text>
		  <text small muted>{{version}}</text>
		</row>`,
		component.Scope{"status": a.status, "version": version})
}
