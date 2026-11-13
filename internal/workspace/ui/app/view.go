package ui

import (
	"embed"
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

// view is the whole window as a function of the app state. Its shape:
//
//	Stack
//	├─ Col                  the frame
//	│  ├─ Row
//	│  │  ├─ sidebar        collections tree
//	│  │  └─ mainColumn     header, request tabs, URL bar, request | response
//	│  └─ statusBar
//	├─ menuView             settings, floating above, only while open
//	└─ TooltipLayer         the tooltip of the hovered element, last so it is on top
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
		PopupMenuLayer(a.ctx),
		a.dialogView(),
		a.calls.View(a),
		a.toastsView(),
		TooltipLayer(&a.tips),
	)
}

// ── Sidebar ────────────────────────────────────────────────────────────

//go:embed sidebar.vml treefolder.vml treerequest.vml
var sidebarTemplates embed.FS

var (
	sidebarTemplate     = markup.MustLoad(sidebarTemplates, "sidebar.vml")
	treeFolderTemplate  = markup.MustLoad(sidebarTemplates, "treefolder.vml")
	treeRequestTemplate = markup.MustLoad(sidebarTemplates, "treerequest.vml")
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
		"mode":       label,
		"switchMode": switchTo,
		"filter":     &a.filter,
		"showSent":   a.showSent,
		"sentList":   sentList,
		"treeScroll": &a.treeScroll,
		"rows":       rows,
		"headerH":    headerH,
		"gutter":     w.Gutter,
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
	Triangle string  // a folder's disclosure icon
	Badge    string  // a request's method or protocol
	Color    w.Token // the badge's color
	// DropBorder outlines the row a drag would drop on; empty, no outline.
	DropBorder w.Token
	Press      func()
	Menu       func(pointerX, pointerY int)
	DragOver   func()
}

// treeRow is one row of the tree, a folder or a request: one template
// each, since their rows differ in padding and alignment.
func (a *App) treeRow(row collection.Row, selected bool) Node {
	node := row.Node()
	item := treeItem{
		Name:     row.Name,
		Indent:   w.Gutter + row.Depth*treeIndent,
		Selected: selected,
		Press:    func() { a.pressTreeRow(row) },
		Menu:     func(pointerX, pointerY int) { a.openMenu(pointerX, pointerY, a.treeMenuItems(row)) },
		DragOver: func() { a.dropTarget = node },
	}
	item.Before, item.Match, item.After = collection.SplitMatch(row.Name, a.filter.Text)
	template := treeFolderTemplate
	if row.Folder {
		item.Triangle = "triangle-right"
		if row.Open {
			item.Triangle = "triangle-down"
		}
	} else {
		template = treeRequestTemplate
		item.Indent += 6
		item.Badge = requestBadge(row.Req.Protocol, row.Req.Method)
		item.Color = appearance.MethodToken(row.Req.Method)
	}
	if a.dragNode != nil && a.dropTarget == node && node != a.dragNode {
		item.DropBorder = w.AccentLine
	}
	return template.MustBuild(markup.Scope{"row": item, "rowH": w.RowH, "gutter": w.Gutter, "badgeW": badgeW})
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
		a.pane(t, requestPanesFor(t), &t.paneTab, &t.reqScroll, &t.reqTabsX, nil),
		a.pane(t, responsePanesFor(t), &t.respTab, &t.respScroll, &t.respTabsX, a.responseStatus(t)),
	}
	if a.settings.Layout == "stacked" {
		items = append(items, Vertical())
	}
	return Split(&a.paneSplit, 0.5, items...)
}

func (a *App) header() Node {
	return markup.View(`
		<row h="44" pad-x="12" bg="hero" control="hero">
		  <text bold color="hero-text">{{title}}</text>
		  <spacer/>
		  <slot name="env"/>
		  <spacer w="12"/>
		  <row w="44" on-click="toggleMenu" tooltip="{{settingsTip}}">
		    <icon name="dots" grow="1" color="hero-muted" hover-color="hero-text"/>
		  </row>
		</row>`,
		markup.Scope{
			"title":       a.col.Name,
			"env":         a.envChip(),
			"toggleMenu":  a.toggleSettings,
			"settingsTip": a.hint("Settings", "settings"),
		})
}

