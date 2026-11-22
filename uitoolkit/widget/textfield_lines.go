package widget

import (
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// tabWidth is the indent step: the number of spaces Tab inserts, to the
// next multiple of it. It is not the display tab stop (core.TabStop).
const tabWidth = 2

// newMultiline returns a multi-line field holding text.
func newMultiline(text string) TextField {
	f := NewTextField(text)
	f.Multiline = true
	return f
}

// LineCount is the number of lines in the text (at least 1).
func (f *TextField) LineCount() int { return strings.Count(f.Text, "\n") + 1 }

// lineStart is the position of the start of the line containing pos.
func lineStart(runes []rune, pos int) int {
	for pos > 0 && runes[pos-1] != '\n' {
		pos--
	}
	return pos
}

// lineEnd is the position of the end of the line containing pos (before its newline).
func lineEnd(runes []rune, pos int) int {
	for pos < len(runes) && runes[pos] != '\n' {
		pos++
	}
	return pos
}

// LineCol returns the zero-based line and column of rune position pos.
func (f *TextField) LineCol(pos int) (line, col int) {
	r := []rune(f.Text)
	pos = min(max(pos, 0), len(r))
	for _, c := range r[:pos] {
		if c == '\n' {
			line++
		}
	}
	return line, pos - lineStart(r, pos)
}

// PosAt returns the rune position of a line and column, clamped to the
// text: a column past the end of a line is its end, a line past the end
// is the end of the text.
func (f *TextField) PosAt(line, col int) int {
	r := []rune(f.Text)
	pos := 0
	for i := 0; i < line; i++ {
		end := lineEnd(r, pos)
		if end >= len(r) {
			return len(r)
		}
		pos = end + 1
	}
	return min(pos+max(col, 0), lineEnd(r, pos))
}

// moveLines moves the cursor dy lines, keeping the goal display column (a
// tab counts to its stop, as the editor draws it). Moving
// past the first or last line goes to the start or end of the text.
func (f *TextField) moveLines(dy int, extend bool) {
	runes := []rune(f.Text)
	line, _ := f.LineCol(f.Cursor)
	start := lineStart(runes, f.Cursor)
	col := core.TextColumn(runes[start:], f.Cursor-start) // a display column, so a tab counts to its stop
	if f.goal > 0 {
		col = f.goal - 1
	}
	target := line + dy
	var pos int
	switch {
	case target < 0:
		pos = 0
	case target >= f.LineCount():
		pos = len(runes)
	default:
		at := f.PosAt(target, 0)
		pos = at + core.IndexAt(runes[at:lineEnd(runes, at)], col)
	}
	f.moveTo(pos, extend)
	f.goal = col + 1
}

// newline inserts a line break plus the indentation of the current
// line (up to the cursor), replacing the selection.
func (f *TextField) newline() bool {
	r := []rune(f.Text)
	lo := f.Cursor
	if l, _, ok := f.Selection(); ok {
		lo = l
	}
	start := lineStart(r, lo)
	indent := 0
	for start+indent < lo && (r[start+indent] == ' ' || r[start+indent] == '\t') {
		indent++
	}
	return f.insert("\n"+string(r[start:start+indent]), kindOther)
}

// selectedLines is the range of line starts touched by the selection
// (or the cursor's line): from the first line's start to the last
// line's start. A selection ending at the very start of a line does not
// include that line.
func (f *TextField) selectedLines() (starts []int) {
	r := []rune(f.Text)
	lo, hi := f.Cursor, f.Cursor
	if l, h, ok := f.Selection(); ok {
		lo, hi = l, h
		if hi > lo && hi > 0 && r[hi-1] == '\n' {
			hi--
		}
	}
	for pos := lineStart(r, lo); ; {
		starts = append(starts, pos)
		end := lineEnd(r, pos)
		if end >= hi || end >= len(r) {
			return starts
		}
		pos = end + 1
	}
}

// shiftPos moves p by the changes made at line starts: delta[i] runes
// were added (positive) or removed (negative) at starts[i].
func shiftPos(value int, starts, delta []int) int {
	out := value
	for i, s := range starts {
		if s <= value {
			out += delta[i]
			if delta[i] < 0 && value < s-delta[i] {
				out = s + sumBefore(delta, i) // a position inside the removed run moves to the line start
			}
		}
	}
	return out
}

func sumBefore(delta []int, index int) int {
	n := 0
	for _, d := range delta[:index] {
		n += d
	}
	return n
}

// indent inserts spaces: at the next tab stop for a caret or a
// single-line selection, or at the start of every selected line.
func (f *TextField) indent() bool {
	starts := f.selectedLines()
	if len(starts) == 1 {
		_, col := f.LineCol(f.Cursor)
		if lo, _, ok := f.Selection(); ok {
			_, col = f.LineCol(lo)
		}
		return f.insert(strings.Repeat(" ", tabWidth-col%tabWidth), kindOther)
	}
	return f.shiftLines(starts, func(_ []rune, _ int) int { return tabWidth })
}

// dedent removes one indent level (up to tabWidth spaces, or a tab)
// from the start of every selected line or the cursor's line.
func (f *TextField) dedent() bool {
	return f.shiftLines(f.selectedLines(), func(runes []rune, start int) int {
		if start < len(runes) && runes[start] == '\t' {
			return -1
		}
		n := 0
		for n < tabWidth && start+n < len(runes) && runes[start+n] == ' ' {
			n++
		}
		return -n
	})
}

// shiftLines applies change(r, lineStart) to each line start: positive
// adds that many spaces, negative removes that many runes. The cursor
// and the selection follow the text.
func (f *TextField) shiftLines(starts []int, change func(runes []rune, start int) int) bool {
	r := []rune(f.Text)
	delta := make([]int, len(starts))
	any := false
	for i, s := range starts {
		delta[i] = change(r, s)
		any = any || delta[i] != 0
	}
	if !any {
		return false
	}
	f.record(kindOther)
	var out []rune
	prev := 0
	for i, s := range starts {
		out = append(out, r[prev:s]...)
		if delta[i] > 0 {
			out = append(out, []rune(strings.Repeat(" ", delta[i]))...)
			prev = s
		} else {
			prev = s - delta[i]
		}
	}
	out = append(out, r[prev:]...)
	f.Cursor, f.Anchor = shiftPos(f.Cursor, starts, delta), shiftPos(f.Anchor, starts, delta)
	f.Text = string(out)
	return true
}
