package core

import (
	"strings"
	"unicode"
)

// Edit is a cursor or deletion key applied to a TextField.
type Edit int

const (
	EditBackspace Edit = iota
	EditDelete
	EditLeft
	EditRight
	EditHome
	EditEnd
	EditWordLeft
	EditWordRight
	EditDeleteWordLeft
	EditDeleteWordRight
	EditUp
	EditDown
	EditPageUp
	EditPageDown
	EditDocStart
	EditDocEnd
	EditNewline
	EditIndent
	EditDedent
)

// Pixels of context kept visible beyond the cursor when the text
// scrolls, and the number of undo steps kept.
const (
	revealMargin = 12
	historyLimit = 100
)

// editKind groups edits for undo: consecutive edits of the same
// coalescing kind (typing, deleting) are one undo step.
type editKind int

const (
	kindNone   editKind = iota // no edit yet, or the cursor moved since
	kindType                   // typed characters
	kindDelete                 // backspace and delete, one character at a time
	kindOther                  // paste, cut, word deletion: always its own step
)

// snapshot is a restorable state of a field.
type snapshot struct {
	text           string
	cursor, anchor int
	hasSel         bool
}

// TextField is a single-line editable string with a cursor, an
// optional selection and undo history. Positions count runes.
type TextField struct {
	Text     string
	Cursor   int
	Anchor   int  // the fixed end of the selection; meaningful when HasSel
	HasSel   bool // a selection exists between Anchor and Cursor
	Focused  bool
	Dragging bool // the pointer went down inside and is still held
	ScrollX  int  // pixels the text is scrolled left inside its view

	// Multiline keeps line breaks and enables vertical movement; ViewLines
	// is how many lines PageUp/PageDown move.
	Multiline bool
	ViewLines int

	// Masked shows bullets instead of the text and refuses to copy or
	// cut it, for secrets.
	Masked bool
	goal   int // goal column + 1 for vertical movement, 0 when unset

	undo, redo []snapshot
	last       editKind
}

// NewTextField returns a field holding text with the cursor at its end.
func NewTextField(text string) TextField {
	return TextField{Text: text, Cursor: len([]rune(text))}
}

func (f *TextField) clamp(n int) int { return min(max(n, 0), len([]rune(f.Text))) }

// Selection returns the selected range, low to high; ok is false when
// nothing is selected.
func (f *TextField) Selection() (lo, hi int, ok bool) {
	if !f.HasSel || f.Anchor == f.Cursor {
		return 0, 0, false
	}
	return min(f.Anchor, f.Cursor), max(f.Anchor, f.Cursor), true
}

// SelectedText is the selected text, or "".
func (f *TextField) SelectedText() string {
	lo, hi, ok := f.Selection()
	if !ok {
		return ""
	}
	return string([]rune(f.Text)[lo:hi])
}

// SelectAll selects the whole text.
func (f *TextField) SelectAll() {
	f.Anchor, f.Cursor, f.HasSel = 0, len([]rune(f.Text)), true
	f.last = kindNone
}

// --- undo ---------------------------------------------------------------

func (f *TextField) snapshot() snapshot {
	return snapshot{f.Text, f.Cursor, f.Anchor, f.HasSel}
}

func (f *TextField) restore(s snapshot) {
	f.Text, f.Cursor, f.Anchor, f.HasSel = s.text, s.cursor, s.anchor, s.hasSel
	f.last = kindNone
}

// record saves the state before an edit of the given kind. Typing and
// deleting coalesce with the previous edit of the same kind.
func (f *TextField) record(kind editKind) {
	if kind != kindOther && f.last == kind {
		return
	}
	f.undo = append(f.undo, f.snapshot())
	if len(f.undo) > historyLimit {
		f.undo = f.undo[len(f.undo)-historyLimit:]
	}
	f.redo = nil
	f.last = kind
}

// Undo restores the state before the last edit group; it reports
// whether anything changed.
func (f *TextField) Undo() bool {
	if len(f.undo) == 0 {
		return false
	}
	f.redo = append(f.redo, f.snapshot())
	f.restore(f.undo[len(f.undo)-1])
	f.undo = f.undo[:len(f.undo)-1]
	return true
}

