// Package vml is a declarative, Vue-flavoured way to build widget UIs in
// plain Go. A view is a function that returns a tree of nodes built from
// small constructors (Row, Col, Text, Field, If, For, ...) and chained
// attributes:
//
//	vml.Row(
//		vml.Text("Hello").Muted(),
//		vml.Spacer(),
//		vml.Button("Send").OnClick(send),
//	).Gap(8).Pad(12).H(40).Bg(widget.Bg)
//
// State lives in the application. The view is called every frame, so a
// tree is always a pure function of the state: no diffing, no reactive
// system. Layout is a flexbox subset: fixed size (W, H), proportional
// share of the free space (Grow), or the content's own size.
package vml

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/widget"
)

// Node is an element of the view tree. Constructors return *El so
// attributes chain.
type Node = *El

// behavior is what makes one kind of element different from another:
// how big it wants to be, how it places its children, what it paints
// and how it reacts to input. Kinds are values of this interface, not
// switch cases.
type behavior interface {
	measure(e *El, c *widget.Ctx, maxW int) (w, h int)
	layout(e *El, c *widget.Ctx)
	draw(e *El, c *widget.Ctx, dst *ebiten.Image)
	update(e *El, c *widget.Ctx)
}

// El is one element: a behavior plus the attributes shared by all.
type El struct {
	b    behavior
	kids []*El
	rect widget.Rect // set by layout

	w, h             int // fixed size; 0 = content size
	grow             float64
	optional, hidden bool // optional elements are dropped when the row is too narrow
	padL, padT       int
	padR, padB       int
	gap              int

	bg, border, borderTop, borderBottom widget.Token
	hoverBg, underline                  widget.Token
	underInset                          int
	shadow, clip, lighten               bool

	text                       string
	fg                         widget.Token
	small, mono, ellipsis, ctr bool
	placeholder                string
	segments                   func(string) []Seg
	field                      *widget.TextField
	view                       *widget.CodeView
	icon                       iconKind
	on                         bool

	onClick, onChange, onEnter func()

	float                bool
	floatTop, floatRight int
	modal                bool
	onDismiss            func()
}

func newEl(b behavior, kids []*El) *El { return &El{b: b, kids: flatten(kids)} }

// flatten drops nil children (a false If) and splices fragments (For).
func flatten(kids []*El) []*El {
	var out []*El
	for _, k := range kids {
		switch {
		case k == nil:
		case isFragment(k):
			out = append(out, k.kids...)
		default:
			out = append(out, k)
		}
	}
	return out
}

// Size attributes.

// W fixes the width in pixels.
func (e *El) W(n int) *El { e.w = n; return e }

// H fixes the height in pixels.
func (e *El) H(n int) *El { e.h = n; return e }

// Grow makes the element take a share of the free space in its row or
// column, proportional to weight.
func (e *El) Grow(weight float64) *El { e.grow = weight; return e }

// Optional lets a row or column drop the element when there is not
// enough room for everything.
func (e *El) Optional() *El { e.optional = true; return e }

// Pad sets the same padding on all sides.
func (e *El) Pad(n int) *El { e.padL, e.padT, e.padR, e.padB = n, n, n, n; return e }

// PadXY sets horizontal and vertical padding.
func (e *El) PadXY(x, y int) *El { e.padL, e.padR, e.padT, e.padB = x, x, y, y; return e }

// PadL sets the left padding.
func (e *El) PadL(n int) *El { e.padL = n; return e }

// PadR sets the right padding.
func (e *El) PadR(n int) *El { e.padR = n; return e }

// Gap sets the space between children.
func (e *El) Gap(n int) *El { e.gap = n; return e }

// Appearance attributes.

// Bg fills the element with a theme color.
func (e *El) Bg(t widget.Token) *El { e.bg = t; return e }

// Border outlines the element.
func (e *El) Border(t widget.Token) *El { e.border = t; return e }

// BorderTop draws a hairline along the top edge.
func (e *El) BorderTop(t widget.Token) *El { e.borderTop = t; return e }

// BorderBottom draws a hairline along the bottom edge, under the
// background and below the children.
func (e *El) BorderBottom(t widget.Token) *El { e.borderBottom = t; return e }

// HoverBg fills the element while the pointer is over it.
func (e *El) HoverBg(t widget.Token) *El { e.hoverBg = t; return e }

// Lighten brightens the background while the pointer is over it.
func (e *El) Lighten() *El { e.lighten = true; return e }

// Underline draws a two-pixel bar along the bottom, inset on both sides.
func (e *El) Underline(t widget.Token, inset int) *El {
	e.underline, e.underInset = t, inset
	return e
}

// Shadow draws a soft offset shadow behind the element.
func (e *El) Shadow() *El { e.shadow = true; return e }

// Clip cuts the children off at the element's edge.
func (e *El) Clip() *El { e.clip = true; return e }

// Events.

// OnClick runs f when the element is clicked.
func (e *El) OnClick(f func()) *El { e.onClick = f; return e }

// Float takes the element out of the layout of its Stack and pins it
// top and right, this many pixels from those edges.
func (e *El) Float(top, right int) *El {
	e.float, e.floatTop, e.floatRight = true, top, right
	return e
}

// Modal makes a floating element swallow every click; a click outside
// it runs onDismiss.
func (e *El) Modal(onDismiss func()) *El {
	e.modal, e.onDismiss = true, onDismiss
	return e
}
