package ui

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	treeIndent = 16
	badgeW     = 40
)

func (a *App) drawWindow(dst *ebiten.Image) {
	c := &a.ctx
	th := c.Th
	l := Compute(a.w, a.h)
	dst.Fill(th.Bg)

	a.drawSidebar(dst, l)
	a.drawHeader(dst, l.Header)

	rects, _ := a.tabStripRects(l.Tabs)
	a.drawTabStrip(dst, l.Tabs, rects)

	t := a.cur()
	a.drawURLBar(dst, t, l.URL)
	a.drawPanes(dst, t, l.Request, requestPanes, t.paneTab)
	a.drawPanes(dst, t, l.Response, responsePanes, t.respTab)
	c.fill(dst, l.Split, th.Border)
	a.drawResponseStatus(dst, t, l.Response)
	a.drawStatusBar(dst, l.Status)
}

func (a *App) drawSidebar(dst *ebiten.Image, l Layout) {
	c := &a.ctx
	th := c.Th
	side := Rect{0, 0, l.SidebarDivider.X + 1, l.Tree.Y + l.Tree.H}
	c.fill(dst, side, th.Sidebar)
	c.fill(dst, l.SidebarDivider, th.Border)
	c.textMid(dst, c.F.UI, "Collections", th.Pad, l.SidebarHead, th.Text)
	c.hline(dst, l.SidebarHead.X, l.SidebarHead.Y+l.SidebarHead.H-1, l.SidebarHead.W-1, th.Border)

	tree := c.clip(dst, l.Tree)
	cur := a.cur().req
	for i, row := range treeRows(a.col.Folder, a.expanded) {
		rr := Rect{l.Tree.X, l.Tree.Y + i*th.RowH, l.Tree.W - 1, th.RowH}
		switch {
		case row.Req != nil && row.Req == cur:
			c.fill(tree, rr, th.Selected)
		case c.In.Over(rr) && c.In.Over(l.Tree):
			c.fill(tree, rr, th.Hover)
		}
		x := rr.X + th.Pad + row.Depth*treeIndent
		if row.Folder {
			c.triangle(tree, float32(x+6), float32(rr.Y+rr.H/2), 4, row.Open, th.Muted)
			c.textMid(tree, c.F.UI, row.Name, x+18, rr, th.Text)
			continue
		}
		c.textMid(tree, c.F.Small, requestBadge(row.Req.Protocol, row.Req.Method), x+6, rr, th.MethodColor(row.Req.Method))
		c.textMid(tree, c.F.UI, c.ellipsize(c.F.UI, row.Name, rr.W-(x+6+badgeW-rr.X)-th.Pad), x+6+badgeW, rr, th.Text)
	}
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

func (a *App) drawHeader(dst *ebiten.Image, r Rect) {
	c := &a.ctx
	c.textMid(dst, c.F.UI, a.col.Name, r.X+c.Th.Pad, r, c.Th.Text)
}

// tabStripRects places the open-request tabs and the new-tab button.
func (a *App) tabStripRects(r Rect) (tabs []Rect, plus Rect) {
	c := &a.ctx
	x := r.X
	for _, t := range a.tabs {
		w := 14 + c.width(c.F.Small, requestBadge(t.req.Protocol, t.req.Method)) + 8 + c.width(c.F.UI, t.req.Name) + 14
		tabs = append(tabs, Rect{x, r.Y, w, r.H})
		x += w
	}
	return tabs, Rect{x, r.Y, 40, r.H}
}

func (a *App) drawTabStrip(dst *ebiten.Image, r Rect, rects []Rect) {
	c := &a.ctx
	th := c.Th
	c.fill(dst, r, th.Sidebar)
	c.hline(dst, r.X, r.Y+r.H-1, r.W, th.Border)
	for i, tr := range rects {
		t := a.tabs[i]
		if i == a.active {
			c.fill(dst, tr, th.Bg)
			c.outline(dst, tr, th.Border)
			c.fill(dst, Rect{tr.X + 1, tr.Y + tr.H - 1, tr.W - 2, 1}, th.Bg)
		}
		badge := requestBadge(t.req.Protocol, t.req.Method)
		c.textMid(dst, c.F.Small, badge, tr.X+14, tr, th.MethodColor(t.req.Method))
		c.textMid(dst, c.F.UI, t.req.Name, tr.X+14+c.width(c.F.Small, badge)+8, tr, th.Text)
	}
	_, plus := a.tabStripRects(r)
	c.textCenter(dst, c.F.UI, "+", plus, th.Muted)
}

func (a *App) drawURLBar(dst *ebiten.Image, t *reqTab, r Rect) {
	c := &a.ctx
	th := c.Th
	protoR, methodR, field, send := urlBar(r, t.req.Protocol == "rest")
	box := Rect{protoR.X, protoR.Y, field.X + field.W - protoR.X, protoR.H}
	c.fill(dst, box, th.Panel)
	c.outline(dst, box, th.Border)

	c.textCenter(dst, c.F.UI, protocolLabel[t.req.Protocol], protoR, th.Muted)
	if methodR.W > 0 {
		c.textCenter(dst, c.F.UI, t.req.Method, methodR, th.MethodColor(t.req.Method))
	}
	for _, chip := range []Rect{protoR, methodR} {
		if chip.W > 0 {
			c.fill(dst, Rect{chip.X + chip.W - 1, chip.Y + 8, 1, chip.H - 16}, th.Border)
		}
	}

	a.drawURLText(dst, t, field)
	label := "Send"
	if t.sending {
		label = "..."
	}
	c.button(dst, send, label, th.Accent, th.AccentText)
}

func (a *App) drawURLText(dst *ebiten.Image, t *reqTab, field Rect) {
	c := &a.ctx
	clipped := c.clip(dst, field)
	x := field.X + 10
	// {{variables}} get their own color so they stand out from the
	// literal URL text.
	for _, seg := range splitVars(t.url.Text) {
		clr := c.Th.Text
		if seg.isVar {
			clr = c.Th.OK
		}
		c.textMid(clipped, c.F.UI, seg.text, x, field, clr)
		x += c.width(c.F.UI, seg.text)
	}
	if t.url.Text == "" {
		c.textMid(clipped, c.F.UI, "Enter URL", x, field, c.Th.Muted)
	}
	if t.url.Focused && (c.Tick/30)%2 == 0 {
		cx := field.X + 10 + c.width(c.F.UI, string([]rune(t.url.Text)[:t.url.Cursor]))
		c.fill(clipped, Rect{cx, field.Y + 8, 1, field.H - 16}, c.Th.Text)
	}
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

func (a *App) drawPanes(dst *ebiten.Image, t *reqTab, r Rect, panes []pane, active int) {
	c := &a.ctx
	tabsR, _ := r.SplitTop(paneTabsH)
	c.drawTabs(dst, tabsR, a.tabItems(panes, t), active)

	p := panes[active]
	content := a.contentRect(r)
	if msg, isErr := paneEmpty(p, t); msg != "" {
		clr := c.Th.Muted
		if isErr {
			clr = c.Th.Error
		}
		a.drawLines(dst, content, msg, clr)
		return
	}
	switch p.kind {
	case paneCode:
		c.drawCodeView(dst, a.codeRect(r), p.code(t))
	case paneNote:
		a.drawLines(dst, content, p.note(t), c.Th.Text)
	case paneTable:
		if p.title != "" {
			c.textMid(dst, c.F.UI, p.title, content.X, Rect{content.X, content.Y, content.W, c.Th.RowH}, c.Th.Muted)
		}
		c.drawTable(c.clip(dst, content), a.tableRect(content, p), *p.rows(a, t), true, p.editable)
	}
}

// drawLines draws multi-line text from the top-left of r.
func (a *App) drawLines(dst *ebiten.Image, r Rect, s string, clr color.RGBA) {
	c := &a.ctx
	for i, line := range strings.Split(s, "\n") {
		c.text(dst, c.F.UI, line, r.X, r.Y+i*(c.lineH(c.F.UI)+6), clr)
	}
}

// drawResponseStatus shows "200 OK  1.4s  156.67KB" at the right end of
// the response tab strip.
func (a *App) drawResponseStatus(dst *ebiten.Image, t *reqTab, r Rect) {
	if t.resp == nil || t.resp.Err != "" {
		return
	}
	c := &a.ctx
	tabsR, _ := r.SplitTop(paneTabsH)
	clr := c.Th.OK
	if t.resp.Status >= 400 {
		clr = c.Th.Error
	}
	status := fmt.Sprintf("%d %s", t.resp.Status, t.resp.StatusText)
	rest := fmt.Sprintf("%s  %s", formatDuration(t.resp.Duration), formatSize(t.resp.Size))
	x := tabsR.X + tabsR.W - c.Th.Pad - c.width(c.F.UI, rest)
	if tabs := c.tabRects(tabsR, a.tabItems(responsePanes, t)); x-12-c.width(c.F.UI, status) < tabs[len(tabs)-1].X+tabs[len(tabs)-1].W {
		return // no room beside the tabs
	}
	c.textMid(dst, c.F.UI, rest, x, tabsR, c.Th.Text)
	x -= 12 + c.width(c.F.UI, status)
	c.textMid(dst, c.F.UI, status, x, tabsR, clr)
}

func (a *App) drawStatusBar(dst *ebiten.Image, r Rect) {
	c := &a.ctx
	c.fill(dst, r, c.Th.Sidebar)
	c.hline(dst, r.X, r.Y, r.W, c.Th.Border)
	c.textMid(dst, c.F.Small, a.status, r.X+c.Th.Pad, r, c.Th.Muted)
	c.textMid(dst, c.F.Small, version, r.X+r.W-c.Th.Pad-c.width(c.F.Small, version), r, c.Th.Muted)
}