// Redo reapplies the edit group an Undo removed.
func (f *TextField) Redo() bool {
	if len(f.redo) == 0 {
		return false
	}
	f.undo = append(f.undo, f.snapshot())
	f.restore(f.redo[len(f.redo)-1])
	f.redo = f.redo[:len(f.redo)-1]
	return true
}

// --- editing ------------------------------------------------------------

// DeleteSelection removes the selected text and reports whether there was any.
func (f *TextField) DeleteSelection() bool {
	if _, _, ok := f.Selection(); !ok {
		f.HasSel = false
		return false
	}
	f.record(kindOther)
	f.deleteSelection()
	return true
}

// deleteSelection removes the selection without recording history.
func (f *TextField) deleteSelection() {
	lo, hi, ok := f.Selection()
	if !ok {
		f.HasSel = false
		return
	}
	runes := []rune(f.Text)
	f.Text = string(append(runes[:lo], runes[hi:]...))
	f.Cursor, f.HasSel = lo, false
}

// Insert replaces the selection (if any) with s at the cursor, as a
// paste: its own undo step. Line breaks become spaces and other control
// characters are dropped, since the field has one line. It reports
// whether the text changed.
func (f *TextField) Insert(s string) bool { return f.insert(s, kindOther) }

func (f *TextField) insert(s string, kind editKind) bool {
	if f.Multiline {
		s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
	}
	clean := strings.Map(func(r rune) rune {
		if f.Multiline && (r == '\n' || r == '\t') {
			return r
		}
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}
		if r < ' ' || r == 0x7f {
			return -1
		}
		return r
	}, s)
	_, _, hasSel := f.Selection()
	if clean == "" && !hasSel {
		return false
	}
	f.record(kind)
	f.deleteSelection()
	f.Cursor = f.clamp(f.Cursor)
	runes := []rune(f.Text)
	ins := []rune(clean)
	f.Text = string(append(runes[:f.Cursor], append(ins, runes[f.Cursor:]...)...))
	f.Cursor += len(ins)
	return true
}

// keepPrintable drops control characters that key presses report as
// typed text (Enter, Tab, Delete).
func keepPrintable(runes []rune) []rune {
	var out []rune
	for _, r := range runes {
		if r >= ' ' && r != 0x7f {
			out = append(out, r)
		}
	}
	return out
}

// --- movement and selection ---------------------------------------------

// moveTo puts the cursor at pos; with extend it grows the selection
// from where the cursor was, otherwise it clears the selection.
func (f *TextField) moveTo(pos int, extend bool) {
	pos = f.clamp(pos)
	if extend {
		if !f.HasSel {
			f.Anchor, f.HasSel = f.Cursor, true
		}
	} else {
		f.HasSel = false
	}
	f.Cursor = pos
	f.last = kindNone
	f.goal = 0
}

// Press handles the pointer going down at rune position pos: it places
// the cursor (or extends the selection with shift) and starts a drag.
func (f *TextField) Press(pos int, extend bool) {
	f.moveTo(pos, extend)
	f.Dragging = true
}

// PressN is Press with a click count: 2 selects the word at pos, 3
// selects everything. Neither starts a drag.
func (f *TextField) PressN(pos int, extend bool, clicks int) {
	switch clicks {
	case 2:
		lo, hi := wordAt([]rune(f.Text), f.clamp(pos))
		f.Anchor, f.Cursor, f.HasSel, f.Dragging = lo, hi, true, false
		f.last = kindNone
	case 3:
		if f.Multiline {
			r := []rune(f.Text)
			pos = f.clamp(pos)
			f.Anchor, f.Cursor, f.HasSel = lineStart(r, pos), min(lineEnd(r, pos)+1, len(r)), true
			f.last = kindNone
		} else {
			f.SelectAll()
		}
		f.Dragging = false
	default:
		f.Press(pos, extend)
	}
}

// Drag extends the selection to pos while the pointer is held.
func (f *TextField) Drag(pos int) {
	if !f.HasSel {
		f.Anchor, f.HasSel = f.Cursor, true
	}
	f.Cursor = f.clamp(pos)
}

func isWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' }

// wordLeft is the start of the word before pos (skipping separators).
func wordLeft(r []rune, pos int) int {
	i := min(max(pos, 0), len(r))
	for i > 0 && !isWordRune(r[i-1]) {
		i--
	}
	for i > 0 && isWordRune(r[i-1]) {
		i--
	}
	return i
}

