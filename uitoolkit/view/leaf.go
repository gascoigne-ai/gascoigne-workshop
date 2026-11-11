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

// Lines stacks the lines of s, one Text each, in the given color.
func Lines(s string, clr core.Token, opts ...Option) *El {
	rows := []Item{}
	for _, l := range strings.Split(s, "\n") {
		rows = append(rows, Text(l, Color(clr), H(core.LabelH)))
	}
	for _, o := range opts {
		rows = append(rows, o)
	}
	return Col(rows...)
}

// checkB is the checkbox: a 14px box, an accent square with a check when
// on, a plain outline when off.
type checkB struct{ plain }

func (checkB) measure(*El, *core.Ctx, int) (int, int) { return 14, 14 }

// update runs OnChange when the box is clicked; the caller owns the value.
func (checkB) update(e *El, c *core.Ctx) {
	if e.onChange != nil && c.In.Hit(e.rect) {
		e.onChange()
	}
}

func (checkB) draw(e *El, c *core.Ctx, dst *ebiten.Image) {
	r := e.rect
	drawCheckbox(c, dst, core.Rect{X: r.X + (r.W-14)/2, Y: r.Y + (r.H-14)/2, W: 14, H: 14}, e.on, c.Hovered(r))
}

// drawCheckbox draws a 14px box: an accent square with a white check
// when on, a plain outline when off (darker under the pointer).
func drawCheckbox(c *core.Ctx, dst *ebiten.Image, box core.Rect, on, hover bool) {
	if !on {
		edge := core.Rule
		if hover {
			edge = core.Muted
		}
		c.OutlineStyled(dst, box, c.Color(edge), c.Radius("checkbox"))
		return
	}
	accent := c.Color(core.Accent)
	c.FillStyled(dst, box, accent, accent, c.Radius("checkbox"))
	x, y := float32(box.X), float32(box.Y)
	mark := c.Color(core.AccentText)
	c.Line(dst, x+3.5, y+7.5, x+6, y+10, 1.8, mark)
	c.Line(dst, x+6, y+10, x+10.5, y+4.5, 1.8, mark)
}

// Checkbox is a square that shows an accent fill and a check when on.
// OnChange runs when it is clicked; the caller flips the value it passes in.
func Checkbox(on bool, opts ...Option) *El {
	e := newEl(checkB{}, with(nil, opts))
	e.on = on
	return e
}

// The named icons below are wrappers over Icon for the icons the toolkit
// ships (view/icons/*.svg).

// CrossIcon is a small x, for delete buttons; it turns red under the pointer.
func CrossIcon(opts ...Option) *El { return Icon("cross", opts...) }

// TriangleIcon is a disclosure arrow: down when open, right when closed.
func TriangleIcon(open bool, opts ...Option) *El {
	if open {
		return Icon("triangle-down", opts...)
	}
	return Icon("triangle-right", opts...)
}

// GripIcon is the six-dot handle that rows are dragged by.
func GripIcon(opts ...Option) *El { return Icon("grip", opts...) }

// DotsIcon is the three-dot "more" glyph.
func DotsIcon(opts ...Option) *El { return Icon("dots", opts...) }

// TrashIcon is a trash can, for delete buttons; it turns red under the pointer.
func TrashIcon(opts ...Option) *El { return Icon("trash", opts...) }

// CopyIcon is two overlapping sheets.
func CopyIcon(opts ...Option) *El { return Icon("copy", opts...) }

// FolderIcon is an outlined folder, for choosing a file.
func FolderIcon(opts ...Option) *El { return Icon("folder", opts...) }

// SearchIcon is a magnifying glass.
func SearchIcon(opts ...Option) *El { return Icon("search", opts...) }

// PlusIcon is a plus sign.
func PlusIcon(opts ...Option) *El { return Icon("plus", opts...) }

// PromptIcon is a terminal prompt, ">_".
func PromptIcon(opts ...Option) *El { return Icon("prompt", opts...) }

// ChevronUpIcon and ChevronDownIcon are small arrows for stepping through a list.
func ChevronUpIcon(opts ...Option) *El   { return Icon("chevron-up", opts...) }
func ChevronDownIcon(opts ...Option) *El { return Icon("chevron-down", opts...) }
