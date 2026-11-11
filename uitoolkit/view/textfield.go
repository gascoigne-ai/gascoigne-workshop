package view

import "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"

// revealMargin is the pixels of context kept visible beyond the cursor
// when the text scrolls.
const revealMargin = 12

// TextField is the state of a text input: the text being edited (the
// embedded core.TextBuffer, whose methods and fields it promotes) and
// what the input adds, focus, the pointer drag and the horizontal scroll.
type TextField struct {
	core.TextBuffer
	Focused  bool
	Dragging bool // the pointer went down inside and is still held
	ScrollX  int  // pixels the text is scrolled left inside its view
}

// NewTextField returns a field holding text with the cursor at its end.
func NewTextField(text string) TextField {
	return TextField{TextBuffer: core.NewTextBuffer(text)}
}

// Press handles the pointer going down at rune position pos: it places
// the cursor (or extends the selection with shift) and starts a drag.
func (f *TextField) Press(pos int, extend bool) {
	f.MoveTo(pos, extend)
	f.Dragging = true
}

// PressN is Press with a click count: 2 selects the word at pos, 3
// selects the line in an area and everything in a field. Neither starts
// a drag.
func (f *TextField) PressN(pos int, extend bool, clicks int) {
	switch clicks {
	case 2:
		f.SelectWordAt(pos)
		f.Dragging = false
	case 3:
		f.SelectLineAt(pos)
		f.Dragging = false
	default:
		f.Press(pos, extend)
	}
}

// Drag extends the selection to pos while the pointer is held.
func (f *TextField) Drag(pos int) { f.ExtendTo(pos) }

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
