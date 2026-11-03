package widget

import (
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// Span is a run of text colored by a theme token.
type Span struct {
	Text  string
	Style Token
}

// Line is one row of a CodeView. Foldable lines open a block that is
// indented deeper than the line itself.
type Line struct {
	Spans    []Span
	Foldable bool
}

// CodeView is a read-only, scrollable, foldable text viewer with line
// numbers.
type CodeView struct {
	Lines    []Line
	Folded   map[int]bool // by original line index
	ScrollY  int          // pixels
	ScrollX  int          // pixels
	maxRunes int          // runes in the widest line, for the horizontal extent

	// Whole-line selection, by original line index. Focused views
	// answer Ctrl+C and Ctrl+A.
	Focused            bool
	HasSel             bool
	selAnchor, selLine int
	dragging           bool
	drag               barDrag
}

// SetLines replaces the content and resets folds and scroll.
func (v *CodeView) SetLines(lines []Line) {
	v.Lines = lines
	v.Folded = map[int]bool{}
	v.ScrollY = 0
	v.HasSel, v.dragging, v.drag = false, false, barDrag{}
	v.ScrollX, v.maxRunes = 0, 0
	for i := range lines {
		n := 0
		for _, sp := range lines[i].Spans {
			n += len([]rune(sp.Text))
		}
		v.maxRunes = max(v.maxRunes, n)
	}
}

// lineText is the plain text of line i.
func (v *CodeView) lineText(i int) string {
	var b strings.Builder
	for _, s := range v.Lines[i].Spans {
		b.WriteString(s.Text)
	}
	return b.String()
}

// AllText is every line, folded or not, joined by newlines.
func (v *CodeView) AllText() string { return v.textOf(0, len(v.Lines)-1) }

func (v *CodeView) textOf(lo, hi int) string {
	parts := make([]string, 0, hi-lo+1)
	for i := lo; i <= hi && i < len(v.Lines); i++ {
		parts = append(parts, v.lineText(i))
	}
	return strings.Join(parts, "\n")
}

// selected returns the selected line range, low to high.
func (v *CodeView) selected() (lo, hi int, ok bool) {
	if !v.HasSel || len(v.Lines) == 0 {
		return 0, 0, false
	}
	return min(v.selAnchor, v.selLine), max(v.selAnchor, v.selLine), true
}

// SelectedText is the selected lines joined by newlines, or "".
func (v *CodeView) SelectedText() string {
	lo, hi, ok := v.selected()
	if !ok {
		return ""
	}
	return v.textOf(lo, hi)
}

// SelectLines selects original lines a through b (either order).
func (v *CodeView) SelectLines(a, b int) {
	last := len(v.Lines) - 1
	v.selAnchor, v.selLine, v.HasSel = min(max(a, 0), last), min(max(b, 0), last), last >= 0
}

// ToggleFold folds or unfolds the foldable line at original index i.
func (v *CodeView) ToggleFold(i int) {
	if i < 0 || i >= len(v.Lines) || !v.Lines[i].Foldable {
		return
	}
	if v.Folded == nil {
		v.Folded = map[int]bool{}
	}
	v.Folded[i] = !v.Folded[i]
}

// Visible lists the original indexes of the lines to show. A folded
// line hides everything up to and including its closing line.
func (v *CodeView) Visible() []int {
	var out []int
	for i := 0; i < len(v.Lines); i++ {
		out = append(out, i)
		if !v.Folded[i] || !v.Lines[i].Foldable {
			continue
		}
		indent := indentOf(v.Lines[i])
		for i+1 < len(v.Lines) {
			i++
			if indentOf(v.Lines[i]) <= indent {
				break
			}
		}
	}
	return out
}

// indentOf counts the leading spaces of a line.
func indentOf(l Line) int {
	if len(l.Spans) == 0 {
		return 0
	}
	t := l.Spans[0].Text
	return len(t) - len(strings.TrimLeft(t, " "))
}

// Scroll moves the view by dy pixels, keeping it inside the content of
// contentH pixels shown in a viewport of viewH.
func (v *CodeView) Scroll(dy, contentH, viewH int) {
	v.ScrollY = min(max(v.ScrollY+dy, 0), max(contentH-viewH, 0))
}

const foldColW = 18

// codeGeometry is the pixel layout of a CodeView in r.
type codeGeometry struct {
	lineH, charW, numW, textX int
}

func (v *CodeView) geometry(c *Ctx, r Rect) codeGeometry {
	g := codeGeometry{lineH: c.LineH(c.F.Mono) + 4, charW: c.Width(c.F.Mono, "0")}
	g.numW = g.charW * len(strconv.Itoa(max(len(v.Lines), 1)))
	g.textX = r.X + Pad + g.numW + foldColW + 8
	return g
}

// Update handles wheel and scrollbar scrolling, fold-arrow clicks, line
// selection by click or drag, and the keyboard while focused: Ctrl+A,
// Ctrl+C, PageUp/PageDown, Home and End.
func (v *CodeView) Update(c *Ctx, r Rect) {
	in := c.In
	over := in.Over(r)
	if in.Click {
		v.Focused = over
	}
	g := v.geometry(c, r)
	vis := v.Visible()
	b := v.bars(c, r, g, len(vis))
	v.updateBars(c, g, b, len(vis))
	if len(vis) > 0 && !v.drag.active() {
		v.updatePointer(c, r, g, b, vis, over)
	}
	v.clampScroll(c, r, g, b, len(vis))
	if v.Focused {
		v.updateKeys(c, r, g, b, len(vis))
	}
}

func (v *CodeView) clampScroll(c *Ctx, r Rect, g codeGeometry, b bars, visible int) {
	v.ScrollY = ClampScroll(v.ScrollY, visible*g.lineH, b.viewH)
	v.ScrollX = ClampScroll(v.ScrollX, v.contentWidth(c, r), b.viewW)
}

// updateBars handles grabbing, dragging and page-clicking the scrollbars.
func (v *CodeView) updateBars(c *Ctx, g codeGeometry, b bars, visible int) {
	contentW := v.contentWidth(c, Rect{W: b.viewW}) // only the extent matters
	v.drag.update(c, b, contentW, visible*g.lineH, &v.ScrollX, &v.ScrollY)
}

// lineAt is the original line under pixel row y, clamped to the content.
func (v *CodeView) lineAt(y int, r Rect, g codeGeometry, vis []int) int {
	row := min(max((y-r.Y+v.ScrollY)/g.lineH, 0), len(vis)-1)
	return vis[row]
}

func (v *CodeView) updatePointer(c *Ctx, r Rect, g codeGeometry, b bars, vis []int, over bool) {
	in := c.In
	if v.dragging {
		if in.Held {
			v.selLine = v.lineAt(in.Y, r, g, vis)
		} else {
			v.dragging = false
		}
	}
	if !over {
		return
	}
	if in.Wheel != 0 {
		if in.Shift {
			v.ScrollX = ClampScroll(v.ScrollX+int(-in.Wheel*float64(g.charW)*6), v.contentWidth(c, r), b.viewW)
		} else {
			v.Scroll(int(-in.Wheel*float64(g.lineH)*3), len(vis)*g.lineH, b.viewH)
		}
		c.In.Wheel = 0 // consumed, so an enclosing scroller does not move too
	}
	if !in.Click || b.vTrack.Contains(in.X, in.Y) || b.hTrack.Contains(in.X, in.Y) {
		return
	}
	foldCol := Rect{r.X + Pad + g.numW + 4, r.Y, foldColW, r.H}
	line := v.lineAt(in.Y, r, g, vis)
	switch {
	case foldCol.Contains(in.X, in.Y):
		v.ToggleFold(line)
	case in.X >= foldCol.X+foldColW:
		v.selAnchor, v.selLine, v.HasSel, v.dragging = line, line, true, true
	}
}

// updateKeys handles the keyboard for a focused view.
func (v *CodeView) updateKeys(c *Ctx, r Rect, g codeGeometry, b bars, visible int) {
	in := c.In
	contentH := visible * g.lineH
	if in.SelectAll {
		v.SelectLines(0, len(v.Lines)-1)
	}
	if in.Copy && c.Clipboard != nil {
		text := v.SelectedText()
		if text == "" {
			text = v.AllText()
		}
		_ = c.Clipboard.Write(text) // the clipboard reports its own failures
	}
	for _, k := range in.Keys {
		switch k {
		case ebiten.KeyPageDown:
			v.ScrollY = ClampScroll(v.ScrollY+b.viewH-g.lineH, contentH, b.viewH)
		case ebiten.KeyPageUp:
			v.ScrollY = ClampScroll(v.ScrollY-b.viewH+g.lineH, contentH, b.viewH)
		case ebiten.KeyHome:
			v.ScrollY = 0
		case ebiten.KeyEnd:
			v.ScrollY = ClampScroll(contentH, contentH, b.viewH)
		}
	}
}

// Draw paints the visible lines inside r; the caller fills the background.
// Line numbers and fold arrows stay put while the text scrolls sideways.
func (v *CodeView) Draw(c *Ctx, dst *ebiten.Image, r Rect) {
	g := v.geometry(c, r)
	vis := v.Visible()
	b := v.bars(c, r, g, len(vis))
	v.clampScroll(c, r, g, b, len(vis))
	muted := c.Col(Muted)
	gutter := Rect{X: r.X, Y: r.Y, W: g.textX - r.X - 4, H: b.viewH}
	body := Rect{X: gutter.X + gutter.W, Y: r.Y, W: b.viewW - gutter.W, H: b.viewH}
	area := c.Clip(dst, Rect{X: r.X, Y: r.Y, W: b.viewW, H: b.viewH})
	text := c.Clip(dst, body)

	for row := v.ScrollY / g.lineH; row < len(vis); row++ {
		y := r.Y + row*g.lineH - v.ScrollY
		if y >= r.Y+b.viewH {
			break
		}
		idx := vis[row]
		line := v.Lines[idx]
		cell := Rect{r.X, y, b.viewW, g.lineH}
		if lo, hi, ok := v.selected(); ok && idx >= lo && idx <= hi {
			c.Fill(area, cell, c.Col(Selection))
		}

		x := g.textX - v.ScrollX
		for _, sp := range line.Spans {
			c.TextMid(text, c.F.Mono, sp.Text, x, cell, c.Col(sp.Style))
			x += c.Width(c.F.Mono, sp.Text)
		}
		if v.Folded[idx] {
			c.TextMid(text, c.F.Mono, " ... }", x, cell, muted)
		}

		num := strconv.Itoa(idx + 1)
		c.TextMid(area, c.F.Mono, num, r.X+Pad+g.numW-len(num)*g.charW, cell, muted)
		if line.Foldable {
			cx := float32(r.X + Pad + g.numW + 4 + foldColW/2)
			c.Triangle(area, cx, float32(y+g.lineH/2), 4, !v.Folded[idx], muted)
		}
	}
	c.drawBars(dst, b, v.drag, v.contentWidth(c, Rect{W: b.viewW}), len(vis)*g.lineH, v.ScrollX, v.ScrollY)
}

// contentWidth is the width of the widest line plus the gutter.
func (v *CodeView) contentWidth(c *Ctx, r Rect) int {
	g := v.geometry(c, r)
	return g.textX - r.X + v.maxRunes*g.charW + Pad
}

func (v *CodeView) bars(c *Ctx, r Rect, g codeGeometry, visible int) bars {
	return layoutBars(r, v.contentWidth(c, r), visible*g.lineH)
}

// Selected reports whether any line is selected.
func (v *CodeView) Selected() bool { return v.HasSel }
