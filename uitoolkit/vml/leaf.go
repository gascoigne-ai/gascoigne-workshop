package vml

import (
	"image/color"
	"math"

	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// text draws one line.
type textB struct{ plain }

func (e *El) face(c *widget.Ctx) *text.GoTextFace {
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

// Bold uses the bold face, for headings and table headers.
func Bold() Option { return func(e *El) { e.bold = true } }

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
		rows = append(rows, Text(l, Color(clr), H(widget.LabelH)))
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
	iconTrash
	iconCopy
	iconSearch
	iconPlus
	iconPrompt
	iconUp
	iconDown
)

// icon draws a small vector glyph centered in its box.
type iconB struct{ plain }

var iconSizes = map[iconKind][2]int{
	iconCheck: {14, 14}, iconCross: {10, 10}, iconTriangle: {10, 10}, iconDots: {4, 20},
	iconTrash: {14, 16}, iconCopy: {14, 14}, iconSearch: {14, 14}, iconPlus: {12, 12}, iconPrompt: {16, 14}, iconUp: {12, 8}, iconDown: {12, 8},
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
	case iconTrash, iconCopy, iconSearch, iconPlus, iconPrompt, iconUp, iconDown:
		if c.Hovered(r) {
			clr = c.Col(glyphHover[e.icon])
		}
		glyphs[e.icon](c, dst, cx, cy, clr)
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
		c.OutlineStyled(dst, box, c.Col(edge), c.Radius("checkbox"))
		return
	}
	accent := c.Col(widget.Accent)
	c.FillStyled(dst, box, accent, accent, c.Radius("checkbox"))
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

// glyphHover is the color a glyph takes under the pointer: red for the
// destructive one, the text color for the rest.
var glyphHover = map[iconKind]widget.Token{
	iconTrash: widget.Error, iconCopy: widget.Text, iconSearch: widget.Text, iconPlus: widget.Text, iconPrompt: widget.Text,
	iconUp: widget.Text, iconDown: widget.Text,
}

// glyphs draw the line icons around a center point, in one stroke width.
var glyphs = map[iconKind]func(c *widget.Ctx, dst *ebiten.Image, cx, cy float32, clr color.RGBA){
	iconTrash: func(c *widget.Ctx, dst *ebiten.Image, cx, cy float32, clr color.RGBA) {
		c.Line(dst, cx-6, cy-4.5, cx+6, cy-4.5, 1.4, clr) // lid
		c.Line(dst, cx-2, cy-6.5, cx+2, cy-6.5, 1.4, clr) // handle
		c.Line(dst, cx-2, cy-6.5, cx-2, cy-4.5, 1.4, clr)
		c.Line(dst, cx+2, cy-6.5, cx+2, cy-4.5, 1.4, clr)
		c.Line(dst, cx-4.5, cy-3, cx-3.7, cy+6.5, 1.4, clr) // body
		c.Line(dst, cx+4.5, cy-3, cx+3.7, cy+6.5, 1.4, clr)
		c.Line(dst, cx-3.7, cy+6.5, cx+3.7, cy+6.5, 1.4, clr)
		c.Line(dst, cx-1.5, cy-1, cx-1.5, cy+4.5, 1.2, clr) // slats
		c.Line(dst, cx+1.5, cy-1, cx+1.5, cy+4.5, 1.2, clr)
	},
	iconCopy: func(c *widget.Ctx, dst *ebiten.Image, cx, cy float32, clr color.RGBA) {
		c.Line(dst, cx-6, cy-6, cx+2, cy-6, 1.4, clr) // back sheet: top and left edges
		c.Line(dst, cx-6, cy-6, cx-6, cy+2, 1.4, clr)
		x0, y0, x1, y1 := cx-3, cy-3, cx+6, cy+6 // front sheet
		c.Line(dst, x0, y0, x1, y0, 1.4, clr)
		c.Line(dst, x1, y0, x1, y1, 1.4, clr)
		c.Line(dst, x1, y1, x0, y1, 1.4, clr)
		c.Line(dst, x0, y1, x0, y0, 1.4, clr)
	},
	iconSearch: func(c *widget.Ctx, dst *ebiten.Image, cx, cy float32, clr color.RGBA) {
		const n = 14
		ox, oy, r := cx-1.5, cy-1.5, float32(4.5)
		px, py := ox+r, oy
		for i := 1; i <= n; i++ {
			a := float64(i) * 2 * math.Pi / n
			x, y := ox+r*float32(math.Cos(a)), oy+r*float32(math.Sin(a))
			c.Line(dst, px, py, x, y, 1.4, clr)
			px, py = x, y
		}
		c.Line(dst, cx+1.8, cy+1.8, cx+6, cy+6, 1.8, clr) // handle
	},
	iconPlus: func(c *widget.Ctx, dst *ebiten.Image, cx, cy float32, clr color.RGBA) {
		c.Line(dst, cx-5, cy, cx+5, cy, 1.5, clr)
		c.Line(dst, cx, cy-5, cx, cy+5, 1.5, clr)
	},
	iconUp: func(c *widget.Ctx, dst *ebiten.Image, cx, cy float32, clr color.RGBA) {
		c.Line(dst, cx-5, cy+2.5, cx, cy-2.5, 1.5, clr)
		c.Line(dst, cx, cy-2.5, cx+5, cy+2.5, 1.5, clr)
	},
	iconDown: func(c *widget.Ctx, dst *ebiten.Image, cx, cy float32, clr color.RGBA) {
		c.Line(dst, cx-5, cy-2.5, cx, cy+2.5, 1.5, clr)
		c.Line(dst, cx, cy+2.5, cx+5, cy-2.5, 1.5, clr)
	},
	iconPrompt: func(c *widget.Ctx, dst *ebiten.Image, cx, cy float32, clr color.RGBA) {
		c.Line(dst, cx-7, cy-4, cx-2.5, cy, 1.5, clr) // the > chevron
		c.Line(dst, cx-2.5, cy, cx-7, cy+4, 1.5, clr)
		c.Line(dst, cx, cy+5, cx+7, cy+5, 1.5, clr) // the _ cursor
	},
}

// Trash is a trash can, for delete buttons; it turns red under the pointer.
func Trash(opts ...Option) *El { return icon(iconTrash, false, opts) }

// CopyIcon is two overlapping sheets.
func CopyIcon(opts ...Option) *El { return icon(iconCopy, false, opts) }

// SearchIcon is a magnifying glass.
func SearchIcon(opts ...Option) *El { return icon(iconSearch, false, opts) }

// Plus is a plus sign.
func Plus(opts ...Option) *El { return icon(iconPlus, false, opts) }

// PromptIcon is a terminal prompt, ">_".
func PromptIcon(opts ...Option) *El { return icon(iconPrompt, false, opts) }

// ChevronUp and ChevronDown are small arrows for stepping through a list.
func ChevronUp(opts ...Option) *El   { return icon(iconUp, false, opts) }
func ChevronDown(opts ...Option) *El { return icon(iconDown, false, opts) }

// IconButton is Button with an icon in place of the label.
func IconButton(glyph *El, opts ...Option) *El {
	return Row(with([]Item{Bg(widget.Sunken), Control("icon-button"), Lighten(), glyph}, opts)...)
}
