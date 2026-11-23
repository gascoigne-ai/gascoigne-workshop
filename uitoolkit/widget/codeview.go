package widget

import (
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

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
	Lines   []Line
	Folded  map[int]bool // by original line index
	ScrollY int          // pixels
	ScrollX int          // pixels
	maxCols int          // display columns of the widest line, for the horizontal extent

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

	// gen counts SetLines calls; texts and laid are valid for the lines of
	// one gen.
	gen   uint64
	texts textCache
	laid  layoutCache
	// layoutBuilds counts the layouts built, for the tests of the cache.
	layoutBuilds int
	// digitScans counts the scans for the widest line number, for the same.
	digitScans int
	// rowBuf holds the pieces of the row being drawn; it is overwritten for
	// each row and nothing keeps it after Draw returns.
	rowBuf []Span
}

// noReveal is the reveal of a view with nothing to scroll to.
var noReveal = textPos{line: -1}

// wrapWidth is the room in pixels for the text of a row, 0 when the view
// does not wrap.
func (v *CodeView) wrapWidth(geometry codeGeometry, rect core.Rect) float64 {
	return geometry.wrapWidth(v.Wrap, rect)
}

// layout is the visual rows of the visible lines. A folded line is one row
// that is drawn whole. Every rune is one advance wide, as drawing places
// text by column.
func (v *CodeView) layout(geometry codeGeometry, rect core.Rect) wrapLayout {
	return v.layoutAt(v.wrapWidth(geometry, rect), geometry)
}

func (v *CodeView) layoutAt(width float64, geometry codeGeometry) wrapLayout {
	key := layoutKey{lines: v.linesKey(), folds: v.foldsKey(), width: width, adv: geometry.adv}
	if v.laid.ok && v.laid.key == key {
		return v.laid.layout
	}
	v.layoutBuilds++
	visible := v.Visible()
	lines := make([]wrapLine, len(visible))
	for i, line := range visible {
		lines[i] = wrapLine{line: line, text: v.lineRunes(line), whole: v.folded(line)}
	}
	v.laid = layoutCache{key: key, layout: layoutWrap(lines, width, monoColumns(geometry.adv)), ok: true}
	return v.laid.layout
}

// folded reports whether line i is shown folded, as one row of its preview.
func (v *CodeView) folded(index int) bool { return v.Folded[index] && v.Lines[index].Foldable }

// keepTopLine keeps the line at the top of the view in place when Wrap
// has changed since the last layout, as the number of rows above it does.
func (v *CodeView) keepTopLine(geometry codeGeometry, rect core.Rect) {
	if v.Wrap == v.laidWrap {
		return
	}
	before := v.layoutAt(geometry.wrapWidth(v.laidWrap, rect), geometry)
	v.laidWrap = v.Wrap
	v.ScrollY = geometry.keepTop(v.ScrollY, before, v.layout(geometry, rect))
}

// appendRowSpans appends the part of spans from rune from to rune to. The
// pieces are substrings of the spans' texts, so a row costs no copy and,
// with a reused dst, no allocation.
func appendRowSpans(dst, spans []Span, from, to int) []Span {
	pos := 0
	for _, sp := range spans {
		count := utf8.RuneCountInString(sp.Text)
		lo, hi := max(from-pos, 0), min(to-pos, count)
		if lo < hi {
			start := runeOffset(sp.Text, lo)
			end := start + runeOffset(sp.Text[start:], hi-lo)
			dst = append(dst, Span{Text: sp.Text[start:end], Style: sp.Style, Icon: sp.Icon})
		}
		pos += count
	}
	return dst
}

// runeOffset is the byte offset of rune number index in text, len(text)
// when the text has fewer runes. A byte that is not valid UTF-8 counts as
// one rune, as []rune(text) counts it.
func runeOffset(text string, index int) int {
	for offset := range text {
		if index == 0 {
			return offset
		}
		index--
	}
	return len(text)
}

