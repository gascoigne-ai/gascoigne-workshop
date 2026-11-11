package widget

import (
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Line is one row of a CodeView. Foldable lines open a block that is
// indented deeper than the line itself.
type Line struct {
	Spans    []Span
	Foldable bool
	// FoldedSpans, when set, replace Spans while the line is folded, to
	// show a one-line preview of the hidden block.
	FoldedSpans []Span
	Depth       int // nesting level of a block's opening line, for folding by depth
	// Num is the line number shown in the gutter, for views that show only
	// some of a text's lines; 0 numbers the line by its position.
	Num int
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

	// Follow keeps the view at its end as lines are added, like a log;
	// scrolling up by the user pauses it and reaching the end resumes it.
	Follow bool

	// Wrap breaks long lines at the width of the view instead of scrolling
	// sideways. Line numbers, folds, selection and search stay by original
	// line.
	Wrap bool
}

// wrapRow is one visual row: runes from..to of an original line.
type wrapRow struct{ line, from, to int }

// wrapRows lays the visible lines out in rows of at most cols runes;
// without room for even one column (cols < 1) each line is one row.
func wrapRows(vis []int, runesIn func(line int) int, cols int) []wrapRow {
	rows := make([]wrapRow, 0, len(vis))
	for _, line := range vis {
		n := runesIn(line)
		if cols < 1 || n <= cols {
			rows = append(rows, wrapRow{line, 0, n})
			continue
		}
		for from := 0; from < n; from += cols {
			rows = append(rows, wrapRow{line, from, min(from+cols, n)})
		}
	}
	return rows
}

// wrapCols is how many columns of text fit in the view, 0 when the view
// does not wrap.
func (v *CodeView) wrapCols(g codeGeometry, r core.Rect) int {
	if !v.Wrap {
		return 0
	}
	// The vertical bar is assumed present, so the columns do not change
	// when wrapping makes the content long enough to need it.
	room := r.W - scrollbarW - (g.textX - r.X) - core.Gutter
	return max(int(float64(room)/g.adv), 1)
}

// rows are the visual rows of the visible lines. A folded line is one row
// that is drawn whole.
func (v *CodeView) rows(g codeGeometry, r core.Rect, vis []int) []wrapRow {
	return wrapRows(vis, func(line int) int {
		if v.Folded[line] && v.Lines[line].Foldable {
			return 0
		}
		return v.lineLen(line)
	}, v.wrapCols(g, r))
}

// end is the rune column the row stops at: a row with no end of its own
// (a line drawn whole, such as a folded one) runs to the end of its line.
func (row wrapRow) end(lineLen int) int {
	if row.to == 0 {
		return lineLen
	}
	return min(row.to, lineLen)
}

// whole reports whether the row is all of its line, drawn without cutting.
func (row wrapRow) whole(lineLen int) bool {
	return row.from == 0 && (row.to >= lineLen || row.to == 0)
}

// cut is the part of spans from rune from to rune to.
func cutSpans(spans []Span, from, to int) []Span {
	var out []Span
	pos := 0
	for _, sp := range spans {
		runes := []rune(sp.Text)
		lo, hi := max(from-pos, 0), min(to-pos, len(runes))
		if lo < hi {
			out = append(out, Span{Text: string(runes[lo:hi]), Style: sp.Style})
		}
		pos += len(runes)
	}
	return out
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

// foldedMore is what a folded block without a preview of its own shows
// after its first line.
const foldedMore = " ... }"

// shownSpans are the spans line i is drawn with: the preview of a folded
// block, else the line itself.
func (v *CodeView) shownSpans(i int) []Span {
	line := v.Lines[i]
	if !v.Folded[i] || !line.Foldable {
		return line.Spans
	}
	if len(line.FoldedSpans) > 0 {
		return line.FoldedSpans
	}
	return append(append([]Span(nil), line.Spans...), Span{Text: foldedMore, Style: core.Muted})
}

// lineText is the plain text of line i as it is shown, so that columns,
// words and copied text follow what the user sees.
func (v *CodeView) lineText(i int) string {
	var b strings.Builder
	for _, s := range v.shownSpans(i) {
		b.WriteString(s.Text)
	}
	return b.String()
}

// rawText is the text of line i itself, whether or not it is folded.
func (v *CodeView) rawText(i int) string {
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
		parts = append(parts, v.rawText(i))
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
	last := len(v.Lines) - 1
	if lo == (textPos{}) && hi.line == last && hi.col >= v.lineLen(last) {
		return v.AllText() // everything is selected: copy it all, not the folds' previews
	}
	parts := make([]string, 0, hi.line-lo.line+1)
	for _, i := range v.Visible() {
		if i < lo.line || i > hi.line {
			continue
		}
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

// LineNumber is the number shown for line i: its Num, else its position.
func (v *CodeView) LineNumber(i int) int {
	if i >= 0 && i < len(v.Lines) && v.Lines[i].Num > 0 {
		return v.Lines[i].Num
	}
	return i + 1
}

// numberDigits is how many digits the widest line number has.
func (v *CodeView) numberDigits() int {
	widest := max(len(v.Lines), 1)
	for i := range v.Lines {
		widest = max(widest, v.LineNumber(i))
	}
	return len(strconv.Itoa(widest))
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

// ScrollBy moves the view by dy pixels, keeping it inside the content of
// contentH pixels shown in a viewport of viewH.
func (v *CodeView) ScrollBy(dy, contentH, viewH int) {
	v.ScrollY = min(max(v.ScrollY+dy, 0), max(contentH-viewH, 0))
}

const foldColW = 18

// codeGeometry is the pixel layout of a CodeView in r.
type codeGeometry struct {
	lineH, charW, numW, textX int
	adv                       float64 // the exact advance of one character
}

// colX is the pixel offset of column col from the start of the text, by the
// exact advance, so the caret, selection and matches line up with the text
// however long the line.
func (g codeGeometry) colX(col int) int { return int(float64(col)*g.adv + 0.5) }

// colAt is the column nearest to offset x from the start of the text.
func (g codeGeometry) colAt(x int) int { return int((float64(x) + g.adv/2) / g.adv) }

func (v *CodeView) geometry(c *core.Ctx, r core.Rect) codeGeometry {
	g := codeGeometry{lineH: c.LineH(c.F.Mono) + 4, charW: c.Width(c.F.Mono, "0"), adv: c.Advance(c.F.Mono)}
	g.numW = g.charW * v.numberDigits()
	g.textX = r.X + core.Gutter + g.numW + foldColW + 8
	return g
}

// Update handles wheel and scrollbar scrolling, fold-arrow clicks, line
// selection by click or drag, and the keyboard while focused: Ctrl+A,
// Ctrl+C, PageUp/PageDown, Home and End.
func (v *CodeView) Update(c *core.Ctx, r core.Rect) {
	in := c.In
	over := in.Over(r)
	if in.Click {
		v.Focused = over
	}
	prevY := v.ScrollY
	g := v.geometry(c, r)
	rows := v.rows(g, r, v.Visible())
	b := v.bars(c, r, g, len(rows))
	v.updateBars(c, g, b, len(rows))
	if v.reveal >= 0 {
		v.applyReveal(r, g, b, rows)
		rows = v.rows(g, r, v.Visible())
		b = v.bars(c, r, g, len(rows))
	}
	if len(rows) > 0 && !v.drag.active() {
		v.updatePointer(c, r, g, b, rows, over)
	}
	v.clampScroll(c, r, g, b, len(rows))
	if v.Focused {
		v.updateKeys(c, r, g, b, len(rows))
	}
	v.follow(prevY, len(rows)*g.lineH, b.viewH)
}

// follow pins a Follow view to its end, unless the user just moved it.
func (v *CodeView) follow(prevY, contentH, viewH int) {
	end := max(contentH-viewH, 0)
	if v.ScrollY != prevY { // moved by the user: following resumes at the end only
		v.Follow = v.ScrollY >= end
		return
	}
	if v.Follow {
		v.ScrollY = end
	}
}

func (v *CodeView) clampScroll(c *core.Ctx, r core.Rect, g codeGeometry, b bars, visible int) {
	v.ScrollY = clampScroll(v.ScrollY, visible*g.lineH, b.viewH)
	v.ScrollX = clampScroll(v.ScrollX, v.contentWidth(c, r), b.viewW)
}

// updateBars handles grabbing, dragging and page-clicking the scrollbars.
func (v *CodeView) updateBars(c *core.Ctx, g codeGeometry, b bars, visible int) {
	contentW := v.contentWidth(c, core.Rect{W: b.viewW}) // only the extent matters
	v.drag.update(c, b, contentW, visible*g.lineH, &v.ScrollX, &v.ScrollY)
}

// rowAt is the visual row under pixel row y, clamped to the content.
func (v *CodeView) rowAt(y int, r core.Rect, g codeGeometry, rows []wrapRow) wrapRow {
	return rows[min(max((y-r.Y+v.ScrollY)/g.lineH, 0), len(rows)-1)]
}

// posAt is the text position under a pixel, clamped to the row's part of
// its line.
func (v *CodeView) posAt(x, y int, r core.Rect, g codeGeometry, rows []wrapRow) textPos {
	row := v.rowAt(y, r, g, rows)
	col := row.from + g.colAt(x-g.textX+v.ScrollX)
	if x-g.textX+v.ScrollX < -g.charW/2 {
		col = row.from
	}
	return textPos{row.line, min(max(col, row.from), row.end(v.lineLen(row.line)))}
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

func (v *CodeView) updatePointer(c *core.Ctx, r core.Rect, g codeGeometry, b bars, rows []wrapRow, over bool) {
	in := c.In
	if v.dragging {
		if in.Held {
			v.selB = v.posAt(in.X, in.Y, r, g, rows)
		} else {
			v.dragging = false
		}
	}
	if !over {
		return
	}
	if in.Wheel != 0 {
		if in.Shift {
			v.ScrollX = clampScroll(v.ScrollX+int(-in.Wheel*float64(g.charW)*6), v.contentWidth(c, r), b.viewW)
		} else {
			v.ScrollBy(int(-in.Wheel*float64(g.lineH)*3), len(rows)*g.lineH, b.viewH)
		}
		c.In.Wheel = 0 // consumed, so an enclosing scroller does not move too
	}
	if !in.Click || b.vTrack.Contains(in.X, in.Y) || b.hTrack.Contains(in.X, in.Y) {
		return
	}
	foldCol := core.Rect{X: r.X + core.Gutter + g.numW + 4, Y: r.Y, W: foldColW, H: r.H}
	switch {
	case foldCol.Contains(in.X, in.Y):
		v.ToggleFold(v.rowAt(in.Y, r, g, rows).line)
	case in.X >= foldCol.X+foldColW:
		v.pressText(v.posAt(in.X, in.Y, r, g, rows), in.Clicks)
	}
}

// updateKeys handles the keyboard for a focused view.
func (v *CodeView) updateKeys(c *core.Ctx, r core.Rect, g codeGeometry, b bars, rowCount int) {
	in := c.In
	contentH := rowCount * g.lineH
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
	if in.Shift {
		for _, edit := range in.Edits {
			v.extendSelection(edit)
		}
	}
	for _, k := range in.Keys {
		switch k {
		case ebiten.KeyPageDown:
			v.ScrollY = clampScroll(v.ScrollY+b.viewH-g.lineH, contentH, b.viewH)
		case ebiten.KeyPageUp:
			v.ScrollY = clampScroll(v.ScrollY-b.viewH+g.lineH, contentH, b.viewH)
		case ebiten.KeyHome:
			v.ScrollY = 0
		case ebiten.KeyEnd:
			v.ScrollY = clampScroll(contentH, contentH, b.viewH)
		}
	}
}

// Draw paints the visible lines inside r; the caller fills the background.
// Line numbers and fold arrows stay put while the text scrolls sideways.
func (v *CodeView) Draw(c *core.Ctx, dst *ebiten.Image, r core.Rect) {
	g := v.geometry(c, r)
	rows := v.rows(g, r, v.Visible())
	b := v.bars(c, r, g, len(rows))
	v.clampScroll(c, r, g, b, len(rows))
	muted := c.Color(core.Muted)
	gutter := core.Rect{X: r.X, Y: r.Y, W: g.textX - r.X - 4, H: b.viewH}
	body := core.Rect{X: gutter.X + gutter.W, Y: r.Y, W: b.viewW - gutter.W, H: b.viewH}
	area := c.Clip(dst, core.Rect{X: r.X, Y: r.Y, W: b.viewW, H: b.viewH})
	text := c.Clip(dst, body)

	for rowIdx := v.ScrollY / g.lineH; rowIdx < len(rows); rowIdx++ {
		y := r.Y + rowIdx*g.lineH - v.ScrollY
		if y >= r.Y+b.viewH {
			break
		}
		row := rows[rowIdx]
		idx := row.line
		line := v.Lines[idx]
		cell := core.Rect{X: r.X, Y: y, W: b.viewW, H: g.lineH}
		v.drawMatches(c, text, g, cell, row)
		if lo, hi, ok := v.selRange(); ok && idx >= lo.line && idx <= hi.line {
			v.drawSelection(c, text, g, cell, row, lo, hi)
		}

		spans := v.shownSpans(idx)
		if !row.whole(v.lineLen(idx)) {
			spans = cutSpans(spans, row.from, row.to)
		}
		col := 0 // spans are placed by column, so rounding does not add up along the line
		for _, sp := range spans {
			c.TextMid(text, c.F.Mono, sp.Text, g.textX-v.ScrollX+g.colX(col), cell, c.Color(sp.Style))
			col += len([]rune(sp.Text))
		}

		if row.from > 0 { // a continuation row has no number and no fold arrow
			continue
		}
		num := strconv.Itoa(v.LineNumber(idx))
		c.TextMid(area, c.F.Mono, num, r.X+core.Gutter+g.numW-len(num)*g.charW, cell, muted)
		if line.Foldable {
			cx := float32(r.X + core.Gutter + g.numW + 4 + foldColW/2)
			c.Triangle(area, cx, float32(y+g.lineH/2), 4, !v.Folded[idx], muted)
		}
	}
	drawBars(c, dst, b, v.drag, v.contentWidth(c, core.Rect{W: b.viewW}), len(rows)*g.lineH, v.ScrollX, v.ScrollY)
}

// contentWidth is the width of the widest line plus the gutter.
func (v *CodeView) contentWidth(c *core.Ctx, r core.Rect) int {
	if v.Wrap {
		return 0 // wrapped text fits the width
	}
	g := v.geometry(c, r)
	return g.textX - r.X + g.colX(v.maxRunes) + core.Gutter
}

func (v *CodeView) bars(c *core.Ctx, r core.Rect, g codeGeometry, visible int) bars {
	return layoutBars(r, v.contentWidth(c, r), visible*g.lineH)
}

// Selected reports whether any text is selected.
func (v *CodeView) Selected() bool { _, _, ok := v.selRange(); return ok }

// drawSelection fills the selected part of line idx; lines that continue
// into the next selected line also fill one character for the newline.
func (v *CodeView) drawSelection(c *core.Ctx, dst *ebiten.Image, g codeGeometry, cell core.Rect, row wrapRow, lo, hi textPos) {
	idx := row.line
	from, to := row.from, row.end(v.lineLen(idx))
	if idx == lo.line {
		from = max(from, lo.col)
	}
	last := to >= v.lineLen(idx) // the row ends its line
	if idx == hi.line {
		to = min(to, hi.col)
	} else if last {
		to++ // the line break
	}
	if to < from {
		return
	}
	x0 := g.textX - v.ScrollX + g.colX(from-row.from)
	x1 := g.textX - v.ScrollX + g.colX(to-row.from)
	c.Fill(dst, core.Rect{X: x0, Y: cell.Y, W: max(x1-x0, 0), H: cell.H}, c.Color(core.Selection))
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
		hay := []rune(v.rawText(i))
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
func (v *CodeView) applyReveal(r core.Rect, g codeGeometry, b bars, rows []wrapRow) {
	line := v.reveal
	v.reveal = -1
	if line < 0 || line >= len(v.Lines) {
		return
	}
	row := rowOf(rows, line)
	if row < 0 { // hidden by a fold: open the folds, then it is visible
		v.Folded = map[int]bool{}
		row = rowOf(v.rows(g, r, v.Visible()), line)
	}
	top, bottom := row*g.lineH, (row+1)*g.lineH
	if top < v.ScrollY || bottom > v.ScrollY+b.viewH {
		v.ScrollY = max(top-(b.viewH-g.lineH)/2, 0) // centered
	}
}

// rowOf is the first visual row of an original line, -1 when it is
// folded away.
func rowOf(rows []wrapRow, line int) int {
	for i, row := range rows {
		if row.line == line {
			return i
		}
	}
	return -1
}

// drawMatches fills the search hits on line idx.
func (v *CodeView) drawMatches(c *core.Ctx, dst *ebiten.Image, g codeGeometry, cell core.Rect, row wrapRow) {
	for i, m := range v.matches {
		from, to := max(m.Col, row.from), min(m.Col+m.Len, row.to)
		if m.Line != row.line || to <= from {
			continue
		}
		tok := core.FindMatch
		if i == v.current {
			tok = core.FindActive
		}
		x := g.textX - v.ScrollX + g.colX(from-row.from)
		c.Fill(dst, core.Rect{X: x, Y: cell.Y, W: g.colX(to-row.from) - g.colX(from-row.from), H: cell.H}, c.Color(tok))
	}
}

// Matches returns the search hits currently highlighted.
func (v *CodeView) Matches() []Match { return v.matches }

// extendSelection moves the end of the selection for one movement key
// held with Shift. The caret is the end the pointer last set; a view with
// none starts at its first visible line. Folded lines are skipped, and
// the view scrolls to keep the caret in sight.
func (v *CodeView) extendSelection(edit core.Edit) {
	visible := v.Visible()
	if len(visible) == 0 {
		return
	}
	if !v.HasSel {
		v.selA, v.selB, v.HasSel = textPos{visible[0], 0}, textPos{visible[0], 0}, true
	}
	v.selB = v.moveCaret(v.selB, edit, visible)
	v.RevealLine(v.selB.line)
}

// moveCaret is pos after one movement key, staying on the visible lines.
func (v *CodeView) moveCaret(pos textPos, edit core.Edit, visible []int) textPos {
	row := sort.SearchInts(visible, pos.line)
	row = min(row, len(visible)-1)
	pos.line = visible[row]
	atLine := func(target int) textPos {
		return textPos{visible[target], min(pos.col, v.lineLen(visible[target]))}
	}
	switch edit {
	case core.EditLeft:
		if pos.col > 0 {
			return textPos{pos.line, pos.col - 1}
		}
		if row > 0 {
			return textPos{visible[row-1], v.lineLen(visible[row-1])}
		}
	case core.EditRight:
		if pos.col < v.lineLen(pos.line) {
			return textPos{pos.line, pos.col + 1}
		}
		if row+1 < len(visible) {
			return textPos{visible[row+1], 0}
		}
	case core.EditUp:
		if row > 0 {
			return atLine(row - 1)
		}
		return textPos{pos.line, 0}
	case core.EditDown:
		if row+1 < len(visible) {
			return atLine(row + 1)
		}
		return textPos{pos.line, v.lineLen(pos.line)}
	case core.EditHome:
		return textPos{pos.line, 0}
	case core.EditEnd:
		return textPos{pos.line, v.lineLen(pos.line)}
	}
	return pos
}

func (v *CodeView) HasFocus() bool   { return v.Focused }
func (v *CodeView) SetFocus(on bool) { v.Focused = on }
