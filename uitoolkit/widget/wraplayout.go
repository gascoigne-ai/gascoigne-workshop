package widget

import (
	"sort"
	"unicode"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// wrapLine is one logical line handed to the wrap layout.
type wrapLine struct {
	line  int // original line index, so folded-away lines can be left out
	text  []rune
	whole bool // drawn on one row and never broken, as a folded line's preview is
}

// visualRow is one row of wrapped text: runes from..to (to exclusive) of an
// original line. col is the display column of from in its line, so a row
// that starts mid-line keeps the line's tab stops.
type visualRow struct {
	line, from, to int
	col            int
}

// columns is how a layout counts: next is the display column after a rune
// at a column (core.ColumnAfter in the views, so a tab reaches its stop),
// adv the width of one column in pixels.
type columns struct {
	next func(col int, char rune) int
	adv  float64
}

// monoColumns are the columns of a monospace view: every rune one advance
// wide, a tab to its stop.
func monoColumns(adv float64) columns { return columns{next: core.ColumnAfter, adv: adv} }

// span is the display column after text[from:to] drawn from column col.
func (cols columns) span(text []rune, from, to, col int) int {
	for _, char := range text[from:to] {
		col = cols.next(col, char)
	}
	return col
}

// spanString is span over all of text, without making its runes.
func (cols columns) spanString(text string, col int) int {
	for _, char := range text {
		col = cols.next(col, char)
	}
	return col
}

// wrapLayout is the visual rows of a run of lines. It keeps the texts and
// the columns that made the rows so that positions and pixels map back
// through the same widths the rows were cut by.
type wrapLayout struct {
	rows  []visualRow
	texts map[int][]rune // by original line
	cols  columns
}

// layoutWrap breaks lines into visual rows no wider than width, as counted
// rune by rune in cols. A row ends before the word that would overflow it;
// a word wider than the view is cut at the width; spaces at a break hang at
// the end of their row, so the next row starts with text. Every row holds
// at least one rune, so a width smaller than a glyph still shows the text.
// A width of 0 or less means no wrap: one row per line. Lines come in
// increasing original order, each once, as Visible lists them: the
// mappings search the rows by line.
func layoutWrap(lines []wrapLine, width float64, cols columns) wrapLayout {
	layout := wrapLayout{rows: make([]visualRow, 0, len(lines)), texts: make(map[int][]rune, len(lines)), cols: cols}
	for _, line := range lines {
		layout.texts[line.line] = line.text
		if line.whole || width <= 0 {
			layout.rows = append(layout.rows, visualRow{line: line.line, to: len(line.text)})
			continue
		}
		layout.rows = append(layout.rows, wrapRunes(line.line, line.text, width, cols)...)
	}
	return layout
}

// wrapRunes is the rows of one line. An empty line still has its row.
func wrapRunes(line int, text []rune, width float64, cols columns) []visualRow {
	rows := []visualRow{}
	col := 0
	for from := 0; from < len(text) || len(rows) == 0; {
		to := fitRunes(text, from, col, width, cols)
		if to < len(text) {
			to = lastBreak(text, from, to)
		}
		rows = append(rows, visualRow{line: line, from: from, to: to, col: col})
		col = cols.span(text, from, to, col)
		from = to
	}
	return rows
}

// fitRunes is how far from from (at display column col) the text fits in
// width: at least one rune, and spaces past the width still fit since they
// hang.
func fitRunes(text []rune, from, col int, width float64, cols columns) int {
	start := col
	for i := from; i < len(text); i++ {
		col = cols.next(col, text[i])
		if float64(col-start)*cols.adv > width && i > from && !unicode.IsSpace(text[i]) {
			return i
		}
	}
	return len(text)
}

// lastBreak is the last place in from..to where the row may end, or to
// itself when the row is one word that has to be cut.
func lastBreak(text []rune, from, to int) int {
	for i := to; i > from; i-- {
		if canBreak(text, i) {
			return i
		}
	}
	return to
}

// canBreak reports whether a row may end before rune i: after spaces, or
// where a word starts after punctuation, so a row ends with the
// punctuation ("example." | "com") and never starts with it. The word rule
// is the one of word-wise motion, isWordRune.
func canBreak(text []rune, index int) bool {
	before, after := text[index-1], text[index]
	if unicode.IsSpace(after) {
		return false // spaces hang at the end of the row they follow
	}
	return unicode.IsSpace(before) || (isWordRune(after) && !isWordRune(before))
}

// rowOf is the row that shows pos, -1 when its line has no row (folded
// away or not laid out). A column at a break belongs to the row it starts,
// as a caret there is drawn at the start of the next row; a column past
// the line's end is on its last row.
func (layout wrapLayout) rowOf(pos textPos) int {
	col := max(pos.col, 0)
	after := sort.Search(len(layout.rows), func(index int) bool {
		row := layout.rows[index]
		return row.line > pos.line || (row.line == pos.line && row.from > col)
	})
	if after == 0 || layout.rows[after-1].line != pos.line {
		return -1
	}
	return after - 1
}

// rowStart is the position at the start of a row (clamped to the rows),
// the line and column a view scrolled to that row shows first.
func (layout wrapLayout) rowStart(row int) textPos {
	if len(layout.rows) == 0 {
		return textPos{}
	}
	start := layout.rows[min(max(row, 0), len(layout.rows)-1)]
	return textPos{start.line, start.from}
}

// rowCol is the row that shows pos and the display column of pos from the
// start of that row; the row is -1 when the line has none.
func (layout wrapLayout) rowCol(pos textPos) (row, col int) {
	row = layout.rowOf(pos)
	if row < 0 {
		return -1, 0
	}
	return row, layout.colIn(layout.rows[row], min(pos.col, layout.rows[row].to))
}

// colIn is the display column of rune index of row's line, counted from
// the row's start (an index before it is its start). An index past the
// row's end adds one column a rune: the line break a selection covers.
func (layout wrapLayout) colIn(row visualRow, index int) int {
	return layout.walk(row).colIn(index)
}

// rowWalker counts display columns along a row for rune indexes asked in
// increasing order, so the many hits on one long row cost one pass.
type rowWalker struct {
	layout     wrapLayout
	row        visualRow
	index, col int // the rune index reached and its column along the line
}

func (layout wrapLayout) walk(row visualRow) *rowWalker {
	return &rowWalker{layout: layout, row: row, index: row.from, col: row.col}
}

// colIn is wrapLayout.colIn; an index before the one asked last starts
// over from the row's start.
func (walker *rowWalker) colIn(index int) int {
	if index < walker.index {
		walker.index, walker.col = walker.row.from, walker.row.col
	}
	text := walker.layout.texts[walker.row.line]
	for ; walker.index < min(index, walker.row.to); walker.index++ {
		walker.col = walker.layout.cols.next(walker.col, text[walker.index])
	}
	return walker.col - walker.row.col + max(index-walker.row.to, 0)
}

// xOf is the row that shows pos and the pixel offset of pos from the start
// of that row; the row is -1 when the line has none.
func (layout wrapLayout) xOf(pos textPos) (row int, x float64) {
	row, col := layout.rowCol(pos)
	return row, float64(col) * layout.cols.adv
}

// posAt is the position nearest to pixel offset x on a row (both clamped:
// left of the text is the row's start, right of it the row's end), for
// hit-testing a click. A row that its line continues on ends one rune
// early, before the hanging space or the cut, because its own end column
// is drawn at the start of the next row (rowOf).
func (layout wrapLayout) posAt(row int, posX float64) textPos {
	if len(layout.rows) == 0 {
		return textPos{}
	}
	row = min(max(row, 0), len(layout.rows)-1)
	at := layout.rows[row]
	end := at.to
	if continued := row+1 < len(layout.rows) && layout.rows[row+1].line == at.line; continued && end > at.from {
		end--
	}
	text := layout.texts[at.line]
	col := at.col
	for index := at.from; index < end; index++ {
		next := layout.cols.next(col, text[index])
		if posX < float64(col+next-2*at.col)*layout.cols.adv/2 {
			return textPos{at.line, index}
		}
		col = next
	}
	return textPos{at.line, end}
}
