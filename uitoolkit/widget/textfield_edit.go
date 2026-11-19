package widget

import (
	"strings"
	"unicode"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

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
func (f *TextField) Insert(text string) bool { return f.insert(text, kindOther) }

func (f *TextField) insert(text string, kind editKind) bool {
	if f.Multiline {
		text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	}
	clean := strings.Map(func(char rune) rune {
		if f.Multiline && (char == '\n' || char == '\t') {
			return char
		}
		if char == '\n' || char == '\r' || char == '\t' {
			return ' '
		}
		if char < ' ' || char == 0x7f {
			return -1
		}
		return char
	}, text)
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

// isWordRune reports whether char belongs to a word: a letter of any
// script, a digit, an underscore, or a combining mark, so a letter written
// with a separate accent is not split from it. Everything else
// (punctuation, spaces, line breaks) separates words.
func isWordRune(char rune) bool {
	return unicode.IsLetter(char) || unicode.IsMark(char) || unicode.IsDigit(char) || char == '_'
}

// wordLeft is the start of the word before pos (skipping separators).
func wordLeft(runes []rune, pos int) int {
	i := min(max(pos, 0), len(runes))
	for i > 0 && !isWordRune(runes[i-1]) {
		i--
	}
	for i > 0 && isWordRune(runes[i-1]) {
		i--
	}
	return i
}

// wordRight is the end of the word after pos (skipping separators).
func wordRight(runes []rune, pos int) int {
	i := min(max(pos, 0), len(runes))
	for i < len(runes) && !isWordRune(runes[i]) {
		i++
	}
	for i < len(runes) && isWordRune(runes[i]) {
		i++
	}
	return i
}

// wordAt is the range of the run of word characters, or of separator
// characters, at pos (the character to its right, else to its left).
func wordAt(runes []rune, pos int) (lo, hi int) {
	if len(runes) == 0 {
		return 0, 0
	}
	at := min(pos, len(runes)-1)
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

// Apply inserts typed runes at the cursor, then runs edits in order.
// With extend (shift held) the movement keys select instead of just
// moving. It reports whether Text changed.
func (f *TextField) Apply(typed []rune, edits []core.Edit, extend bool) (changed bool) {
	f.Cursor = f.clamp(f.Cursor)
	if printable := keepPrintable(typed); len(printable) > 0 {
		changed = f.insert(string(printable), kindType) || changed
	}
	for _, e := range edits {
		changed = f.edit(e, extend) || changed
	}
	return changed
}

func (f *TextField) edit(edit core.Edit, extend bool) (changed bool) {
	runes := []rune(f.Text)
	switch edit {
	case core.EditBackspace:
		return f.deleteRange(f.Cursor-1, f.Cursor, kindDelete)
	case core.EditDelete:
		return f.deleteRange(f.Cursor, f.Cursor+1, kindDelete)
	case core.EditDeleteWordLeft:
		return f.deleteRange(wordLeft(runes, f.Cursor), f.Cursor, kindOther)
	case core.EditDeleteWordRight:
		return f.deleteRange(f.Cursor, wordRight(runes, f.Cursor), kindOther)
	case core.EditDeleteToLineStart:
		// At the line start the range would be empty; taking the line
		// break joins the lines, as the macOS editors do.
		lo := lineStart(runes, f.Cursor)
		if lo == f.Cursor {
			lo--
		}
		return f.deleteRange(lo, f.Cursor, kindOther)
	case core.EditDeleteToLineEnd:
		hi := lineEnd(runes, f.Cursor)
		if hi == f.Cursor {
			hi++
		}
		return f.deleteRange(f.Cursor, hi, kindOther)
	case core.EditLeft:
		f.step(-1, extend)
	case core.EditRight:
		f.step(1, extend)
	case core.EditWordLeft:
		f.moveTo(wordLeft(runes, f.Cursor), extend)
	case core.EditWordRight:
		f.moveTo(wordRight(runes, f.Cursor), extend)
	case core.EditHome:
		f.moveTo(lineStart(runes, f.Cursor), extend)
	case core.EditEnd:
		f.moveTo(lineEnd(runes, f.Cursor), extend)
	case core.EditDocStart:
		f.moveTo(0, extend)
	case core.EditDocEnd:
		f.moveTo(len(runes), extend)
	case core.EditUp, core.EditDown, core.EditPageUp, core.EditPageDown:
		f.vertical(edit, extend)
	case core.EditNewline:
		return f.Multiline && f.newline()
	case core.EditIndent:
		return f.Multiline && f.indent()
	case core.EditDedent:
		return f.Multiline && f.dedent()
	}
	return false
}

// vertical moves by lines in a multi-line field; single-line fields ignore it.
func (f *TextField) vertical(edit core.Edit, extend bool) {
	if !f.Multiline {
		return
	}
	page := max(f.ViewLines, 1)
	dy := map[core.Edit]int{core.EditUp: -1, core.EditDown: 1, core.EditPageUp: -page, core.EditPageDown: page}[edit]
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

// RunShortcuts runs the undo, redo, select-all, copy, cut and paste
// shortcuts in c.In against the field and its clipboard. It reports
// whether the text changed.
func (f *TextField) RunShortcuts(ctx *core.Ctx) (changed bool) {
	in := ctx.In
	if in.Undo {
		changed = f.Undo() || changed
	}
	if in.Redo {
		changed = f.Redo() || changed
	}
	if in.SelectAll {
		f.SelectAll()
	}
	if ctx.Clipboard == nil {
		return changed
	}
	if (in.Copy || in.Cut) && !f.Masked { // secrets never reach the clipboard
		if sel := f.SelectedText(); sel != "" {
			_ = ctx.Clipboard.Write(sel) // the clipboard reports its own failures
		}
	}
	if in.Cut && !f.Masked {
		changed = f.DeleteSelection() || changed
	}
	if in.Paste {
		if text, err := ctx.Clipboard.Read(); err == nil {
			changed = f.Insert(text) || changed
		}
	}
	return changed
}
