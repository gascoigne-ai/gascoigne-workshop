package widget

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

const fieldPad = 10

// FieldPad sets a field's inner padding, for fields inset in a box that
// already has padding of its own, so text lines up with its neighbours.
func FieldPad(n int) Option { return func(e *El) { e.Behavior().(*fieldB).inset = n } }

// fieldB is a single-line text input editing a TextField.
type fieldB struct {
	Base
	field       *TextField
	placeholder string
	segments    func(string) []Span
	onEnter     func()
	onTab       func(back bool)
	inset       int // FieldPad, 0 for the default
}

// textInset is the space between a field's edge and its text: fieldPad,
// or what FieldPad set.
func (b *fieldB) textInset() int {
	if b.inset > 0 {
		return b.inset
	}
	return fieldPad
}

func (b *fieldB) FocusTarget() Focuser { return b.field }

// KeepsTab: a field with OnTab uses Tab itself.
func (b *fieldB) KeepsTab() bool { return b.onTab != nil }

func (b *fieldB) Update(e *El, c *core.Ctx) {
	f, in, r := b.field, c.In, e.Rect()
	at := func() int { return c.CursorAt(f.Display(), in.X-(r.X+b.textInset())+f.ScrollX) }
	if in.Click {
		f.Focused = r.Contains(in.X, in.Y)
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
		if onChange := e.ChangeHandler(); onChange != nil {
			onChange()
		}
	}
	if in.Enter && !in.Ctrl && b.onEnter != nil { // Ctrl+Enter is a shortcut of the app
		b.onEnter()
	}
	if in.Tab && b.onTab != nil {
		b.onTab(in.Shift)
	}
	f.Reveal(c.Width(c.F.UI, string([]rune(f.Display())[:f.Cursor])), r.W-2*b.textInset())
}

func (b *fieldB) Draw(e *El, c *core.Ctx, dst *ebiten.Image) {
	f, r := b.field, e.Rect()
	dst = c.Clip(dst, r)
	x := r.X + b.textInset() - f.ScrollX
	if lo, hi, ok := f.Selection(); ok {
		runes := []rune(f.Display())
		x0 := x + c.Width(c.F.UI, string(runes[:lo]))
		x1 := x + c.Width(c.F.UI, string(runes[:hi]))
		c.Fill(dst, core.Rect{X: x0, Y: r.Y + 6, W: x1 - x0, H: r.H - 12}, c.Color(core.Selection))
	}
	segs := []Span{{Text: f.Display(), Style: core.Text}}
	if b.segments != nil && !f.Masked {
		segs = b.segments(f.Text)
	}
	for _, s := range segs {
		c.TextMid(dst, c.F.UI, s.Text, x, r, c.Color(s.Style))
		x += c.Width(c.F.UI, s.Text)
	}
	if f.Text == "" {
		c.TextMid(dst, c.F.UI, b.placeholder, r.X+b.textInset(), r, c.Color(core.Muted))
	}
	if f.Focused && c.CaretVisible() {
		shown := []rune(f.Display())
		cx := r.X + b.textInset() - f.ScrollX + c.Width(c.F.UI, string(shown[:f.Cursor]))
		drawBlockCaret(c, dst, c.F.UI, caretGlyph(shown, f.Cursor), cx, r)
	}
}

// DrawOver is the focus ring, over the field's outline.
func (b *fieldB) DrawOver(e *El, c *core.Ctx, dst *ebiten.Image) {
	if b.field.Focused {
		c.OutlineStyled(dst, e.Rect(), c.Color(core.AccentLine), c.Th.Style(e.ControlName()).Radius)
	}
}

// Field is a single-line text input bound to f (Vue's v-model: the
// widget edits f in place).
func Field(f *TextField, opts ...Option) *El {
	return Custom(&fieldB{field: f}, with([]Item{Control("field")}, opts)...)
}

// Placeholder is the hint shown while the field is empty.
func Placeholder(s string) Option { return func(e *El) { e.Behavior().(*fieldB).placeholder = s } }

// Segments colors parts of the field's text.
func Segments(f func(text string) []Span) Option {
	return func(e *El) { e.Behavior().(*fieldB).segments = f }
}

// OnTab runs when Tab is pressed in the focused field; back is true with Shift.
func OnTab(f func(back bool)) Option { return func(e *El) { e.Behavior().(*fieldB).onTab = f } }

// OnEnter runs when Enter is pressed in the focused field.
func OnEnter(f func()) Option { return func(e *El) { e.Behavior().(*fieldB).onEnter = f } }
