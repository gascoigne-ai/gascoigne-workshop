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
	if in.Click {
		f.Focused = e.rect.Contains(in.X, in.Y)
		if f.Focused {
			f.Cursor = c.CursorAt(f.Text, in.X-(e.rect.X+fieldPad))
		}
	}
	if !f.Focused {
		return
	}
	if f.Apply(in.Typed, in.Edits) && e.onChange != nil {
		e.onChange()
	}
	if in.Enter && e.onEnter != nil {
		e.onEnter()
	}
}

func (fieldB) draw(e *El, c *widget.Ctx, dst *ebiten.Image) {
	f, r := e.field, e.rect
	dst = c.Clip(dst, r)
	x := r.X + fieldPad
	segs := []Seg{{Text: f.Text, Style: widget.Text}}
	if e.segments != nil {
		segs = e.segments(f.Text)
	}
	for _, s := range segs {
		c.TextMid(dst, c.F.UI, s.Text, x, r, c.Col(s.Style))
		x += c.Width(c.F.UI, s.Text)
	}
	if f.Text == "" {
		c.TextMid(dst, c.F.UI, e.placeholder, x, r, c.Col(widget.Muted))
	}
	if f.Focused && (c.Tick/30)%2 == 0 {
		cx := r.X + fieldPad + c.Width(c.F.UI, string([]rune(f.Text)[:f.Cursor]))
		c.VLine(dst, cx, r.Y+8, r.H-16, c.Col(widget.Text))
	}
}

// Field is a single-line text input bound to f (Vue's v-model: the
// widget edits f in place).
func Field(f *widget.TextField) *El {
	e := newEl(fieldB{}, nil)
	e.field = f
	return e
}

// Placeholder is the hint shown while the field is empty.
func (e *El) Placeholder(s string) *El { e.placeholder = s; return e }

// Segments colors parts of the field's text.
func (e *El) Segments(f func(text string) []Seg) *El { e.segments = f; return e }

// OnChange runs after the field's text was edited.
func (e *El) OnChange(f func()) *El { e.onChange = f; return e }

// OnEnter runs when Enter is pressed in the focused field.
func (e *El) OnEnter(f func()) *El { e.onEnter = f; return e }

// code shows a widget.CodeView.
type codeB struct{ plain }

func (codeB) measure(*El, *widget.Ctx, int) (int, int) { return 0, 0 }
func (codeB) update(e *El, c *widget.Ctx)              { e.view.Update(c, e.rect) }
func (codeB) draw(e *El, c *widget.Ctx, dst *ebiten.Image) {
	e.view.Draw(c, dst, e.rect)
}

// Code is a scrollable, foldable code viewer; give it a Bg.
func Code(v *widget.CodeView) *El {
	e := newEl(codeB{}, nil)
	e.view = v
	return e
}
