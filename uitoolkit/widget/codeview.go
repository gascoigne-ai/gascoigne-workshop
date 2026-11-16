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
	reveal  textPos // position to scroll to on the next update, line -1 for none

	// Follow keeps the view at its end as lines are added, like a log;
	// scrolling up by the user pauses it and reaching the end resumes it.
	Follow bool

	// Wrap breaks long lines before a word at the width of the view
	// instead of scrolling sideways. Line numbers, folds, selection and
	// search stay by original line. Code's Wrap option sets it every frame.
	Wrap bool
	// laidWrap is the Wrap the scroll position was last laid out with, so
	// switching it can keep the top line in place.
	laidWrap bool
}

// noReveal is the reveal of a view with nothing to scroll to.
var noReveal = textPos{line: -1}

// wrapWidth is the room in pixels for the text of a row, 0 when the view
// does not wrap.
func (v *CodeView) wrapWidth(geometry codeGeometry, rect core.Rect) float64 {
	return widthFor(v.Wrap, geometry, rect)
}

// widthFor is wrapWidth for a view that does or does not wrap.
func widthFor(wrap bool, geometry codeGeometry, rect core.Rect) float64 {
	if !wrap {
		return 0
	}
	// The vertical bar is assumed present, so the rows do not change when
	// wrapping makes the content long enough to need it. At least one pixel,
	// since a width of 0 would mean no wrap.
	room := rect.W - scrollbarW - (geometry.textX - rect.X) - core.Gutter
	return float64(max(room, 1))
}

// layout is the visual rows of the visible lines. A folded line is one row
// that is drawn whole. Every rune is one advance wide, as drawing places
// text by column.
func (v *CodeView) layout(geometry codeGeometry, rect core.Rect) wrapLayout {
	return v.layoutAt(v.wrapWidth(geometry, rect), geometry)
}

func (v *CodeView) layoutAt(width float64, geometry codeGeometry) wrapLayout {
	visible := v.Visible()
	lines := make([]wrapLine, len(visible))
	for i, line := range visible {
		lines[i] = wrapLine{line: line, text: []rune(v.lineText(line)), whole: v.folded(line)}
	}
	return layoutWrap(lines, width, func(rune) float64 { return geometry.adv })
}

// folded reports whether line i is shown folded, as one row of its preview.
func (v *CodeView) folded(index int) bool { return v.Folded[index] && v.Lines[index].Foldable }

