package widget

import (
	"strconv"
	"strings"
	"unicode"

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

	// Text selection. Focused views answer Ctrl+C, Ctrl+A and paging keys.
	Focused    bool
	HasSel     bool
	selA, selB textPos // selection from selA to selB; either may come first
	dragging   bool
	drag       barDrag

	matches []Match // search hits to highlight
	current int     // index of the active hit, -1 for none
	reveal  int     // original line to scroll to on the next update, -1 for none
}

// SetLines replaces the content and resets folds and scroll.
func (v *CodeView) SetLines(lines []Line) {
	v.Lines = lines
	v.Folded = map[int]bool{}
	v.ScrollY = 0
	v.HasSel, v.dragging, v.drag = false, false, barDrag{}
	v.ScrollX, v.maxRunes = 0, 0
	v.matches, v.current, v.reveal = nil, -1, -1
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

// textPos is a position in the text: an original line and a rune column.
type textPos struct{ line, col int }

func (p textPos) before(q textPos) bool {
	return p.line < q.line || (p.line == q.line && p.col < q.col)
}

// lineLen is the number of runes in line i.
func (v *CodeView) lineLen(i int) int { return len([]rune(v.lineText(i))) }

// selRange returns the selection low to high; ok is false when it is empty.
func (v *CodeView) selRange() (lo, hi textPos, ok bool) {
	if !v.HasSel || len(v.Lines) == 0 || v.selA == v.selB {
		return textPos{}, textPos{}, false
	}
	lo, hi = v.selA, v.selB
	if hi.before(lo) {
		lo, hi = hi, lo
	}
	return lo, hi, true
}

// SelectedText is the selected text (lines joined by newlines), or "".
func (v *CodeView) SelectedText() string {
	lo, hi, ok := v.selRange()
	if !ok {
		return ""
	}
	parts := make([]string, 0, hi.line-lo.line+1)
	for i := lo.line; i <= hi.line; i++ {
		runes := []rune(v.lineText(i))
		from, to := 0, len(runes)
		if i == lo.line {
			from = min(lo.col, len(runes))
		}
		if i == hi.line {
			to = min(hi.col, len(runes))
		}
		parts = append(parts, string(runes[from:to]))
	}
	return strings.Join(parts, "\n")
}

// SelectLines selects original lines a through b (either order), whole.
func (v *CodeView) SelectLines(a, b int) {
	last := len(v.Lines) - 1
	if last < 0 {
		v.HasSel = false
		return
	}
	lo, hi := min(max(a, 0), last), min(max(b, 0), last)
	if hi < lo {
		lo, hi = hi, lo
	}
	v.selA, v.selB, v.HasSel = textPos{lo, 0}, textPos{hi, v.lineLen(hi)}, true
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
	if v.reveal >= 0 {
		v.applyReveal(g, b)
		vis = v.Visible()
		b = v.bars(c, r, g, len(vis))
	}
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

// posAt is the text position under a pixel, clamped to the line.
func (v *CodeView) posAt(x, y int, r Rect, g codeGeometry, vis []int) textPos {
	line := v.lineAt(y, r, g, vis)
	col := (x - g.textX + v.ScrollX + g.charW/2) / g.charW
	return textPos{line, min(max(col, 0), v.lineLen(line))}
}

// pressText starts a selection at p: a double click selects the word,
// a triple click the line, a single click starts a drag.
func (v *CodeView) pressText(p textPos, clicks int) {
	runes := []rune(v.lineText(p.line))
	switch {
	case clicks == 2 && len(runes) > 0:
		lo, hi := wordRange(runes, p.col)
		v.selA, v.selB, v.HasSel, v.dragging = textPos{p.line, lo}, textPos{p.line, hi}, true, false
	case clicks >= 3:
		v.selA, v.selB, v.HasSel, v.dragging = textPos{p.line, 0}, textPos{p.line, len(runes)}, true, false
	default:
		v.selA, v.selB, v.HasSel, v.dragging = p, p, true, true
	}
}

// wordRange is the run of word characters, or of other characters, at col.
func wordRange(r []rune, col int) (lo, hi int) {
	at := min(col, len(r)-1)
	word := isWordRune(r[at])
	lo, hi = at, at+1
	for lo > 0 && isWordRune(r[lo-1]) == word {
		lo--
	}
	for hi < len(r) && isWordRune(r[hi]) == word {
		hi++
	}
	return lo, hi
}

func (v *CodeView) updatePointer(c *Ctx, r Rect, g codeGeometry, b bars, vis []int, over bool) {
	in := c.In
	if v.dragging {
		if in.Held {
			v.selB = v.posAt(in.X, in.Y, r, g, vis)
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
		v.pressText(v.posAt(in.X, in.Y, r, g, vis), in.Clicks)
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
		v.drawMatches(c, text, g, cell, idx)
		if lo, hi, ok := v.selRange(); ok && idx >= lo.line && idx <= hi.line {
			v.drawSelection(c, text, g, cell, idx, lo, hi)
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

// Selected reports whether any text is selected.
func (v *CodeView) Selected() bool { _, _, ok := v.selRange(); return ok }

// drawSelection fills the selected part of line idx; lines that continue
// into the next selected line also fill one character for the newline.
func (v *CodeView) drawSelection(c *Ctx, dst *ebiten.Image, g codeGeometry, cell Rect, idx int, lo, hi textPos) {
	from, to := 0, v.lineLen(idx)
	if idx == lo.line {
		from = lo.col
	}
	if idx == hi.line {
		to = hi.col
	} else {
		to++ // the line break
	}
	x0 := g.textX - v.ScrollX + from*g.charW
	x1 := g.textX - v.ScrollX + to*g.charW
	c.Fill(dst, Rect{X: x0, Y: cell.Y, W: max(x1-x0, 0), H: cell.H}, c.Col(Selection))
}

// Match is one search hit: an original line, a rune column and a length.
type Match struct{ Line, Col, Len int }

// Find returns the case-insensitive, non-overlapping matches of query in
// every line (folded ones included), in text order. An empty query has none.
func (v *CodeView) Find(query string) []Match {
	needle := []rune(strings.ToLower(query))
	if len(needle) == 0 {
		return nil
	}
	var found []Match
	for i := range v.Lines {
		hay := []rune(v.lineText(i))
		for col := 0; col+len(needle) <= len(hay); {
			if equalFold(hay[col:col+len(needle)], needle) {
				found = append(found, Match{Line: i, Col: col, Len: len(needle)})
				col += len(needle)
				continue
			}
			col++
		}
	}
	return found
}

// equalFold compares rune by rune, so columns stay aligned for text
// whose lower-casing changes its byte length.
func equalFold(a, b []rune) bool {
	for i := range a {
		if unicode.ToLower(a[i]) != b[i] {
			return false
		}
	}
	return true
}

// SetMatches highlights matches, with the one at index current (or none
// when current is out of range) marked as the active one.
func (v *CodeView) SetMatches(matches []Match, current int) {
	v.matches = matches
	if current < 0 || current >= len(matches) {
		current = -1
	}
	v.current = current
}

// RevealLine scrolls so the original line is in view on the next
// update, unfolding whatever hides it.
func (v *CodeView) RevealLine(line int) { v.reveal = line }

// applyReveal carries out a pending RevealLine.
func (v *CodeView) applyReveal(g codeGeometry, b bars) {
	line := v.reveal
	v.reveal = -1
	if line < 0 || line >= len(v.Lines) {
		return
	}
	row := v.rowOf(line)
	if row < 0 { // hidden by a fold: open the folds, then it is visible
		v.Folded = map[int]bool{}
		row = v.rowOf(line)
	}
	top, bottom := row*g.lineH, (row+1)*g.lineH
	if top < v.ScrollY || bottom > v.ScrollY+b.viewH {
		v.ScrollY = max(top-(b.viewH-g.lineH)/2, 0) // centered
	}
}

// rowOf is the visible row of an original line, -1 when it is folded away.
func (v *CodeView) rowOf(line int) int {
	for row, idx := range v.Visible() {
		if idx == line {
			return row
		}
	}
	return -1
}

// drawMatches fills the search hits on line idx.
func (v *CodeView) drawMatches(c *Ctx, dst *ebiten.Image, g codeGeometry, cell Rect, idx int) {
	for i, m := range v.matches {
		if m.Line != idx {
			continue
		}
		tok := FindMatch
		if i == v.current {
			tok = FindActive
		}
		x := g.textX - v.ScrollX + m.Col*g.charW
		c.Fill(dst, Rect{X: x, Y: cell.Y, W: m.Len * g.charW, H: cell.H}, c.Col(tok))
	}
}

// Matches returns the search hits currently highlighted.
func (v *CodeView) Matches() []Match { return v.matches }
