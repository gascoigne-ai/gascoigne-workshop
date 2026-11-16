package widget

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

const fieldPad = 10

// FieldPad sets a field's inner padding, for fields inset in a box that
// already has padding of its own, so text lines up with its neighbours.
func FieldPad(count int) Option {
	return func(element *El) { element.Behavior().(*fieldB).inset = count }
}

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

func (b *fieldB) Update(element *El, ctx *core.Ctx) {
	f, in, r := b.field, ctx.In, element.Rect()
	at := func() int { return ctx.CursorAt(f.Display(), in.X-(r.X+b.textInset())+f.ScrollX) }
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
	changed := f.RunShortcuts(ctx)
	typed := in.Typed
	if in.Ctrl {
		typed = nil // Ctrl+letter is a shortcut, not text
	}
	if f.Apply(typed, in.Edits, in.Shift) || changed {
		if onChange := element.ChangeHandler(); onChange != nil {
			onChange()
		}
	}
	if in.Enter && !in.Ctrl && b.onEnter != nil { // Ctrl+Enter is a shortcut of the app
		b.onEnter()
	}
	if in.Tab && b.onTab != nil {
		b.onTab(in.Shift)
	}
	f.Cursor = f.clamp(f.Cursor) // OnChange may have replaced the text
	f.Reveal(ctx.Width(ctx.F.UI, string([]rune(f.Display())[:f.Cursor])), r.W-2*b.textInset())
}

func (b *fieldB) Draw(element *El, ctx *core.Ctx, dst *ebiten.Image) {
	f, r := b.field, element.Rect()
	dst = ctx.Clip(dst, r)
	x := r.X + b.textInset() - f.ScrollX
	if lo, hi, ok := f.Selection(); ok {
		runes := []rune(f.Display())
		x0 := x + ctx.Width(ctx.F.UI, string(runes[:lo]))
		x1 := x + ctx.Width(ctx.F.UI, string(runes[:hi]))
		ctx.Fill(dst, core.Rect{X: x0, Y: r.Y + 6, W: x1 - x0, H: r.H - 12}, ctx.Color(core.Selection))
	}
	segs := []Span{{Text: f.Display(), Style: core.Text}}
	if b.segments != nil && !f.Masked {
		segs = b.segments(f.Text)
	}
	for _, s := range segs {
		ctx.TextMid(dst, ctx.F.UI, s.Text, x, r, ctx.Color(s.Style))
		x += ctx.Width(ctx.F.UI, s.Text)
	}
	if f.Text == "" {
		ctx.TextMid(dst, ctx.F.UI, b.placeholder, r.X+b.textInset(), r, ctx.Color(core.Muted))
	}
	if f.Focused && ctx.CaretVisible() {
		shown, cursor := []rune(f.Display()), f.clamp(f.Cursor)
		cx := r.X + b.textInset() - f.ScrollX + ctx.Width(ctx.F.UI, string(shown[:cursor]))
		drawBlockCaret(ctx, dst, ctx.F.UI, caretGlyph(shown, cursor), cx, r)
	}
}

// DrawOver is the focus ring, over the field's outline.
func (b *fieldB) DrawOver(element *El, ctx *core.Ctx, dst *ebiten.Image) {
	if b.field.Focused {
		ctx.OutlineStyled(dst, element.Rect(), ctx.Color(core.AccentLine), ctx.Th.Style(element.ControlName()).Radius)
	}
}

// Field is a single-line text input bound to f (Vue's v-model: the
// widget edits f in place).
func Field(textField *TextField, opts ...Option) *El {
	return Custom(&fieldB{field: textField}, with([]Item{Control("field")}, opts)...)
}

// Placeholder is the hint shown while the field is empty.
func Placeholder(text string) Option {
	return func(element *El) { element.Behavior().(*fieldB).placeholder = text }
}

// Segments colors parts of the field's text.
func Segments(callback func(text string) []Span) Option {
	return func(element *El) { element.Behavior().(*fieldB).segments = callback }
}

// OnTab runs when Tab is pressed in the focused field; back is true with Shift.
func OnTab(callback func(back bool)) Option {
	return func(element *El) { element.Behavior().(*fieldB).onTab = callback }
}

// OnEnter runs when Enter is pressed in the focused field.
func OnEnter(callback func()) Option {
	return func(element *El) { element.Behavior().(*fieldB).onEnter = callback }
}
