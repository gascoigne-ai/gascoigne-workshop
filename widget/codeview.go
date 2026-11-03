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
	Lines   []Line
	Folded  map[int]bool // by original line index
	ScrollY int          // pixels

	// Whole-line selection, by original line index. Focused views
	// answer Ctrl+C and Ctrl+A.
	Focused            bool
	HasSel             bool
	selAnchor, selLine int
	dragging           bool
}

// SetLines replaces the content and resets folds and scroll.
func (v *CodeView) SetLines(lines []Line) {
	v.Lines = lines
	v.Folded = map[int]bool{}
	v.ScrollY = 0
	v.HasSel, v.dragging = false, false
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

// lineAt is the original line under pixel row y, clamped to the content.
func (v *CodeView) lineAt(y int, r Rect, g codeGeometry, vis []int) int {
	row := min(max((y-r.Y+v.ScrollY)/g.lineH, 0), len(vis)-1)
	return vis[row]
}

// Update handles wheel scrolling, fold-arrow clicks, line selection by
// click or drag, and Ctrl+A / Ctrl+C while focused.
func (v *CodeView) Update(c *Ctx, r Rect) {
	in := c.In
	over := c.In.Over(r)
	if in.Click {
		v.Focused = over
	}
	g := v.geometry(c, r)
	vis := v.Visible()
	if len(vis) > 0 {
		v.updatePointer(c, r, g, vis, over)
	}
	if !v.Focused {
		return
	}
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
}

func (v *CodeView) updatePointer(c *Ctx, r Rect, g codeGeometry, vis []int, over bool) {
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
		v.Scroll(int(-in.Wheel*float64(g.lineH)*3), len(vis)*g.lineH, r.H)
	}
	if !in.Click {
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

// Draw paints the visible lines inside r; the caller fills the background.
func (v *CodeView) Draw(c *Ctx, dst *ebiten.Image, r Rect) {
	g := v.geometry(c, r)
	vis := v.Visible()
	v.Scroll(0, len(vis)*g.lineH, r.H)
	dst = c.Clip(dst, r)
	muted := c.Col(Muted)

	for row := v.ScrollY / g.lineH; row < len(vis); row++ {
		y := r.Y + row*g.lineH - v.ScrollY
		if y >= r.Y+r.H {
			break
		}
		idx := vis[row]
		line := v.Lines[idx]
		cell := Rect{r.X, y, r.W, g.lineH}
		if lo, hi, ok := v.selected(); ok && idx >= lo && idx <= hi {
			c.Fill(dst, cell, c.Col(Selection))
		}

		num := strconv.Itoa(idx + 1)
		c.TextMid(dst, c.F.Mono, num, r.X+Pad+g.numW-len(num)*g.charW, cell, muted)
		if line.Foldable {
			cx := float32(r.X + Pad + g.numW + 4 + foldColW/2)
			c.Triangle(dst, cx, float32(y+g.lineH/2), 4, !v.Folded[idx], muted)
		}

		x := g.textX
		for _, sp := range line.Spans {
			c.TextMid(dst, c.F.Mono, sp.Text, x, cell, c.Col(sp.Style))
			x += c.Width(c.F.Mono, sp.Text)
		}
		if v.Folded[idx] {
			c.TextMid(dst, c.F.Mono, " ... }", x, cell, muted)
		}
	}
}
