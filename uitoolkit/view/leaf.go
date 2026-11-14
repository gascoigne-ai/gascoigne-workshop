package view

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// text draws one line.
type textB struct{ plain }

func (e *El) face(c *core.Ctx) *text.GoTextFace {
	face := e.chosenFace(c)
	if e.fontSource == nil {
		return face
	}
	return &text.GoTextFace{Source: e.fontSource, Size: face.Size}
}

func (e *El) chosenFace(c *core.Ctx) *text.GoTextFace {
	switch {
	case e.mono:
		return c.F.Mono
	case e.small && e.bold:
		return c.F.SmallBold
	case e.bold:
		return c.F.Bold
	case e.small:
		return c.F.Small
	}
	return c.F.UI
}

func (e *El) color(c *core.Ctx, def core.Token) core.Token {
	if e.fg != "" {
		return e.fg
	}
	return def
}

func (textB) measure(e *El, c *core.Ctx, _ int) (int, int) {
	face := e.face(c)
	return c.Width(face, e.text), c.LineH(face)
}

func (textB) draw(e *El, c *core.Ctx, dst *ebiten.Image) {
	face := e.face(c)
	s := e.text
	if e.ellipsis {
		s = c.Ellipsize(face, s, e.rect.W)
	}
	clr := c.Color(e.color(c, core.Text))
	if e.ctr {
		c.TextCenter(dst, face, s, e.rect, clr)
		return
	}
	c.TextMid(dst, face, s, e.rect.X, e.rect, clr)
}

// Text is one line of text, vertically centered in its box.
func Text(s string, opts ...Option) *El {
	e := newEl(textB{}, with(nil, opts))
	e.text = s
	return e
}

// Caption is a small uppercase muted label.
func Caption(s string, opts ...Option) *El {
	return Text(strings.ToUpper(s), append([]Option{Small(), Muted()}, opts...)...)
}

// Color sets the text (or icon) color.
func Color(t core.Token) Option { return func(e *El) { e.fg = t } }

// Muted is Color(core.Muted).
func Muted() Option { return Color(core.Muted) }

// Bold uses the bold face, for headings and table headers.
func Bold() Option { return func(e *El) { e.bold = true } }

// Small uses the small font.
func Small() Option { return func(e *El) { e.small = true } }

// Mono uses the monospace font.
func Mono() Option { return func(e *El) { e.mono = true } }

// Ellipsis shortens text that does not fit with "...".
func Ellipsis() Option { return func(e *El) { e.ellipsis = true } }

// TextCenter centers text horizontally in its box.
func TextCenter() Option { return func(e *El) { e.ctr = true } }

// Lines stacks the lines of s, one Text each, in the given color. The
// options set the column; Small among them makes every line a small text
// NoteH high, as a Note is, so a longer help takes little room.
func Lines(s string, clr core.Token, opts ...Option) *El {
	col := Col(with(nil, opts)...)
	lineOpts := []Option{Color(clr), H(core.LabelH)}
	if col.small {
		lineOpts = []Option{Color(clr), Small(), H(core.NoteH)}
	}
	for _, line := range strings.Split(s, "\n") {
		col.kids = append(col.kids, Text(line, lineOpts...))
	}
	return col
}
