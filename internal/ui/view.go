package ui

import (
	"fmt"
	"strings"

	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
	. "github.com/gascoigne-ai/gascoigne-workshop/vml"
	w "github.com/gascoigne-ai/gascoigne-workshop/widget"
)

// Sizes of the window chrome, in logical pixels.
const (
	sidebarW  = 260
	headerH   = 44
	tabsH     = 40
	urlH      = 56
	statusH   = 28
	paneTabsH = 40

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
			For(a.visibleRows(), func(_ int, row treeRow) Node {
				return a.treeRow(row, row.node() == a.selected)
			}),
		),
	)
}

func (a *App) treeRow(row treeRow, selected bool) Node {
	look := HoverBg(w.Hover)
	if selected {
		look = Bg(w.Selected)
	}
	node := row.node()
	indent := w.Pad + row.Depth*treeIndent
	events := []Item{
		OnClick(func() { a.pressTreeRow(row) }),
		OnRightClick(func(x, y int) { a.openMenu(x, y, a.treeMenuItems(row)) }),
		OnHeldOver(func() { a.dropTarget = node }),
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
		Text(requestBadge(req.Protocol, req.Method), Small(), Color(methodToken(req.Method)), W(badgeW)),
		Text(row.Name, Ellipsis(), Grow(1)),
	}, events...)...)
}

// pressTreeRow is a press on a row: it selects it, gives the tree the
// keyboard and starts a drag. A request opens at once; a folder toggles
// when the button is released on the same row (see finishDrag), so an
// open folder can still be dragged.
func (a *App) pressTreeRow(row treeRow) {
	a.treeFocus = true
	a.dragNode = row.node()
	if row.Dir == &a.col.Folder {
		a.dragNode = nil // the collection itself cannot be moved
	}
	if row.Folder {
		a.selectNode(row.node())
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
		a.pane(t, requestPanes, &t.paneTab, &t.reqScroll, nil),
		a.pane(t, responsePanes, &t.respTab, &t.respScroll, a.responseStatus(t)),
	}
	if a.settings.Layout == "stacked" {
		items = append(items, Vertical())
	}
	return Split(&a.paneSplit, 0.5, items...)
}

func (a *App) header() Node {
	toggleMenu := OnClick(func() { a.menu = !a.menu })
	return Row(H(headerH), PadXY(w.Pad, 0), Bg(w.Bg),
		Text(a.col.Name),
		Spacer(),
		Row(W(headerH), toggleMenu,
			Dots(Grow(1)),
		),
	)
}

// requestTabs is the strip of open requests, with a "+" for a new one.
func (a *App) requestTabs() Node {
	return Row(H(tabsH), Bg(w.Frame), BorderBottom(w.Border),
		For(a.tabs, func(i int, t *reqTab) Node {
			look := Bg(w.Frame) // inactive tabs blend into the strip; the active one joins the pane
			if i == a.active {
				look = Bg(w.Bg)
			}
			grab := OnClick(func() { a.active, a.dragTab = i, i })
			return Row(look, OnMiddleClick(func() { a.closeTab(i) }),
				OnRightClick(func(x, y int) { a.openMenu(x, y, a.tabMenuItems(i)) }),
				Row(PadL(14), Gap(8), AlignItems(AlignBaseline), grab, OnHeldOver(func() { a.dragOver(i) }),
					Text(requestBadge(t.req.Protocol, t.req.Method), Small(), Color(methodToken(t.req.Method)), W(tabBadgeW)),
					Text(t.req.Name),
					Row(W(tabDirtyW), AlignSelf(AlignStretch), If(dirty(t), Text(dirtyMark, Color(w.AccentLine), Grow(1)))),
				),
				Row(W(24), OnClick(func() { a.closeTab(i) }),
					Cross(Grow(1)),
				),
			)
		}),
		Row(W(40), OnClick(a.newRequest),
			Text("+", Muted(), Center(), Grow(1)),
		),
	)
}

// urlBar is [protocol | method | URL field ........] [Send].
func (a *App) urlBar(t *reqTab) Node {
	isREST := t.req.Protocol == "rest"
	send := PrimaryButton("Send", W(80), OnClick(func() { a.send(t) }))
	if t.sending {
		send = PrimaryButton("Cancel", W(80), OnClick(func() { a.cancelSend(t) }))
	}
	return Row(H(urlH), PadXY(w.Pad, 8), Gap(8), Bg(w.Bg),
		Row(Grow(1), Bg(w.Sunken), Border(w.Border),
			chip(protocolLabel[t.req.Protocol], w.Muted, 64, func() { a.cycleProtocol(t) }),
			VRuleInset(8),
			If(isREST, chip(t.req.Method, methodToken(t.req.Method), 84, func() { a.cycleMethod(t) })),
			If(isREST, VRuleInset(8)),
			Field(&t.url, Grow(1),
				Placeholder("Enter URL"),
				Segments(varSegments),
				OnChange(func() { a.urlEdited(t) }),
				OnEnter(func() { a.send(t) }),
				a.textMenuOpt(&t.url, func() { a.urlEdited(t) }),
			),
		),
		send,
	)
}

// chip is a clickable label of fixed width, used for protocol and method.
func chip(label string, color w.Token, width int, onClick func()) Node {
	return Row(W(width), OnClick(onClick),
		Text(label, Center(), Grow(1), Color(color)),
	)
}

// varSegments colors {{variables}} so they stand out from the literal URL.
func varSegments(s string) []Seg {
	var segs []Seg
	for _, seg := range splitVars(s) {
		style := w.Text
		if seg.isVar {
			style = w.OK
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
func (a *App) pane(t *reqTab, panes []pane, active *int, scroll *ScrollState, extra Node) Node {
	selectTab := func(i int) { *active, scroll.Y = i, 0 }
	return Col(Grow(1), Bg(w.Bg),
		Row(H(paneTabsH), BorderBottom(w.Border),
			Tabs(a.tabItems(panes, t), *active, selectTab),
			Spacer(),
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
	statusColor := w.OK
	if t.resp.Status >= 400 {
		statusColor = w.Error
	}
	return Row(Gap(12), PadR(w.Pad), Optional(),
		Text(fmt.Sprintf("%d %s", t.resp.Status, t.resp.StatusText), Color(statusColor)),
		Text(fmt.Sprintf("%s  %s", formatDuration(t.resp.Duration), formatSize(t.resp.Size))),
	)
}

// ── Status bar ─────────────────────────────────────────────────────────

func (a *App) statusBar() Node {
	return Row(H(statusH), PadXY(w.Pad, 0), Gap(w.Pad), Bg(w.Frame), BorderTop(w.Border),
		Text(a.status, Small(), Muted(), Ellipsis(), Grow(1)),
		Text(version, Small(), Muted()),
	)
}
