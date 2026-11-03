package ui

import (
	"fmt"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/vml"
	"github.com/gascoigne-ai/gascoigne-workshop/widget"
)

// Sizes of the window chrome, in logical pixels.
const (
	sidebarW   = 260
	headerH    = 44
	tabsH      = 40
	urlH       = 56
	statusH    = 28
	paneTabsH  = 40
	treeIndent = 16
	badgeW     = 40
)

// view is the whole window as a function of the app state: a frame
// with the sidebar, the main column and the status bar, and the
// settings menu floating above when open.
func (a *App) view() vml.Node {
	return vml.Stack(
		vml.Col(
			vml.Row(a.sidebar(), vml.VRule(), a.mainColumn()).Grow(1),
			a.statusBar(),
		).Bg(widget.Frame),
		a.menuView(),
	)
}

func (a *App) sidebar() vml.Node {
	cur := a.cur().req
	return vml.Col(
		vml.Row(vml.Caption("Collections")).PadXY(widget.Pad, 0).H(headerH).Bg(widget.Bg).BorderBottom(widget.Rule),
		vml.Col(vml.For(treeRows(a.col.Folder, a.expanded), func(_ int, row treeRow) vml.Node {
			return a.treeRow(row, row.Req != nil && row.Req == cur)
		})).Grow(1).Bg(widget.Sunken).Clip(),
	).W(min(sidebarW, a.w/3) - 1)
}

func (a *App) treeRow(row treeRow, selected bool) vml.Node {
	pad := widget.Pad + row.Depth*treeIndent
	var line *vml.El
	if row.Folder {
		line = vml.Row(vml.Triangle(row.Open).W(12), vml.Text(row.Name)).Gap(6).PadL(pad)
	} else {
		req := row.Req
		line = vml.Row(
			vml.Text(requestBadge(req.Protocol, req.Method)).Small().Color(methodToken(req.Method)).W(badgeW),
			vml.Text(row.Name).Ellipsis().Grow(1),
		).PadL(pad + 6).PadR(widget.Pad)
	}
	line.H(widget.RowH).OnClick(func() { a.clickTree(row) })
	if selected {
		return line.Bg(widget.Selected)
	}
	return line.HoverBg(widget.Hover)
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

func (a *App) mainColumn() vml.Node {
	t := a.cur()
	return vml.Col(
		a.header(),
		a.tabStrip(),
		a.urlBar(t),
		vml.Row(
			a.pane(t, requestPanes, &t.paneTab, nil),
			vml.VRule(),
			a.pane(t, responsePanes, &t.respTab, a.responseStatus(t)),
		).Grow(1),
	).Grow(1)
}

func (a *App) header() vml.Node {
	return vml.Row(
		vml.Text(a.col.Name),
		vml.Spacer(),
		vml.Row(vml.Dots().Grow(1)).W(headerH).OnClick(func() { a.menu = !a.menu }),
	).PadXY(widget.Pad, 0).H(headerH).Bg(widget.Bg)
}

func (a *App) tabStrip() vml.Node {
	return vml.Row(
		vml.For(a.tabs, func(i int, t *reqTab) vml.Node {
			tab := vml.Row(
				vml.Text(requestBadge(t.req.Protocol, t.req.Method)).Small().Color(methodToken(t.req.Method)),
				vml.Text(t.req.Name),
			).Gap(8).PadXY(14, 0).OnClick(func() { a.active = i })
			if i == a.active {
				tab.Bg(widget.Bg)
			}
			return tab
		}),
		vml.Row(vml.Text("+").Muted().Center().Grow(1)).W(40).OnClick(a.newRequest),
	).H(tabsH).Bg(widget.Frame).BorderBottom(widget.Border)
}

func (a *App) urlBar(t *reqTab) vml.Node {
	rest := t.req.Protocol == "rest"
	send := "Send"
	if t.sending {
		send = "..."
	}
	return vml.Row(
		vml.Row(
			chip(protocolLabel[t.req.Protocol], widget.Muted, 64, func() { a.cycleProtocol(t) }),
			vml.VRuleInset(8),
			vml.If(rest, chip(t.req.Method, methodToken(t.req.Method), 84, func() { a.cycleMethod(t) })),
			vml.If(rest, vml.VRuleInset(8)),
			vml.Field(&t.url).
				Placeholder("Enter URL").
				Segments(varSegments).
				OnChange(func() { a.urlEdited(t) }).
				OnEnter(func() { a.send(t) }).
				Grow(1),
		).Grow(1).Bg(widget.Sunken).Border(widget.Border),
		vml.PrimaryButton(send).W(80).OnClick(func() { a.send(t) }),
	).Gap(8).PadXY(widget.Pad, 8).H(urlH).Bg(widget.Bg)
}

// chip is a clickable label of fixed width, used for protocol and method.
func chip(label string, clr widget.Token, w int, onClick func()) vml.Node {
	return vml.Row(vml.Text(label).Center().Grow(1).Color(clr)).W(w).OnClick(onClick)
}

// varSegments colors {{variables}} so they stand out from the literal URL.
func varSegments(s string) []vml.Seg {
	var segs []vml.Seg
	for _, seg := range splitVars(s) {
		style := widget.Text
		if seg.isVar {
			style = widget.OK
		}
		segs = append(segs, vml.Seg{Text: seg.text, Style: style})
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
func (a *App) pane(t *reqTab, panes []pane, active *int, extra vml.Node) vml.Node {
	p := panes[*active]
	return vml.Col(
		vml.Row(
			vml.Tabs(a.tabItems(panes, t), *active, func(i int) { *active = i }),
			vml.Spacer(),
			extra,
		).H(paneTabsH).BorderBottom(widget.Border),
		a.paneBody(p, t),
	).Grow(1).Bg(widget.Bg)
}

// responseStatus is "200 OK  1.4s  156.67KB", dropped when the tabs need the room.
func (a *App) responseStatus(t *reqTab) vml.Node {
	if t.resp == nil || t.resp.Err != "" {
		return nil
	}
	clr := widget.OK
	if t.resp.Status >= 400 {
		clr = widget.Error
	}
	return vml.Row(
		vml.Text(fmt.Sprintf("%d %s", t.resp.Status, t.resp.StatusText)).Color(clr),
		vml.Text(fmt.Sprintf("%s  %s", formatDuration(t.resp.Duration), formatSize(t.resp.Size))),
	).Gap(12).PadR(widget.Pad).Optional()
}

func (a *App) statusBar() vml.Node {
	return vml.Row(
		vml.Text(a.status).Small().Muted().Ellipsis().Grow(1),
		vml.Text(version).Small().Muted(),
	).Gap(widget.Pad).PadXY(widget.Pad, 0).H(statusH).Bg(widget.Frame).BorderTop(widget.Border)
}
