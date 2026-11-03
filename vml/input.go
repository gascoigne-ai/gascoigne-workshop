package vml

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/widget"
)

// Seg is a run of field text with its own color, so a view can
// highlight parts of what the user typed.
type Seg struct {
	Text  string
	Style widget.Token
}

const fieldPad = 10

// field is a single-line text input editing a widget.TextField.
type fieldB struct{ plain }

func (fieldB) measure(*El, *widget.Ctx, int) (int, int) { return 0, 0 }

func (fieldB) update(e *El, c *widget.Ctx) {
	f, in := e.field, c.In
	at := func() int { return c.CursorAt(f.Display(), in.X-(e.rect.X+fieldPad)+f.ScrollX) }
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
	if in.Enter && e.onEnter != nil {
		e.onEnter()
	}
	if in.Tab && e.onTab != nil {
		e.onTab(in.Shift)
	}
	f.Reveal(c.Width(c.F.UI, string([]rune(f.Display())[:f.Cursor])), e.rect.W-2*fieldPad)
}

func (fieldB) draw(e *El, c *widget.Ctx, dst *ebiten.Image) {
	f, r := e.field, e.rect
	dst = c.Clip(dst, r)
	x := r.X + fieldPad - f.ScrollX
	if lo, hi, ok := f.Selection(); ok {
		runes := []rune(f.Display())
		x0 := x + c.Width(c.F.UI, string(runes[:lo]))
		x1 := x + c.Width(c.F.UI, string(runes[:hi]))
		c.Fill(dst, widget.Rect{X: x0, Y: r.Y + 6, W: x1 - x0, H: r.H - 12}, c.Col(widget.Selection))
	}
	segs := []Seg{{Text: f.Display(), Style: widget.Text}}
	if e.segments != nil && !f.Masked {
		segs = e.segments(f.Text)
	}
	for _, s := range segs {
		c.TextMid(dst, c.F.UI, s.Text, x, r, c.Col(s.Style))
		x += c.Width(c.F.UI, s.Text)
	}
	if f.Text == "" {
		c.TextMid(dst, c.F.UI, e.placeholder, r.X+fieldPad, r, c.Col(widget.Muted))
	}
	if f.Focused {
		c.Outline(dst, r, c.Col(widget.Accent))
	}
	if f.Focused && (c.Tick/30)%2 == 0 {
		cx := r.X + fieldPad - f.ScrollX + c.Width(c.F.UI, string([]rune(f.Display())[:f.Cursor]))
		c.VLine(dst, cx, r.Y+8, r.H-16, c.Col(widget.Text))
	}
}

// Field is a single-line text input bound to f (Vue's v-model: the
// widget edits f in place).
func Field(f *widget.TextField, opts ...Option) *El {
	e := newEl(fieldB{}, with(nil, opts))
	e.field = f
	return e
}

// Placeholder is the hint shown while the field is empty.
func Placeholder(s string) Option { return func(e *El) { e.placeholder = s } }

// Segments colors parts of the field's text.
func Segments(f func(text string) []Seg) Option { return func(e *El) { e.segments = f } }

// OnChange runs after the field's text was edited.
func OnChange(f func()) Option { return func(e *El) { e.onChange = f } }

// OnTab runs when Tab is pressed in the focused field; back is true with Shift.
func OnTab(f func(back bool)) Option { return func(e *El) { e.onTab = f } }

// OnEnter runs when Enter is pressed in the focused field.
func OnEnter(f func()) Option { return func(e *El) { e.onEnter = f } }

// code shows a widget.CodeView.
type codeB struct{ plain }

func (codeB) measure(*El, *widget.Ctx, int) (int, int) { return 0, 0 }
func (codeB) update(e *El, c *widget.Ctx)              { e.view.Update(c, e.rect) }
func (codeB) draw(e *El, c *widget.Ctx, dst *ebiten.Image) {
	e.view.Draw(c, dst, e.rect)
	if e.view.Focused {
		c.Outline(dst, e.rect, c.Col(widget.Accent))
	}
}

// Code is a scrollable, foldable code viewer; give it a Bg.
func Code(v *widget.CodeView, opts ...Option) *El {
	e := newEl(codeB{}, with(nil, opts))
	e.view = v
	return e
}