// keepTopLine keeps the line at the top of the view in place when Wrap
// has changed since the last layout, as the number of rows above it does.
func (v *CodeView) keepTopLine(geometry codeGeometry, rect core.Rect) {
	if v.Wrap == v.laidWrap {
		return
	}
	before := v.layoutAt(widthFor(v.laidWrap, geometry, rect), geometry)
	top := before.rowStart(v.ScrollY / geometry.lineH)
	v.laidWrap = v.Wrap
	if row := v.layout(geometry, rect).rowOf(top); row >= 0 {
		v.ScrollY = row * geometry.lineH
	}
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
	v.matches, v.current, v.reveal = nil, -1, noReveal
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
func (v *CodeView) shownSpans(index int) []Span {
	line := v.Lines[index]
	if !v.Folded[index] || !line.Foldable {
		return line.Spans
	}
	if len(line.FoldedSpans) > 0 {
		return line.FoldedSpans
	}
	return append(append([]Span(nil), line.Spans...), Span{Text: foldedMore, Style: core.Muted})
}

// lineText is the plain text of line i as it is shown, so that columns,
// words and copied text follow what the user sees.
func (v *CodeView) lineText(index int) string {
	var b strings.Builder
	for _, s := range v.shownSpans(index) {
		b.WriteString(s.Text)
	}
	return b.String()
}

// rawText is the text of line i itself, whether or not it is folded.
func (v *CodeView) rawText(index int) string {
	var b strings.Builder
	for _, s := range v.Lines[index].Spans {
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

func (p textPos) before(textPosValue textPos) bool {
	return p.line < textPosValue.line || (p.line == textPosValue.line && p.col < textPosValue.col)
}

// lineLen is the number of runes in line i.
func (v *CodeView) lineLen(index int) int { return len([]rune(v.lineText(index))) }

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
func (v *CodeView) SelectLines(value, intValue int) {
	last := len(v.Lines) - 1
	if last < 0 {
		v.HasSel = false
		return
	}
	lo, hi := min(max(value, 0), last), min(max(intValue, 0), last)
	if hi < lo {
		lo, hi = hi, lo
	}
	v.selA, v.selB, v.HasSel = textPos{lo, 0}, textPos{hi, v.lineLen(hi)}, true
}

// LineNumber is the number shown for line i: its Num, else its position.
func (v *CodeView) LineNumber(index int) int {
	if index >= 0 && index < len(v.Lines) && v.Lines[index].Num > 0 {
		return v.Lines[index].Num
	}
	return index + 1
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
func (v *CodeView) ToggleFold(index int) {
	if index < 0 || index >= len(v.Lines) || !v.Lines[index].Foldable {
		return
	}
	if v.Folded == nil {
		v.Folded = map[int]bool{}
	}
	v.Folded[index] = !v.Folded[index]
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
func indentOf(line Line) int {
	if len(line.Spans) == 0 {
		return 0
	}
	t := line.Spans[0].Text
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
func (g codeGeometry) colAt(posX int) int { return int((float64(posX) + g.adv/2) / g.adv) }

func (v *CodeView) geometry(ctx *core.Ctx, rect core.Rect) codeGeometry {
	g := codeGeometry{lineH: ctx.LineH(ctx.F.Mono) + 4, charW: ctx.Width(ctx.F.Mono, "0"), adv: ctx.Advance(ctx.F.Mono)}
	g.numW = g.charW * v.numberDigits()
	g.textX = rect.X + core.Gutter + g.numW + foldColW + 8
	return g
}

// Update handles wheel and scrollbar scrolling, fold-arrow clicks, line
// selection by click or drag, and the keyboard while focused: Ctrl+A,
// Ctrl+C, PageUp/PageDown, Home and End.
func (v *CodeView) Update(ctx *core.Ctx, rect core.Rect) {
	in := ctx.In
	over := in.Over(rect)
	if in.Click {
		v.Focused = over
	}
	g := v.geometry(ctx, rect)
	v.keepTopLine(g, rect)
	prevY := v.ScrollY
	layout := v.layout(g, rect)
	b := v.bars(ctx, rect, g, len(layout.rows))
	v.updateBars(ctx, g, b, len(layout.rows))
	if v.reveal.line >= 0 {
		v.applyReveal(rect, g, b, layout)
		layout = v.layout(g, rect)
		b = v.bars(ctx, rect, g, len(layout.rows))
	}
	if len(layout.rows) > 0 && !v.drag.active() {
		v.updatePointer(ctx, rect, g, b, layout, over)
	}
	v.clampScroll(ctx, rect, g, b, len(layout.rows))
	if v.Focused {
		v.updateKeys(ctx, g, b, layout)
	}
	v.follow(prevY, len(layout.rows)*g.lineH, b.viewH)
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

func (v *CodeView) clampScroll(ctx *core.Ctx, rect core.Rect, geometry codeGeometry, sizes bars, visible int) {
	v.ScrollY = clampScroll(v.ScrollY, visible*geometry.lineH, sizes.viewH)
	v.ScrollX = clampScroll(v.ScrollX, v.contentWidth(ctx, rect), sizes.viewW)
}

// updateBars handles grabbing, dragging and page-clicking the scrollbars.
func (v *CodeView) updateBars(ctx *core.Ctx, geometry codeGeometry, sizes bars, visible int) {
	contentW := v.contentWidth(ctx, core.Rect{W: sizes.viewW}) // only the extent matters
	v.drag.update(ctx, sizes, contentW, visible*geometry.lineH, &v.ScrollX, &v.ScrollY)
}

// rowAt is the index of the visual row under pixel row y, clamped to the
// content.
func (v *CodeView) rowAt(posY int, rect core.Rect, geometry codeGeometry, layout wrapLayout) int {
	return min(max((posY-rect.Y+v.ScrollY)/geometry.lineH, 0), len(layout.rows)-1)
}

// posAt is the text position under a pixel, clamped to the row's part of
// its line.
func (v *CodeView) posAt(posX, posY int, rect core.Rect, geometry codeGeometry, layout wrapLayout) textPos {
	return layout.posAt(v.rowAt(posY, rect, geometry, layout), float64(posX-geometry.textX+v.ScrollX))
}

// pressText starts a selection at p: a double click selects the word,
// a triple click the line, a single click starts a drag.
func (v *CodeView) pressText(textPosValue textPos, clicks int) {
	runes := []rune(v.lineText(textPosValue.line))
	switch {
	case clicks == 2 && len(runes) > 0:
		lo, hi := wordRange(runes, textPosValue.col)
		v.selA, v.selB, v.HasSel, v.dragging = textPos{textPosValue.line, lo}, textPos{textPosValue.line, hi}, true, false
	case clicks >= 3:
		v.selA, v.selB, v.HasSel, v.dragging = textPos{textPosValue.line, 0}, textPos{textPosValue.line, len(runes)}, true, false
	default:
		v.selA, v.selB, v.HasSel, v.dragging = textPosValue, textPosValue, true, true
	}
}

// wordRange is the run of word characters, or of other characters, at col.
func wordRange(runes []rune, col int) (lo, hi int) {
	at := min(col, len(runes)-1)
	word := isWordRune(runes[at])
	lo, hi = at, at+1
	for lo > 0 && isWordRune(runes[lo-1]) == word {
		lo--
	}
	for hi < len(runes) && isWordRune(runes[hi]) == word {
		hi++
	}
	return lo, hi
}

func (v *CodeView) updatePointer(ctx *core.Ctx, rect core.Rect, geometry codeGeometry, sizes bars, layout wrapLayout, over bool) {
	in := ctx.In
	rowCount := len(layout.rows)
	if v.dragging {
		if in.Held {
			v.selB = v.posAt(in.X, in.Y, rect, geometry, layout)
		} else {
			v.dragging = false
		}
	}
	if !over {
		return
	}
	if in.Wheel != 0 {
		if in.Shift {
			v.ScrollX = clampScroll(v.ScrollX+int(-in.Wheel*float64(geometry.charW)*6), v.contentWidth(ctx, rect), sizes.viewW)
		} else {
			v.ScrollBy(int(-in.Wheel*float64(geometry.lineH)*3), rowCount*geometry.lineH, sizes.viewH)
		}
		ctx.In.Wheel = 0 // consumed, so an enclosing scroller does not move too
	}
	if !in.Click || sizes.vTrack.Contains(in.X, in.Y) || sizes.hTrack.Contains(in.X, in.Y) {
		return
	}
	foldCol := core.Rect{X: rect.X + core.Gutter + geometry.numW + 4, Y: rect.Y, W: foldColW, H: rect.H}
	switch {
	case foldCol.Contains(in.X, in.Y):
		v.ToggleFold(layout.rows[v.rowAt(in.Y, rect, geometry, layout)].line)
	case in.X >= foldCol.X+foldColW:
		v.pressText(v.posAt(in.X, in.Y, rect, geometry, layout), in.Clicks)
	}
}

// updateKeys handles the keyboard for a focused view.
func (v *CodeView) updateKeys(ctx *core.Ctx, geometry codeGeometry, sizes bars, layout wrapLayout) {
	in := ctx.In
	contentH := len(layout.rows) * geometry.lineH
	if in.SelectAll {
		v.SelectLines(0, len(v.Lines)-1)
	}
	if in.Copy && ctx.Clipboard != nil {
		text := v.SelectedText()
		if text == "" {
			text = v.AllText()
		}
		_ = ctx.Clipboard.Write(text) // the clipboard reports its own failures
	}
	if in.Shift {
		for _, edit := range in.Edits {
			v.extendSelection(edit, layout)
		}
	}
	for _, k := range in.Keys {
		switch k {
		case ebiten.KeyPageDown:
			v.ScrollY = clampScroll(v.ScrollY+sizes.viewH-geometry.lineH, contentH, sizes.viewH)
		case ebiten.KeyPageUp:
			v.ScrollY = clampScroll(v.ScrollY-sizes.viewH+geometry.lineH, contentH, sizes.viewH)
		case ebiten.KeyHome:
			v.ScrollY = 0
		case ebiten.KeyEnd:
			v.ScrollY = clampScroll(contentH, contentH, sizes.viewH)
		}
	}
}

// Draw paints the visible lines inside r; the caller fills the background.
// Line numbers and fold arrows stay put while the text scrolls sideways.
func (v *CodeView) Draw(ctx *core.Ctx, dst *ebiten.Image, rect core.Rect) {
	g := v.geometry(ctx, rect)
	rows := v.layout(g, rect).rows
	b := v.bars(ctx, rect, g, len(rows))
	v.clampScroll(ctx, rect, g, b, len(rows))
	muted := ctx.Color(core.Muted)
	gutter := core.Rect{X: rect.X, Y: rect.Y, W: g.textX - rect.X - 4, H: b.viewH}
	body := core.Rect{X: gutter.X + gutter.W, Y: rect.Y, W: b.viewW - gutter.W, H: b.viewH}
	area := ctx.Clip(dst, core.Rect{X: rect.X, Y: rect.Y, W: b.viewW, H: b.viewH})
	text := ctx.Clip(dst, body)

	for rowIdx := v.ScrollY / g.lineH; rowIdx < len(rows); rowIdx++ {
		y := rect.Y + rowIdx*g.lineH - v.ScrollY
		if y >= rect.Y+b.viewH {
			break
		}
		row := rows[rowIdx]
		idx := row.line
		line := v.Lines[idx]
		cell := core.Rect{X: rect.X, Y: y, W: b.viewW, H: g.lineH}
		v.drawMatches(ctx, text, g, cell, row)
		if lo, hi, ok := v.selRange(); ok && idx >= lo.line && idx <= hi.line {
			v.drawSelection(ctx, text, g, cell, row, lo, hi)
		}

		spans := v.shownSpans(idx)
		if row.from > 0 || row.to < v.lineLen(idx) {
			spans = cutSpans(spans, row.from, row.to)
		}
		col := 0 // spans are placed by column, so rounding does not add up along the line
		for _, sp := range spans {
			ctx.TextMid(text, ctx.F.Mono, sp.Text, g.textX-v.ScrollX+g.colX(col), cell, ctx.Color(sp.Style))
			col += len([]rune(sp.Text))
		}

		if row.from > 0 { // a continuation row has no number and no fold arrow
			continue
		}
		num := strconv.Itoa(v.LineNumber(idx))
		ctx.TextMid(area, ctx.F.Mono, num, rect.X+core.Gutter+g.numW-len(num)*g.charW, cell, muted)
		if line.Foldable {
			cx := float32(rect.X + core.Gutter + g.numW + 4 + foldColW/2)
			ctx.Triangle(area, cx, float32(y+g.lineH/2), 4, !v.Folded[idx], muted)
		}
	}
	drawBars(ctx, dst, b, v.drag, v.contentWidth(ctx, core.Rect{W: b.viewW}), len(rows)*g.lineH, v.ScrollX, v.ScrollY)
}

// contentWidth is the width of the widest line plus the gutter.
func (v *CodeView) contentWidth(ctx *core.Ctx, rect core.Rect) int {
	if v.Wrap {
		return 0 // wrapped text fits the width
	}
	g := v.geometry(ctx, rect)
	return g.textX - rect.X + g.colX(v.maxRunes) + core.Gutter
}

func (v *CodeView) bars(ctx *core.Ctx, rect core.Rect, geometry codeGeometry, visible int) bars {
	return layoutBars(rect, v.contentWidth(ctx, rect), visible*geometry.lineH)
}

// Selected reports whether any text is selected.
func (v *CodeView) Selected() bool { _, _, ok := v.selRange(); return ok }

// drawSelection fills the selected part of line idx; lines that continue
// into the next selected line also fill one character for the newline.
func (v *CodeView) drawSelection(ctx *core.Ctx, dst *ebiten.Image, geometry codeGeometry, cell core.Rect, row visualRow, lo, hi textPos) {
	idx := row.line
	from, to := row.from, row.to
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
	x0 := geometry.textX - v.ScrollX + geometry.colX(from-row.from)
	x1 := geometry.textX - v.ScrollX + geometry.colX(to-row.from)
	ctx.Fill(dst, core.Rect{X: x0, Y: cell.Y, W: max(x1-x0, 0), H: cell.H}, ctx.Color(core.Selection))
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
func equalFold(runes, other []rune) bool {
	for i := range runes {
		if unicode.ToLower(runes[i]) != other[i] {
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
func (v *CodeView) RevealLine(line int) { v.reveal = textPos{line, 0} }

// RevealMatch scrolls so the start of the hit is in view on the next
// update, unfolding whatever hides it: on a wrapped line that is the row
// the hit is on, which can be far below the line's first row.
func (v *CodeView) RevealMatch(hit Match) { v.reveal = textPos{hit.Line, hit.Col} }

// applyReveal carries out a pending RevealLine or RevealMatch.
func (v *CodeView) applyReveal(rect core.Rect, geometry codeGeometry, sizes bars, layout wrapLayout) {
	pos := v.reveal
	v.reveal = noReveal
	if pos.line < 0 || pos.line >= len(v.Lines) {
		return
	}
	row := layout.rowOf(pos)
	if row < 0 { // hidden by a fold: open the folds, then it is visible
		v.Folded = map[int]bool{}
		row = v.layout(geometry, rect).rowOf(pos)
	}
	top, bottom := row*geometry.lineH, (row+1)*geometry.lineH
	if top < v.ScrollY || bottom > v.ScrollY+sizes.viewH {
		v.ScrollY = max(top-(sizes.viewH-geometry.lineH)/2, 0) // centered
	}
}

// drawMatches fills the search hits on a row. A folded row shows a
// preview, whose columns are not the line's, so it shows no hits.
func (v *CodeView) drawMatches(ctx *core.Ctx, dst *ebiten.Image, geometry codeGeometry, cell core.Rect, row visualRow) {
	if v.folded(row.line) {
		return
	}
	for i, m := range v.matches {
		from, to := max(m.Col, row.from), min(m.Col+m.Len, row.to)
		if m.Line != row.line || to <= from {
			continue
		}
		tok := core.FindMatch
		if i == v.current {
			tok = core.FindActive
		}
		x := geometry.textX - v.ScrollX + geometry.colX(from-row.from)
		ctx.Fill(dst, core.Rect{X: x, Y: cell.Y, W: geometry.colX(to-row.from) - geometry.colX(from-row.from), H: cell.H}, ctx.Color(tok))
	}
}

// Matches returns the search hits currently highlighted.
func (v *CodeView) Matches() []Match { return v.matches }

// extendSelection moves the end of the selection for one movement key
// held with Shift. The caret is the end the pointer last set; a view with
// none starts at its first visible line. Folded lines are skipped, and
// the view scrolls to keep the caret in sight.
func (v *CodeView) extendSelection(edit core.Edit, layout wrapLayout) {
	visible := v.Visible()
	if len(visible) == 0 {
		return
	}
	if !v.HasSel {
		v.selA, v.selB, v.HasSel = textPos{visible[0], 0}, textPos{visible[0], 0}, true
	}
	v.selB = v.moveCaret(v.selB, edit, visible, layout)
	v.reveal = v.selB
}

// moveCaret is pos after one movement key, staying on the visible lines.
// Up and Down go to the next visual row, as in an editor, at the same
// offset from the row's start.
func (v *CodeView) moveCaret(pos textPos, edit core.Edit, visible []int, layout wrapLayout) textPos {
	row := sort.SearchInts(visible, pos.line)
	row = min(row, len(visible)-1)
	pos.line = visible[row]
	atRow := func(dy int) (textPos, bool) {
		at, x := layout.xOf(pos)
		if at < 0 || at+dy < 0 || at+dy >= len(layout.rows) {
			return pos, false
		}
		return layout.posAt(at+dy, x), true
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
		if above, ok := atRow(-1); ok {
			return above
		}
		return textPos{pos.line, 0}
	case core.EditDown:
		if below, ok := atRow(1); ok {
			return below
		}
		return textPos{pos.line, v.lineLen(pos.line)}
	case core.EditWordLeft:
		return v.wordLeftFrom(pos, row, visible)
	case core.EditWordRight:
		return v.wordRightFrom(pos, row, visible)
	case core.EditHome:
		return textPos{pos.line, 0}
	case core.EditEnd:
		return textPos{pos.line, v.lineLen(pos.line)}
	}
	return pos
}

// wordLeftFrom is the start of the word before pos. Separators are skipped
// across line ends, as in the text editor where a line break is one, so
// lines without a word (a lone brace) are passed over.
func (v *CodeView) wordLeftFrom(pos textPos, row int, visible []int) textPos {
	for {
		runes := []rune(v.lineText(visible[row]))
		col := min(pos.col, len(runes))
		for col > 0 && !isWordRune(runes[col-1]) {
			col--
		}
		if col > 0 || row == 0 {
			return textPos{visible[row], wordLeft(runes, col)}
		}
		row--
		pos = textPos{visible[row], v.lineLen(visible[row])}
	}
}

// wordRightFrom is the end of the word after pos, skipping separators
// across line ends like wordLeftFrom.
func (v *CodeView) wordRightFrom(pos textPos, row int, visible []int) textPos {
	for {
		runes := []rune(v.lineText(visible[row]))
		col := min(pos.col, len(runes))
		for col < len(runes) && !isWordRune(runes[col]) {
			col++
		}
		if col < len(runes) || row+1 == len(visible) {
			return textPos{visible[row], wordRight(runes, col)}
		}
		row++
		pos = textPos{visible[row], 0}
	}
}

func (v *CodeView) HasFocus() bool   { return v.Focused }
func (v *CodeView) SetFocus(on bool) { v.Focused = on }
