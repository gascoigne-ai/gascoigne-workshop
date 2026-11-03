package vml

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/widget"
)

// text draws one line.
type textB struct{ plain }

func (e *El) face(c *widget.Ctx) *text.GoTextFace {
	switch {
	case e.mono:
		return c.F.Mono
	case e.small:
		return c.F.Small
	}
	return c.F.UI
}

func (e *El) color(c *widget.Ctx, def widget.Token) widget.Token {
	if e.fg != "" {
		return e.fg
	}
	return def
}

func (textB) measure(e *El, c *widget.Ctx, _ int) (int, int) {
	face := e.face(c)
	return c.Width(face, e.text), c.LineH(face)
}

func (textB) draw(e *El, c *widget.Ctx, dst *ebiten.Image) {
	face := e.face(c)
	s := e.text
	if e.ellipsis {
		s = c.Ellipsize(face, s, e.rect.W)
	}
	clr := c.Col(e.color(c, widget.Text))
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
func Color(t widget.Token) Option { return func(e *El) { e.fg = t } }

// Muted is Color(widget.Muted).
func Muted() Option { return Color(widget.Muted) }

// Small uses the small font.
func Small() Option { return func(e *El) { e.small = true } }

// Mono uses the monospace font.
func Mono() Option { return func(e *El) { e.mono = true } }

// Ellipsis shortens text that does not fit with "...".
func Ellipsis() Option { return func(e *El) { e.ellipsis = true } }

// Center centers text horizontally in its box.
func Center() Option { return func(e *El) { e.ctr = true } }

// Lines stacks the lines of s, one Text each, in the given color.
func Lines(s string, clr widget.Token, opts ...Option) *El {
	rows := []Item{}
	for _, l := range strings.Split(s, "\n") {
		rows = append(rows, Text(l, Color(clr), H(22)))
	}
	for _, o := range opts {
		rows = append(rows, o)
	}
	return Col(rows...)
}

type iconKind int

const (
	iconCheck iconKind = iota
	iconCross
	iconTriangle
	iconDots
)

// icon draws a small vector glyph centered in its box.
type iconB struct{ plain }

var iconSizes = map[iconKind][2]int{
	iconCheck: {14, 14}, iconCross: {10, 10}, iconTriangle: {10, 10}, iconDots: {4, 20},
}

func (iconB) measure(e *El, _ *widget.Ctx, _ int) (int, int) {
	s := iconSizes[e.icon]
	return s[0], s[1]
}

func (iconB) draw(e *El, c *widget.Ctx, dst *ebiten.Image) {
	r := e.rect
	cx, cy := float32(r.X+r.W/2), float32(r.Y+r.H/2)
	clr := c.Col(e.color(c, widget.Muted))
	switch e.icon {
	case iconCheck:
		drawCheckbox(c, dst, widget.Rect{X: r.X + (r.W-14)/2, Y: r.Y + (r.H-14)/2, W: 14, H: 14}, e.on, c.Hovered(r))
	case iconCross:
		if c.Hovered(r) {
			clr = c.Col(widget.Error) // a delete button turns red under the pointer
		}
		c.Cross(dst, cx, cy, 4, clr)
	case iconTriangle:
		c.Triangle(dst, cx, cy, 4, e.on, clr)
	case iconDots:
		if c.Hovered(r) {
			clr = c.Col(widget.Text)
		}
		for i := -1; i <= 1; i++ {
			c.Fill(dst, widget.Rect{X: int(cx) - 1, Y: int(cy) + i*6 - 1, W: 3, H: 3}, clr)
		}
	}
}

// drawCheckbox draws a 14px box: an accent square with a white check
// when on, a plain outline when off (darker under the pointer).
func drawCheckbox(c *widget.Ctx, dst *ebiten.Image, box widget.Rect, on, hover bool) {
	if !on {
		edge := widget.Rule
		if hover {
			edge = widget.Muted
		}
		c.Outline(dst, box, c.Col(edge))
		return
	}
	c.Fill(dst, box, c.Col(widget.Accent))
	x, y := float32(box.X), float32(box.Y)
	mark := c.Col(widget.AccentText)
	c.Line(dst, x+3.5, y+7.5, x+6, y+10, 1.8, mark)
	c.Line(dst, x+6, y+10, x+10.5, y+4.5, 1.8, mark)
}

func icon(kind iconKind, on bool, opts []Option) *El {
	e := newEl(iconB{}, with(nil, opts))
	e.icon, e.on = kind, on
	return e
}

// Checkbox is a square that shows an accent fill and a check when on.
func Checkbox(on bool, opts ...Option) *El { return icon(iconCheck, on, opts) }

// Cross is a small x, for delete buttons.
func Cross(opts ...Option) *El { return icon(iconCross, false, opts) }

// Triangle is a disclosure arrow: down when open, right when closed.
func Triangle(open bool, opts ...Option) *El { return icon(iconTriangle, open, opts) }

// Dots is the three-dot "more" glyph.
func Dots(opts ...Option) *El { return icon(iconDots, false, opts) }
