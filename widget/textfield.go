package widget

import "strings"

// Edit is a cursor or deletion key applied to a TextField.
type Edit int

const (
	EditBackspace Edit = iota
	EditDelete
	EditLeft
	EditRight
	EditHome
	EditEnd
)

// TextField is a single-line editable string with a cursor and an
// optional selection. Positions count runes.
type TextField struct {
	Text     string
	Cursor   int
	Anchor   int  // the fixed end of the selection; meaningful when HasSel
	HasSel   bool // a selection exists between Anchor and Cursor
	Focused  bool
	Dragging bool // the pointer went down inside and is still held
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
}

// DeleteSelection removes the selected text and reports whether there was any.
func (f *TextField) DeleteSelection() bool {
	lo, hi, ok := f.Selection()
	if !ok {
		f.HasSel = false
		return false
	}
	runes := []rune(f.Text)
	f.Text = string(append(runes[:lo], runes[hi:]...))
	f.Cursor, f.HasSel = lo, false
	return true
}

// Insert replaces the selection (if any) with s at the cursor. Line
// breaks and other control characters are dropped, since the field has
// one line. It reports whether the text changed.
func (f *TextField) Insert(s string) bool {
	clean := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}
		if r < ' ' || r == 0x7f {
			return -1
		}
		return r
	}, s)
	changed := f.DeleteSelection()
	if clean == "" {
		return changed
	}
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
}

// Press handles the pointer going down at rune position pos: it places
// the cursor (or extends the selection with shift) and starts a drag.
func (f *TextField) Press(pos int, extend bool) {
	f.moveTo(pos, extend)
	f.Dragging = true
}

// Drag extends the selection to pos while the pointer is held.
func (f *TextField) Drag(pos int) {
	if !f.HasSel {
		f.Anchor, f.HasSel = f.Cursor, true
	}
	f.Cursor = f.clamp(pos)
}

// Apply inserts typed runes at the cursor, then runs edits in order.
// With extend (shift held) the movement keys select instead of just
// moving. It reports whether Text changed.
func (f *TextField) Apply(typed []rune, edits []Edit, extend bool) (changed bool) {
	f.Cursor = f.clamp(f.Cursor)
	if printable := keepPrintable(typed); len(printable) > 0 {
		changed = f.Insert(string(printable)) || changed
	}
	for _, e := range edits {
		changed = f.edit(e, extend) || changed
	}
	return changed
}

func (f *TextField) edit(e Edit, extend bool) (changed bool) {
	n := len([]rune(f.Text))
	switch e {
	case EditBackspace:
		if f.DeleteSelection() {
			return true
		}
		if f.Cursor > 0 {
			runes := []rune(f.Text)
			f.Text = string(append(runes[:f.Cursor-1], runes[f.Cursor:]...))
			f.Cursor--
			return true
		}
	case EditDelete:
		if f.DeleteSelection() {
			return true
		}
		if f.Cursor < n {
			runes := []rune(f.Text)
			f.Text = string(append(runes[:f.Cursor], runes[f.Cursor+1:]...))
			return true
		}
	case EditLeft:
		f.step(-1, extend)
	case EditRight:
		f.step(1, extend)
	case EditHome:
		f.moveTo(0, extend)
	case EditEnd:
		f.moveTo(n, extend)
	}
	return false
}

// step moves one rune; without extend a selection collapses to its edge.
func (f *TextField) step(dir int, extend bool) {
	if lo, hi, ok := f.Selection(); ok && !extend {
		f.Cursor, f.HasSel = lo, false
		if dir > 0 {
			f.Cursor = hi
		}
		return
	}
	f.moveTo(f.Cursor+dir, extend)
}