// wordRight is the end of the word after pos (skipping separators).
func wordRight(r []rune, pos int) int {
	i := min(max(pos, 0), len(r))
	for i < len(r) && !isWordRune(r[i]) {
		i++
	}
	for i < len(r) && isWordRune(r[i]) {
		i++
	}
	return i
}

// wordAt is the range of the run of word characters, or of separator
// characters, at pos (the character to its right, else to its left).
func wordAt(r []rune, pos int) (lo, hi int) {
	if len(r) == 0 {
		return 0, 0
	}
	at := min(pos, len(r)-1)
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

// Reveal scrolls the field so a cursor cursorX pixels from the start of
// the text stays inside a view viewW wide, with a margin. Text that
// fits is never scrolled.
func (f *TextField) Reveal(cursorX, viewW int) {
	switch {
	case cursorX <= viewW-revealMargin:
		f.ScrollX = 0
	case cursorX-f.ScrollX > viewW-revealMargin:
		f.ScrollX = cursorX - viewW + revealMargin
	case cursorX-f.ScrollX < revealMargin:
		f.ScrollX = max(cursorX-revealMargin, 0)
	}
}

// --- applying keys --------------------------------------------------------

// Apply inserts typed runes at the cursor, then runs edits in order.
// With extend (shift held) the movement keys select instead of just
// moving. It reports whether Text changed.
func (f *TextField) Apply(typed []rune, edits []Edit, extend bool) (changed bool) {
	f.Cursor = f.clamp(f.Cursor)
	if printable := keepPrintable(typed); len(printable) > 0 {
		changed = f.insert(string(printable), kindType) || changed
	}
	for _, e := range edits {
		changed = f.edit(e, extend) || changed
	}
	return changed
}

func (f *TextField) edit(e Edit, extend bool) (changed bool) {
	runes := []rune(f.Text)
	switch e {
	case EditBackspace:
		return f.deleteRange(f.Cursor-1, f.Cursor, kindDelete)
	case EditDelete:
		return f.deleteRange(f.Cursor, f.Cursor+1, kindDelete)
	case EditDeleteWordLeft:
		return f.deleteRange(wordLeft(runes, f.Cursor), f.Cursor, kindOther)
	case EditDeleteWordRight:
		return f.deleteRange(f.Cursor, wordRight(runes, f.Cursor), kindOther)
	case EditLeft:
		f.step(-1, extend)
	case EditRight:
		f.step(1, extend)
	case EditWordLeft:
		f.moveTo(wordLeft(runes, f.Cursor), extend)
	case EditWordRight:
		f.moveTo(wordRight(runes, f.Cursor), extend)
	case EditHome:
		f.moveTo(lineStart(runes, f.Cursor), extend)
	case EditEnd:
		f.moveTo(lineEnd(runes, f.Cursor), extend)
	case EditDocStart:
		f.moveTo(0, extend)
	case EditDocEnd:
		f.moveTo(len(runes), extend)
	case EditUp, EditDown, EditPageUp, EditPageDown:
		f.vertical(e, extend)
	case EditNewline:
		return f.Multiline && f.newline()
	case EditIndent:
		return f.Multiline && f.indent()
	case EditDedent:
		return f.Multiline && f.dedent()
	}
	return false
}

// vertical moves by lines in a multi-line field; single-line fields ignore it.
func (f *TextField) vertical(e Edit, extend bool) {
	if !f.Multiline {
		return
	}
	page := max(f.ViewLines, 1)
	dy := map[Edit]int{EditUp: -1, EditDown: 1, EditPageUp: -page, EditPageDown: page}[e]
	f.moveLines(dy, extend)
}

// deleteRange deletes the selection if there is one, else the runes in
// [lo, hi). It reports whether anything was deleted.
func (f *TextField) deleteRange(lo, hi int, kind editKind) bool {
	if _, _, ok := f.Selection(); ok {
		return f.DeleteSelection()
	}
	lo, hi = f.clamp(lo), f.clamp(hi)
	if lo >= hi {
		return false
	}
	f.record(kind)
	runes := []rune(f.Text)
	f.Text = string(append(runes[:lo], runes[hi:]...))
	f.Cursor = lo
	return true
}

// step moves one rune; without extend a selection collapses to its edge.
func (f *TextField) step(dir int, extend bool) {
	if lo, hi, ok := f.Selection(); ok && !extend {
		f.Cursor, f.HasSel = lo, false
		if dir > 0 {
			f.Cursor = hi
		}
		f.last = kindNone
		return
	}
	f.moveTo(f.Cursor+dir, extend)
}

// TabWidth is the number of spaces Tab inserts (to the next tab stop).
const TabWidth = 2

// NewMultiline returns a multi-line field holding text.
func NewMultiline(text string) TextField {
	f := NewTextField(text)
	f.Multiline = true
	return f
}

// LineCount is the number of lines in the text (at least 1).
func (f *TextField) LineCount() int { return strings.Count(f.Text, "\n") + 1 }

// lineStart is the position of the start of the line containing pos.
func lineStart(r []rune, pos int) int {
	for pos > 0 && r[pos-1] != '\n' {
		pos--
	}
	return pos
}

// lineEnd is the position of the end of the line containing pos (before its newline).
func lineEnd(r []rune, pos int) int {
	for pos < len(r) && r[pos] != '\n' {
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

// moveLines moves the cursor dy lines, keeping the goal column. Moving
// past the first or last line goes to the start or end of the text.
func (f *TextField) moveLines(dy int, extend bool) {
	line, col := f.LineCol(f.Cursor)
	if f.goal > 0 {
		col = f.goal - 1
	}
	target := line + dy
	var pos int
	switch {
	case target < 0:
		pos = 0
	case target >= f.LineCount():
		pos = len([]rune(f.Text))
	default:
		pos = f.PosAt(target, col)
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
func shiftPos(p int, starts, delta []int) int {
	out := p
	for i, s := range starts {
		if s <= p {
			out += delta[i]
			if delta[i] < 0 && p < s-delta[i] {
				out = out - (p - s) // a position inside the removed run moves to the line start
				out = s + sumBefore(delta, i)
			}
		}
	}
	return out
}

func sumBefore(delta []int, i int) int {
	n := 0
	for _, d := range delta[:i] {
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
		return f.insert(strings.Repeat(" ", TabWidth-col%TabWidth), kindOther)
	}
	return f.shiftLines(starts, func(_ []rune, _ int) int { return TabWidth })
}

// dedent removes one indent level (up to TabWidth spaces, or a tab)
// from the start of every selected line or the cursor's line.
func (f *TextField) dedent() bool {
	return f.shiftLines(f.selectedLines(), func(r []rune, start int) int {
		if start < len(r) && r[start] == '\t' {
			return -1
		}
		n := 0
		for n < TabWidth && start+n < len(r) && r[start+n] == ' ' {
			n++
		}
		return -n
	})
}

// shiftLines applies change(r, lineStart) to each line start: positive
// adds that many spaces, negative removes that many runes. The cursor
// and the selection follow the text.
func (f *TextField) shiftLines(starts []int, change func(r []rune, start int) int) bool {
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

// RunShortcuts runs the undo, redo, select-all, copy, cut and paste
// shortcuts in c.In against the field and its clipboard. It reports
// whether the text changed.
func (f *TextField) RunShortcuts(c *Ctx) (changed bool) {
	in := c.In
	if in.Undo {
		changed = f.Undo() || changed
	}
	if in.Redo {
		changed = f.Redo() || changed
	}
	if in.SelectAll {
		f.SelectAll()
	}
	if c.Clipboard == nil {
		return changed
	}
	if (in.Copy || in.Cut) && !f.Masked { // secrets never reach the clipboard
		if sel := f.SelectedText(); sel != "" {
			_ = c.Clipboard.Write(sel) // the clipboard reports its own failures
		}
	}
	if in.Cut && !f.Masked {
		changed = f.DeleteSelection() || changed
	}
	if in.Paste {
		if text, err := c.Clipboard.Read(); err == nil {
			changed = f.Insert(text) || changed
		}
	}
	return changed
}

// Display is what the field shows: its text, or bullets when Masked.
func (f *TextField) Display() string {
	if f.Masked {
		return strings.Repeat("\u2022", len([]rune(f.Text)))
	}
	return f.Text
}

// CanUndo and CanRedo report whether there is anything to undo or redo.
func (f *TextField) CanUndo() bool { return len(f.undo) > 0 }
func (f *TextField) CanRedo() bool { return len(f.redo) > 0 }
