package view

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

const fieldPad = 10

// textInset is the space between a field's edge and its text: fieldPad,
// or what FieldPad set.
func (e *El) textInset() int {
	if e.fieldInset > 0 {
		return e.fieldInset
	}
	return fieldPad
}

// FieldPad sets a field's inner padding, for fields inset in a box that
// already has padding of its own, so text lines up with its neighbours.
func FieldPad(n int) Option { return func(e *El) { e.fieldInset = n } }

// field is a single-line text input editing a TextField.
type fieldB struct{ plain }

func (fieldB) measure(*El, *core.Ctx, int) (int, int) { return 0, 0 }

func (fieldB) update(e *El, c *core.Ctx) {
	f, in := e.field, c.In
	at := func() int { return c.CursorAt(f.Display(), in.X-(e.rect.X+e.textInset())+f.ScrollX) }
	if in.Click {
		f.Focused = e.rect.Contains(in.X, in.Y)
		if f.Focused {
			f.PressN(at(), in.Shift, max(in.Clicks, 1))
		}
	}
	if f.Dragging {
		if in.Held {
			f.Drag(at())
		} else {
			f.Dragging = false
		}
	}
	if !f.Focused {
		return
	}
	changed := f.RunShortcuts(c)
	typed := in.Typed
	if in.Ctrl {
		typed = nil // Ctrl+letter is a shortcut, not text
	}
	if f.Apply(typed, in.Edits, in.Shift) || changed {
		if e.onChange != nil {
			e.onChange()
		}
	}
	if in.Enter && !in.Ctrl && e.onEnter != nil { // Ctrl+Enter is a shortcut of the app
		e.onEnter()
	}
	if in.Tab && e.onTab != nil {
		e.onTab(in.Shift)
	}
	f.Reveal(c.Width(c.F.UI, string([]rune(f.Display())[:f.Cursor])), e.rect.W-2*e.textInset())
}

func (fieldB) draw(e *El, c *core.Ctx, dst *ebiten.Image) {
	f, r := e.field, e.rect
	dst = c.Clip(dst, r)
	x := r.X + e.textInset() - f.ScrollX
	if lo, hi, ok := f.Selection(); ok {
		runes := []rune(f.Display())
		x0 := x + c.Width(c.F.UI, string(runes[:lo]))
		x1 := x + c.Width(c.F.UI, string(runes[:hi]))
		c.Fill(dst, core.Rect{X: x0, Y: r.Y + 6, W: x1 - x0, H: r.H - 12}, c.Color(core.Selection))
	}
	segs := []Span{{Text: f.Display(), Style: core.Text}}
	if e.segments != nil && !f.Masked {
		segs = e.segments(f.Text)
	}
	for _, s := range segs {
		c.TextMid(dst, c.F.UI, s.Text, x, r, c.Color(s.Style))
		x += c.Width(c.F.UI, s.Text)
	}
	if f.Text == "" {
		c.TextMid(dst, c.F.UI, e.placeholder, r.X+e.textInset(), r, c.Color(core.Muted))
	}
	if f.Focused && c.CaretVisible() {
		shown := []rune(f.Display())
		cx := r.X + e.textInset() - f.ScrollX + c.Width(c.F.UI, string(shown[:f.Cursor]))
		c.DrawBlockCaret(dst, c.F.UI, shown, f.Cursor, cx, r)
	}
}

// drawOver is the focus ring, over the field's outline.
func (fieldB) drawOver(e *El, c *core.Ctx, dst *ebiten.Image) {
	if e.field.Focused {
		c.OutlineStyled(dst, e.rect, c.Color(core.AccentLine), c.Th.Style(e.control).Radius)
	}
}

// Field is a single-line text input bound to f (Vue's v-model: the
// widget edits f in place).
func Field(f *TextField, opts ...Option) *El {
	e := newEl(fieldB{}, with([]Item{Control("field")}, opts))
	e.field = f
	return e
}

// Placeholder is the hint shown while the field is empty.
func Placeholder(s string) Option { return func(e *El) { e.placeholder = s } }

// Segments colors parts of the field's text.
func Segments(f func(text string) []Span) Option { return func(e *El) { e.segments = f } }

// OnChange runs after the field's text was edited.
func OnChange(f func()) Option { return func(e *El) { e.onChange = f } }

// OnTab runs when Tab is pressed in the focused field; back is true with Shift.
func OnTab(f func(back bool)) Option { return func(e *El) { e.onTab = f } }

// OnEnter runs when Enter is pressed in the focused field.
func OnEnter(f func()) Option { return func(e *El) { e.onEnter = f } }

// code shows a CodeView.
type codeB struct{ plain }

func (codeB) measure(*El, *core.Ctx, int) (int, int) { return 0, 0 }
func (codeB) update(e *El, c *core.Ctx)              { e.view.Update(c, e.rect) }
func (codeB) draw(e *El, c *core.Ctx, dst *ebiten.Image) {
	e.view.Draw(c, dst, e.rect)
	if e.view.Focused {
		c.Outline(dst, e.rect, c.Color(core.AccentLine))
	}
}

// Code is a scrollable, foldable code viewer; give it a Bg.
func Code(v *CodeView, opts ...Option) *El {
	e := newEl(codeB{}, with(nil, opts))
	e.view = v
	return e
}