// requestTabs is the strip of open requests, with a "+" for a new one.
func (a *App) requestTabs() Node {
	return Row(H(tabsH), PadT(w.TabGap), Bg(w.Frame), Control("tabstrip"), BorderB(w.Border),
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
		look = []Item{ActiveTab()}
	}
	return Row(append(look, OnMiddleClick(func() { a.updateTabs(tabs.DragBegin{Index: i}) }), // a middle drag reorders, like a left one
		OnRightClick(func(x, y int) { a.openMenu(x, y, a.tabMenuItems(i)) }),
		When(i > 0, Separator(12)),
		Row(PadL(14), Gap(8), AlignItems(AlignBaseline), OnClick(func() { a.pressTab(i) }), OnDragOver(func() { a.updateTabs(tabs.DragOver{Index: i}) }),
			Text(requestBadge(t.Req.Protocol, t.Req.Method), Small(), Color(appearance.MethodToken(t.Req.Method)), W(tabBadgeW)),
			Text(t.Req.Name),
			Row(W(tabDirtyW), AlignSelf(AlignStretch), If(a.tabDirty(t), Text(dirtyMark, Color(w.AccentLine), Grow(1)))),
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

//go:embed urlbar.vml
var urlBarSource embed.FS

var urlBarTemplate = markup.MustLoad(urlBarSource, "urlbar.vml")

// urlBar is [protocol | method | URL field ........] [Send]. The button
// is one of three: Send, Connect or Disconnect on a socket, and Cancel
// while a request runs.
func (a *App) urlBar(t *reqTab) Node {
	isREST, isGRPC, socket := t.Req.Protocol == "rest", t.Req.Protocol == "grpc", isSocket(t)
	sending := !socket && t.Sending
	connectionLabel, connectionHint := "Connect", "Open the connection"
	if t.Active() {
		connectionLabel, connectionHint = "Disconnect", "Close the connection"
	}
	grpcMethod, grpcMethodColor := grpcMethodLabel(t)
	return urlBarTemplate.MustBuild(markup.Scope{
		"urlH":             urlH,
		"gutter":           w.Gutter,
		"urlTextGap":       w.Gutter - 4,
		"urlTip":           a.urlTip(t),
		"protocol":         protocolLabel[t.Req.Protocol],
		"divided":          isREST || isGRPC,
		"rest":             isREST,
		"method":           t.Req.Method,
		"methodColor":      appearance.MethodToken(t.Req.Method),
		"grpc":             isGRPC,
		"grpcMethodW":      grpcMethodW,
		"grpcMethod":       grpcMethod,
		"grpcMethodColor":  grpcMethodColor,
		"streamColor":      streamColor(t),
		"url":              &t.url,
		"segments":         a.varSegments,
		"idle":             !socket && !sending,
		"socket":           socket,
		"sending":          sending,
		"sendTip":          a.hint("Send the request", "send"),
		"connectionLabel":  connectionLabel,
		"connectionTip":    a.hint(connectionHint, "send"),
		"cycleProtocol":    func() { a.cycleProtocol(t) },
		"cycleMethod":      func() { a.cycleMethod(t) },
		"pickMethod":       func() { a.pickGRPCMethod(t) },
		"toggleStream":     func() { a.toggleStream(t) },
		"edited":           func() { a.urlEdited(t) },
		"send":             func() { a.send(t) },
		"toggleConnection": func() { a.toggleConnection(t) },
		"cancel":           func() { a.cancelSend(t) },
	})
}

// streamColor shows the Stream chip lit while the tab streams.
func streamColor(t *reqTab) w.Token {
	if t.grpcStream {
		return w.Accent
	}
	return w.Muted
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
		TabBar(
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
	return markup.View(`
		<row h="28" pad="12 0" gap="12" bg="frame" control="statusbar" border-t="border">
		  <text small muted ellipsis grow="1">{{status}}</text>
		  <button w="44" h="20" on-click="openLog" tooltip="Show the log of calls">Log</button>
		  <text small muted>{{version}}</text>
		</row>`,
		markup.Scope{"status": a.status, "version": version, "openLog": a.calls.Open})
}
