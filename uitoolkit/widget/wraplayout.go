package widget

import (
	"sort"
	"unicode"
)

// wrapLine is one logical line handed to the wrap layout.
type wrapLine struct {
	line  int // original line index, so folded-away lines can be left out
	text  []rune
	whole bool // drawn on one row and never broken, as a folded line's preview is
}

// visualRow is one row of wrapped text: runes from..to (to exclusive) of an
// original line. x is the width of the line's runes before from, which
// turns a pixel offset on the row into one along the whole line.
type visualRow struct {
	line, from, to int
	x              float64
}

// wrapLayout is the visual rows of a run of lines. It keeps the texts and
// the measure that made the rows so that positions and pixels map back
// through the same widths the rows were cut by.
type wrapLayout struct {
	rows    []visualRow
	texts   map[int][]rune // by original line
	measure func(char rune) float64
}

// layoutWrap breaks lines into visual rows no wider than width, as measured
// rune by rune. A row ends before the word that would overflow it; a word
// wider than the view is cut at the width; spaces at a break hang at the
// end of their row, so the next row starts with text. Every row holds at
// least one rune, so a width smaller than a glyph still shows the text.
// A width of 0 or less means no wrap: one row per line. Lines come in
// increasing original order, each once, as Visible lists them: the
// mappings search the rows by line.
func layoutWrap(lines []wrapLine, width float64, measure func(char rune) float64) wrapLayout {
	layout := wrapLayout{rows: make([]visualRow, 0, len(lines)), texts: make(map[int][]rune, len(lines)), measure: measure}
	for _, line := range lines {
		layout.texts[line.line] = line.text
		if line.whole || width <= 0 {
			layout.rows = append(layout.rows, visualRow{line: line.line, to: len(line.text)})
			continue
		}
		layout.rows = append(layout.rows, wrapRunes(line.line, line.text, width, measure)...)
	}
	return layout
}

// wrapRunes is the rows of one line. An empty line still has its row.
func wrapRunes(line int, text []rune, width float64, measure func(char rune) float64) []visualRow {
	rows := []visualRow{}
	x := 0.0
	for from := 0; from < len(text) || len(rows) == 0; {
		to := fitRunes(text, from, width, measure)
		if to < len(text) {
			to = lastBreak(text, from, to)
		}
		rows = append(rows, visualRow{line: line, from: from, to: to, x: x})
		x += widthOf(text[from:to], measure)
		from = to
	}
	return rows
}

// fitRunes is how far from from the text fits in width: at least one rune,
// and spaces never overflow, they hang past the edge.
func fitRunes(text []rune, from int, width float64, measure func(char rune) float64) int {
	used := 0.0
	for i := from; i < len(text); i++ {
		used += measure(text[i])
		if used > width && i > from && !unicode.IsSpace(text[i]) {
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
func canBreak(text []rune, i int) bool {
	before, after := text[i-1], text[i]
	if unicode.IsSpace(after) {
		return false // spaces hang at the end of the row they follow
	}
	return unicode.IsSpace(before) || (isWordRune(after) && !isWordRune(before))
}

func widthOf(text []rune, measure func(char rune) float64) float64 {
	sum := 0.0
	for _, char := range text {
		sum += measure(char)
	}
	return sum
}

// rowOf is the row that shows pos, -1 when its line has no row (folded
// away or not laid out). A column at a break belongs to the row it starts,
// as a caret there is drawn at the start of the next row; a column past
// the line's end is on its last row.
func (layout wrapLayout) rowOf(pos textPos) int {
	col := max(pos.col, 0)
	after := sort.Search(len(layout.rows), func(i int) bool {
		row := layout.rows[i]
		return row.line > pos.line || (row.line == pos.line && row.from > col)
	})
	if after == 0 || layout.rows[after-1].line != pos.line {
		return -1
	}
	return after - 1
}

// firstRow is the row a line starts on, -1 when it has none, so a view can
// keep its top line in place when the rows change.
func (layout wrapLayout) firstRow(line int) int { return layout.rowOf(textPos{line, 0}) }

// rowStart is the position at the start of a row (clamped to the rows),
// the line and column a view scrolled to that row shows first.
func (layout wrapLayout) rowStart(row int) textPos {
	if len(layout.rows) == 0 {
		return textPos{}
	}
	start := layout.rows[min(max(row, 0), len(layout.rows)-1)]
	return textPos{start.line, start.from}
}

// xOf is the row that shows pos and the pixel offset of pos from the start
// of that row; the row is -1 when the line has none.
func (layout wrapLayout) xOf(pos textPos) (row int, x float64) {
	row = layout.rowOf(pos)
	if row < 0 {
		return -1, 0
	}
	at := layout.rows[row]
	end := min(max(pos.col, at.from), at.to)
	return row, widthOf(layout.texts[at.line][at.from:end], layout.measure)
}

// posAt is the position nearest to pixel offset x on a row (both clamped:
// left of the text is the row's start, right of it the row's end), for
// hit-testing a click. A row that its line continues on ends one rune
// early, before the hanging space or the cut, because its own end column
// is drawn at the start of the next row (rowOf).
func (layout wrapLayout) posAt(row int, x float64) textPos {
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
	left := 0.0
	for col := at.from; col < end; col++ {
		advance := layout.measure(text[col])
		if x < left+advance/2 {
			return textPos{at.line, col}
		}
		left += advance
	}
	return textPos{at.line, end}
}
