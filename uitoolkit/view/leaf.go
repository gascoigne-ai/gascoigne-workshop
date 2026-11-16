package view

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// text draws one line.
type textB struct{ plain }

func (e *El) face(ctx *core.Ctx) *text.GoTextFace {
	face := e.chosenFace(ctx)
	if e.fontSource == nil {
		return face
	}
	return &text.GoTextFace{Source: e.fontSource, Size: face.Size}
}

func (e *El) chosenFace(ctx *core.Ctx) *text.GoTextFace {
	switch {
	case e.mono:
		return ctx.F.Mono
	case e.small && e.bold:
		return ctx.F.SmallBold
	case e.bold:
		return ctx.F.Bold
	case e.small:
		return ctx.F.Small
	}
	return ctx.F.UI
}

func (e *El) color(ctx *core.Ctx, def core.Token) core.Token {
	if e.fg != "" {
		return e.fg
	}
	return def
}

func (textB) measure(element *El, ctx *core.Ctx, _ int) (int, int) {
	face := element.face(ctx)
	return ctx.Width(face, element.text), ctx.LineH(face)
}

func (textB) draw(element *El, ctx *core.Ctx, dst *ebiten.Image) {
	face := element.face(ctx)
	s := element.text
	if element.ellipsis {
		s = ctx.Ellipsize(face, s, element.rect.W)
	}
	clr := ctx.Color(element.color(ctx, core.Text))
	if element.ctr {
		ctx.TextCenter(dst, face, s, element.rect, clr)
		return
	}
	ctx.TextMid(dst, face, s, element.rect.X, element.rect, clr)
}

// Text is one line of text, vertically centered in its box.
func Text(string string, opts ...Option) *El {
	e := newEl(textB{}, with(nil, opts))
	e.text = string
	return e
}

// Caption is a small uppercase muted label.
func Caption(text string, opts ...Option) *El {
	return Text(strings.ToUpper(text), append([]Option{Small(), Muted()}, opts...)...)
}

// Color sets the text (or icon) color.
func Color(token core.Token) Option { return func(element *El) { element.fg = token } }

// Muted is Color(core.Muted).
func Muted() Option { return Color(core.Muted) }

// Bold uses the bold face, for headings and table headers.
func Bold() Option { return func(element *El) { element.bold = true } }

// Small uses the small font.
func Small() Option { return func(element *El) { element.small = true } }

// Mono uses the monospace font.
func Mono() Option { return func(element *El) { element.mono = true } }

// Ellipsis shortens text that does not fit with "...".
func Ellipsis() Option { return func(element *El) { element.ellipsis = true } }

// TextCenter centers text horizontally in its box.
func TextCenter() Option { return func(element *El) { element.ctr = true } }

// Lines stacks the lines of s, one Text each, in the given color. The
// options set the column; Small among them makes every line a small text
// NoteH high, as a Note is, so a longer help takes little room.
func Lines(text string, clr core.Token, opts ...Option) *El {
	col := Col(with(nil, opts)...)
	lineOpts := []Option{Color(clr), H(core.LabelH)}
	if col.small {
		lineOpts = []Option{Color(clr), Small(), H(core.NoteH)}
	}
	for _, line := range strings.Split(text, "\n") {
		col.kids = append(col.kids, Text(line, lineOpts...))
	}
	return col
}
