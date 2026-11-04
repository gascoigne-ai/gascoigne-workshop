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

	"github.com/gascoigne-ai/gascoigne-workshop/ui/widget"
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
	b     behavior
	kids  []*El
	rect  widget.Rect // set by layout
	hover bool        // the pointer was over this element at the last draw, and hover was not blocked

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
	area                       *widget.TextArea
	icon                       iconKind
	on                         bool

	onClick, onChange, onEnter func()
	onTab                      func(back bool)
	onMiddle, onHeldOver       func()

	scrollMaxH int    // MaxHeight of a Scroll, 0 for none
	tip        string // tooltip text, shown after the pointer rests on the element
	reveal     bool   // an HScroll keeps this child in view when it becomes the flagged one

	alignItems, alignSelf         Align
	justify                       Justify
	minW, maxW, minH, maxH        int
	mL, mT, mR, mB                int // margins
	splitVertical, splitRatio     bool
	splitMinFirst, splitMinSecond int

	float                bool
	floatTop, floatRight int
	floatMode            int // floatCorner, floatPoint or floatMiddle
	focusScope           bool
	onRight              func(x, y int)
	modal                bool
	onDismiss            func()
}

// Item is anything that can go between the parentheses of a
// constructor: a child Node, or an Option that sets an attribute on the
// element being built. Attributes come first, children after, so what an
// element is reads before what it contains.
type Item interface{ apply(parent *El) }

// apply makes e a child of parent; a nil node (a false If) adds nothing.
func (e *El) apply(parent *El) {
	if e != nil {
		parent.kids = append(parent.kids, e)
	}
}

// Option sets one attribute of the element it is passed to.
type Option func(*El)

func (o Option) apply(e *El) { o(e) }

func newEl(b behavior, items []Item) *El {
	e := &El{b: b}
	for _, it := range items {
		if it != nil { // a bare nil, as opposed to a nil node from If
			it.apply(e)
		}
	}
	e.kids = flatten(e.kids)
	return e
}

// flatten splices fragments (For) into their parent's children.
func flatten(kids []*El) []*El {
	var out []*El
	for _, k := range kids {
		if isFragment(k) {
			out = append(out, k.kids...)
			continue
		}
		out = append(out, k)
	}
	return out
}

// Size options.

// W fixes the width in pixels.
func W(n int) Option { return func(e *El) { e.w = n } }

// H fixes the height in pixels.
func H(n int) Option { return func(e *El) { e.h = n } }

// Grow makes the element take a share of the free space in its row or
// column, proportional to weight.
func Grow(weight float64) Option { return func(e *El) { e.grow = weight } }

// Optional lets a row or column drop the element when there is not
// enough room for everything.
func Optional() Option { return func(e *El) { e.optional = true } }

// Pad sets the same padding on all sides.
func Pad(n int) Option {
	return func(e *El) { e.padL, e.padT, e.padR, e.padB = n, n, n, n }
}

// PadXY sets horizontal and vertical padding.
func PadXY(x, y int) Option {
	return func(e *El) { e.padL, e.padR, e.padT, e.padB = x, x, y, y }
}

// PadL sets the left padding.
func PadL(n int) Option { return func(e *El) { e.padL = n } }

// PadR sets the right padding.
func PadR(n int) Option { return func(e *El) { e.padR = n } }

// Gap sets the space between children.
func Gap(n int) Option { return func(e *El) { e.gap = n } }

// Appearance options.

// Bg fills the element with a theme color.
func Bg(t widget.Token) Option { return func(e *El) { e.bg = t } }

// Border outlines the element.
func Border(t widget.Token) Option { return func(e *El) { e.border = t } }

// BorderTop draws a hairline along the top edge.
func BorderTop(t widget.Token) Option { return func(e *El) { e.borderTop = t } }

// BorderBottom draws a hairline along the bottom edge, under the
// background and below the children.
func BorderBottom(t widget.Token) Option { return func(e *El) { e.borderBottom = t } }

// HoverBg fills the element while the pointer is over it.
// Tooltip shows text near the pointer once it has rested on the element
// for TipDelayTicks.
func Tooltip(text string) Option { return func(e *El) { e.tip = text } }

func HoverBg(t widget.Token) Option { return func(e *El) { e.hoverBg = t } }

// Lighten brightens the background while the pointer is over it.
func Lighten() Option { return func(e *El) { e.lighten = true } }

// Underline draws a two-pixel bar along the bottom, inset on both sides.
func Underline(t widget.Token, inset int) Option {
	return func(e *El) { e.underline, e.underInset = t, inset }
}

// Shadow draws a soft offset shadow behind the element.
func Shadow() Option { return func(e *El) { e.shadow = true } }

// Clip cuts the children off at the element's edge.
func Clip() Option { return func(e *El) { e.clip = true } }

// Events and overlays.

// OnClick runs f when the element is clicked.
func OnClick(f func()) Option { return func(e *El) { e.onClick = f } }

// OnMiddleClick runs f when the element is middle-clicked.
func OnMiddleClick(f func()) Option { return func(e *El) { e.onMiddle = f } }

// OnHeldOver runs f every frame the left button is held with the pointer
// over the element, for drag and drop: the element under the dragged
// item learns the item is over it.
func OnHeldOver(f func()) Option { return func(e *El) { e.onHeldOver = f } }

// Float takes the element out of the layout of its Stack and pins it
// top and right, this many pixels from those edges.
func Float(top, right int) Option {
	return func(e *El) { e.float, e.floatTop, e.floatRight = true, top, right }
}

// Modal makes a floating element swallow every click; a click outside
// it runs onDismiss.
func Modal(onDismiss func()) Option {
	return func(e *El) { e.modal, e.onDismiss = true, onDismiss }
}

// If is Vue's v-if: the node when cond holds, else nothing.
func If(cond bool, n Node) Node {
	if cond {
		return n
	}
	return nil
}

// Either is v-if with v-else.
func Either(cond bool, yes, no Node) Node {
	if cond {
		return yes
	}
	return no
}
