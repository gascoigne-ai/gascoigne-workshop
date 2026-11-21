package view

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// text draws one line.
type textB struct{ Base }

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

func (e *El) color(def core.Token) core.Token {
	if e.fg != "" {
		return e.fg
	}
	return def
}

func (textB) Measure(element *El, ctx *core.Ctx, _ int) (int, int) {
	face := element.face(ctx)
	return ctx.Width(face, element.text), ctx.LineH(face)
}

func (textB) Draw(element *El, ctx *core.Ctx, dst *ebiten.Image) {
	face := element.face(ctx)
	s := element.text
	if element.ellipsis {
		s = ctx.Ellipsize(face, s, element.rect.W)
	}
	clr := ctx.Color(element.color(core.Text))
	if element.ctr {
		ctx.TextCenter(dst, face, s, element.rect, clr)
		return
	}
	ctx.TextMid(dst, face, s, element.rect.X, element.rect, clr)
}

// Text is one line of text, vertically centered in its box.
func Text(content string, opts ...Option) *El {
	e := newEl(textB{}, With(nil, opts))
	e.text = content
	return e
}

// Caption is a small uppercase muted label.
func Caption(content string, opts ...Option) *El {
	return Text(strings.ToUpper(content), append([]Option{Small(), Muted()}, opts...)...)
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

// Lines stacks the lines of text, one Text each, in the given color. The
// options set the column; Small among them makes every line a small text
// NoteH high, as a Note is, so a longer help takes little room.
func Lines(content string, clr core.Token, opts ...Option) *El {
	col := Col(With(nil, opts)...)
	lineOpts := []Option{Color(clr), HOf(core.LabelOf)}
	if col.small {
		lineOpts = []Option{Color(clr), Small(), HOf(core.NoteOf)}
	}
	for _, line := range strings.Split(content, "\n") {
		col.kids = append(col.kids, Text(line, lineOpts...))
	}
	return col
}
