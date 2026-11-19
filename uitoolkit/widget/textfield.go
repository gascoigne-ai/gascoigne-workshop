package widget

import (
	"strings"
)

// Pixels of context kept visible beyond the cursor when the text
// scrolls, and the number of undo steps kept.
const (
	revealMargin = 12
	historyLimit = 100
	historyBytes = 8 << 20 // the snapshots of a large text are bounded by size too
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

func (f *TextField) clamp(count int) int { return min(max(count, 0), len([]rune(f.Text))) }

// Selection returns the selected range, low to high; ok is false when
// nothing is selected. The range is kept inside the text, since the text
// can be replaced from outside while Anchor still points into the old one.
func (f *TextField) Selection() (lo, hi int, ok bool) {
	anchor, cursor := f.clamp(f.Anchor), f.clamp(f.Cursor)
	if !f.HasSel || anchor == cursor {
		return 0, 0, false
	}
	return min(anchor, cursor), max(anchor, cursor), true
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

// Display is what the field shows: its text, or bullets when Masked.
func (f *TextField) Display() string {
	if f.Masked {
		return strings.Repeat("\u2022", len([]rune(f.Text)))
	}
	return f.Text
}

// CanUndo reports whether there is anything to undo.
func (f *TextField) CanUndo() bool { return len(f.undo) > 0 }

// CanRedo reports whether there is anything to redo.
func (f *TextField) CanRedo() bool { return len(f.redo) > 0 }

// HasFocus reports whether the field has the keyboard focus.
func (f *TextField) HasFocus() bool { return f.Focused }

// SetFocus gives or takes the keyboard focus; gaining it selects all the text.
func (f *TextField) SetFocus(on bool) {
	f.Focused = on
	if on {
		f.SelectAll()
	}
}