// SetLines replaces the content and resets folds and scroll.
func (v *CodeView) SetLines(lines []Line) {
	v.Lines = lines
	v.Folded = map[int]bool{}
	v.ScrollY = 0
	v.HasSel, v.dragging, v.drag = false, false, barDrag{}
	v.ScrollX, v.maxCols = 0, 0
	v.matches, v.current, v.reveal = nil, -1, noReveal
	v.gen++
	v.texts = v.runesOfLines()
	for _, runes := range v.texts.runes {
		v.maxCols = max(v.maxCols, core.TextColumn(runes, len(runes)))
	}
}

// foldedMore is what a folded block without a preview of its own shows
// after its first line.
const foldedMore = " ... }"

// shownSpans are the spans line i is drawn with: the preview of a folded
// block, else the line itself.
func (v *CodeView) shownSpans(index int) []Span {
	line := v.Lines[index]
	if !v.folded(index) {
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
	return spanText(v.shownSpans(index))
}

// rawText is the text of line i itself, whether or not it is folded.
func (v *CodeView) rawText(index int) string {
	return spanText(v.Lines[index].Spans)
}

// spanText is the plain text of spans joined.
func spanText(spans []Span) string {
	var b strings.Builder
	for _, span := range spans {
		b.WriteString(span.Text)
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

func (p textPos) before(other textPos) bool {
	return p.line < other.line || (p.line == other.line && p.col < other.col)
}

// lineLen is the number of runes in line i.
func (v *CodeView) lineLen(index int) int { return len(v.lineRunes(index)) }

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
func (v *CodeView) SelectLines(start, end int) {
	last := len(v.Lines) - 1
	if last < 0 {
		v.HasSel = false
		return
	}
	lo, hi := min(max(start, 0), last), min(max(end, 0), last)
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

// numberDigits is how many digits the widest line number has, from the
// texts cache.
func (v *CodeView) numberDigits() int { return v.freshTexts().digits }

func (v *CodeView) scanNumberDigits() int {
	v.digitScans++
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
		if !v.folded(i) {
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
	monoGrid
	numW int
}

func (v *CodeView) geometry(ctx *core.Ctx, rect core.Rect) codeGeometry {
	g := codeGeometry{monoGrid: newMonoGrid(ctx)}
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
	contentW := v.contentWidthOf(g, rect)
	b := v.bars(rect, g, contentW, len(layout.rows))
	v.updateBars(ctx, g, b, contentW, len(layout.rows))
	if v.reveal.line >= 0 {
		v.applyReveal(rect, g, b, layout)
		layout = v.layout(g, rect)
		b = v.bars(rect, g, contentW, len(layout.rows))
	}
	if len(layout.rows) > 0 && !v.drag.active() {
		v.updatePointer(ctx, rect, g, b, contentW, layout, over)
	}
	v.clampScroll(g, b, contentW, len(layout.rows))
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

func (v *CodeView) clampScroll(geometry codeGeometry, sizes bars, contentW, visible int) {
	v.ScrollY = clampScroll(v.ScrollY, visible*geometry.lineH, sizes.viewH)
	v.ScrollX = clampScroll(v.ScrollX, contentW, sizes.viewW)
}

// updateBars handles grabbing, dragging and page-clicking the scrollbars.
func (v *CodeView) updateBars(ctx *core.Ctx, geometry codeGeometry, sizes bars, contentW, visible int) {
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
		lo, hi := wordAt(runes, textPosValue.col)
		v.selA, v.selB, v.HasSel, v.dragging = textPos{textPosValue.line, lo}, textPos{textPosValue.line, hi}, true, false
	case clicks >= 3:
		v.selA, v.selB, v.HasSel, v.dragging = textPos{textPosValue.line, 0}, textPos{textPosValue.line, len(runes)}, true, false
	default:
		v.selA, v.selB, v.HasSel, v.dragging = textPosValue, textPosValue, true, true
	}
}

func (v *CodeView) updatePointer(ctx *core.Ctx, rect core.Rect, geometry codeGeometry, sizes bars, contentW int, layout wrapLayout, over bool) {
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
			v.ScrollX = clampScroll(v.ScrollX+geometry.wheelDX(in.Wheel), contentW, sizes.viewW)
		} else {
			v.ScrollBy(geometry.wheelDY(in.Wheel), rowCount*geometry.lineH, sizes.viewH)
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
	layout := v.layout(g, rect)
	rows := layout.rows
	contentW := v.contentWidthOf(g, rect)
	b := v.bars(rect, g, contentW, len(rows))
	v.clampScroll(g, b, contentW, len(rows))
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
		v.drawMatches(ctx, text, g, layout, cell, row)
		if lo, hi, ok := v.selRange(); ok && idx >= lo.line && idx <= hi.line {
			v.drawSelection(ctx, text, g, layout, cell, row, lo, hi)
		}

		spans := v.shownSpans(idx)
		if row.from > 0 || row.to < v.lineLen(idx) {
			v.rowBuf = appendRowSpans(v.rowBuf[:0], spans, row.from, row.to)
			spans = v.rowBuf
		}
		g.drawSpans(ctx, text, v.ScrollX, cell, row, spans)

		if row.from > 0 { // a continuation row has no number and no fold arrow
			continue
		}
		num := strconv.Itoa(v.LineNumber(idx))
		ctx.TextMid(area, ctx.F.Mono, num, rect.X+core.Gutter+g.numW-len(num)*g.charW, cell, muted)
		if line.Foldable {
			arrow := "triangle-down"
			if v.Folded[idx] {
				arrow = "triangle-right"
			}
			cx := rect.X + core.Gutter + g.numW + 4 + foldColW/2
			drawIconAt(ctx, area, arrow, core.Rect{X: cx - foldColW/2, Y: y, W: foldColW, H: g.lineH}, core.Muted)
		}
	}
	drawBars(ctx, dst, b, v.drag, contentW, len(rows)*g.lineH, v.ScrollX, v.ScrollY)
}

// contentWidthOf is the width of the widest line plus the gutter, by a
// geometry built once for the call.
func (v *CodeView) contentWidthOf(geometry codeGeometry, rect core.Rect) int {
	if v.Wrap {
		return 0 // wrapped text fits the width
	}
	return geometry.textX - rect.X + geometry.colX(v.maxCols) + core.Gutter
}

func (v *CodeView) bars(rect core.Rect, geometry codeGeometry, contentW, visible int) bars {
	return layoutBars(rect, contentW, visible*geometry.lineH)
}

// Selected reports whether any text is selected.
func (v *CodeView) Selected() bool { _, _, ok := v.selRange(); return ok }

// drawSelection fills the selected part of line idx; lines that continue
// into the next selected line also fill one character for the newline.
func (v *CodeView) drawSelection(ctx *core.Ctx, dst *ebiten.Image, geometry codeGeometry, layout wrapLayout, cell core.Rect, row visualRow, lo, hi textPos) {
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
	ctx.Fill(dst, geometry.selectionRect(v.ScrollX, layout.colIn(row, from), layout.colIn(row, to), cell.Y, cell.H), ctx.Color(core.Selection))
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
func (v *CodeView) drawMatches(ctx *core.Ctx, dst *ebiten.Image, geometry codeGeometry, layout wrapLayout, cell core.Rect, row visualRow) {
	if v.folded(row.line) {
		return
	}
	walker := layout.walk(row) // matches come sorted, so one pass counts all their columns
	for i, m := range v.matches {
		from, to := max(m.Col, row.from), min(m.Col+m.Len, row.to)
		if m.Line != row.line || to <= from {
			continue
		}
		tok := core.FindMatch
		if i == v.current {
			tok = core.FindActive
		}
		ctx.Fill(dst, geometry.selectionRect(v.ScrollX, walker.colIn(from), walker.colIn(to), cell.Y, cell.H), ctx.Color(tok))
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

// HasFocus reports whether the view has the keyboard focus.
func (v *CodeView) HasFocus() bool { return v.Focused }

// SetFocus gives or takes the keyboard focus.
func (v *CodeView) SetFocus(on bool) { v.Focused = on }
