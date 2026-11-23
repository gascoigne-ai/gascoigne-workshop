// Package view builds widget UIs as a tree of elements, declaratively, in
// plain Go. A view is a function that returns a tree; it is called every
// frame, so the tree is always a pure function of the application state:
// no diffing, no reactive system. Runnable examples: package uitoolkit/examples.
//
//	func view(a *App) view.Node {
//		return view.Col(view.Grow(1), view.Pad(12), view.Gap(8), // options first, by convention
//			view.Text("Hello", view.Muted()),
//			widget.Field(&a.name, widget.Placeholder("Your name"), widget.OnEnter(a.save)),
//			view.If(a.name.Text != "", widget.Button("Save", view.W(80), view.OnClick(a.save))),
//		)
//	}
//
// Constructors take options and children together: Row, Col and Stack
// take both; Text, Button, Field and the other leaves take only options.
// Options are the attributes of an element; they may stand anywhere in
// the list and a later one overrides an earlier one, which is how Button
// lets its caller override its defaults. Most views dot-import the
// package (import . ".../view") so they read like markup; import it as
// `v` if you prefer completion on a prefix.
//
// # Finding your way
//
// The names follow a few rules, so a guess is usually right:
//
//	Size       W H WOf HOf FitW MinW MaxW MinH MaxH Grow Shrink Optional Clip
//	Spacing    Pad PadX PadY PadL PadT PadR PadB   Margin MarginX MarginY MarginL MarginT MarginR MarginB   Gap
//	Layout     Row Col Stack Spacer HRule VRule VRuleInset   AlignItems AlignSelf JustifyContent
//	Look       Bg Border BorderX BorderY BorderL BorderT BorderR BorderB HoverBg BorderInset Shadow Control EdgeT EdgeL EdgeR EdgeB HoverColor Lighten
//	Text       Text Caption Lines   Color Muted Bold Small Mono FontSource Ellipsis TextCenter
//	Input      OnChange   (the text inputs, Field and its options, are in package widget)
//	Events     OnClick OnClickAt OnRightClick OnRightClickIn OnMiddleClick OnDoubleClick OnDragOver   Tooltip KeepInView
//	Events+    OnSelect SelectHandler DismissHandler ChangeHandler El.Select El.Dismiss El.Changed   Separator SeparatorOf   (Button, Tabs, Menu, Dialog and the rest of the ready-made widgets are in package widget)
//	Flow       If When For Frag With
//	Extend     Behavior Base Linear Custom Describer PreUpdater OverDrawer Scroller Focusable Focuser Linear.LayoutIn (El.Inner El.InFlow El.KeepsInView El.Behavior El.Place El.FixedSize El.ColorToken El.HoverColorToken)   a component outside this package is an element through these
//	Layers     ModalLayer DrawBelow DrawFrom   (draw the window under the first modal once, the rest every frame)
//	Overlay    FloatTR FloatAt FloatCenter Moved MoveState DragHandle DismissMargin RoundT RoundB RoundBL Modal KeepOnOutsideClick OnDismiss FocusScope
//
// A side is a letter suffix (PadL, PadT, PadR, PadB) and an axis is X or
// Y (PadX is left and right). The same options are the attributes of a
// template in package markup, in kebab-case (pad-x, border-t).
//
// Layout is a flexbox subset: a fixed size (W, H), a proportional share of
// the free space (Grow), or the content's own size. Dump prints a laid-out
// tree as text, which is how tests compare two views.
package view

import (
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Node is an element of the view tree. Constructors return *El so
// attributes chain.
type Node = *El

// El is one element: a Behavior plus the attributes shared by all.
type El struct {
	b      Behavior
	custom bool // b was supplied by Custom, not one of the toolkit's own kinds
	kids   []*El
	rect   core.Rect   // set by layout
	memo   measureMemo // what the behavior measured in the current Layout pass
	hover  bool        // the pointer was over this element at the last draw, and hover was not blocked

	w, h              int                        // fixed size; 0 = content size
	hOf, wOf, insetOf func(sizes core.Sizes) int // set by HOf, WOf, SeparatorOf; Layout resolves them into h, w, borderInset
	grow              float64
	optional, hidden  bool // optional elements are dropped when the row is too narrow
	shrink            bool // gives up width to an overflowing row, down to its MinW
	padL, padT        int
	padR, padB        int
	gap               int

	bg, border, borderT, borderB, borderL, borderR core.Token
	borderInset                                    int // BorderL and BorderR stop this far from the top and bottom
	hoverBg                                        core.Token
	fontSource                                     *text.GoTextFaceSource // draws a text in this font instead of the interface one
	hoverColor                                     core.Token             // an icon's color under the pointer
	shadow, clip, lighten                          bool

	text                             string
	fg                               core.Token
	small, mono, bold, ellipsis, ctr bool

	onChange             func()
	onClick              func(posX, posY int)                // OnClick and OnClickAt set it; the later wins
	onRight              func(ctx *core.Ctx, posX, posY int) // OnRightClick and OnRightClickIn set it; the later wins
	onMiddle, onDragOver func()
	onDouble             func()
	onSelect             func(i int)

	edgeTok                                  core.Token // Edges: the color of the outline
	edgeTop, edgeLeft, edgeRight, edgeBottom bool
	control                                  string // the theme control style that shapes and shades this element
	tip                                      string // tooltip text, shown after the pointer rests on the element
	reveal                                   bool   // an HScroll keeps this child in view when it becomes the flagged one

	alignItems, alignSelf  Align
	justify                Justify
	minW, maxW, minH, maxH int
	mL, mT, mR, mB         int // margins

	fl            floatOpts // set by Float, FloatAt and FloatCenter
	dragHandle    bool      // set by DragHandle
	dismissMargin int       // set by DismissMargin
	roundCtl      string    // set by RoundT, RoundB and RoundBL: the control whose radius the corners take
	roundCorners  core.Corners
	focusScope    bool
	modal         bool
	onDismiss     func()
	keepOnOutside bool // a modal that swallows an outside click without running onDismiss
}

// Item is anything that can go between the parentheses of a
// constructor: a child Node, or an Option that sets an attribute on the
// element being built. Attributes come first, children after, so what an
// element is reads before what it contains.
type Item interface{ apply(parent *El) }

// apply makes the element a child of parent; a nil node (a false If) adds nothing.
func (e *El) apply(parent *El) {
	if e != nil {
		parent.kids = append(parent.kids, e)
	}
}

// Option sets one attribute of the element it is passed to.
type Option func(*El)

func (o Option) apply(element *El) { o(element) }

func newEl(kind Behavior, items []Item) *El {
	e := &El{b: kind}
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

// floatOpts are the settings of a floating element.
type floatOpts struct {
	on         bool
	mode       floatMode
	top, right int // floatCorner: the distance from the top and right edges
	x, y       int // floatPoint: the top-left point
	moved      *MoveState
}

// dumpTop and dumpRight are the two offsets Dump has always shown: the
// corner offsets, or for a point the y and the x.
func (f floatOpts) dumpTop() int {
	if f.mode == floatPoint {
		return f.y
	}
	return f.top
}

func (f floatOpts) dumpRight() int {
	if f.mode == floatPoint {
		return f.x
	}
	return f.right
}

// Rect is where the last Layout put the element, in window pixels.
func (e *El) Rect() core.Rect { return e.rect }

// Children are the element's child elements in tree order. It is for
// code that looks at a laid-out tree, such as tests of widgets.
func (e *El) Children() []*El { return e.kids }

// memoSlots is how many widths one element remembers: a parent measures a
// child at its own width and lays it out at the inner one, so two or three
// differ at most.
const memoSlots = 3

// measureMemo holds the sizes a behavior measured for some maxW during the
// Layout pass numbered gen. A later pass starts empty.
type measureMemo struct {
	gen   int
	count int
	slots [memoSlots]struct{ maxW, w, h int }
}

// measure is the behavior's size at maxW, asked once per Layout pass.
// Outside a pass nothing is remembered.
func (e *El) measure(ctx *core.Ctx, maxW int) (w, h int) {
	if !ctx.Laying {
		return e.b.Measure(e, ctx, maxW)
	}
	if e.memo.gen != ctx.LayoutGen {
		e.memo = measureMemo{gen: ctx.LayoutGen}
	}
	if _, isText := e.b.(textB); isText {
		maxW = 0 // a line of text is as wide as it is whatever room it has: one entry serves every width
	}
	for i := 0; i < e.memo.count; i++ {
		if slot := e.memo.slots[i]; slot.maxW == maxW {
			return slot.w, slot.h
		}
	}
	w, h = e.b.Measure(e, ctx, maxW)
	if e.memo.count < memoSlots {
		e.memo.slots[e.memo.count] = struct{ maxW, w, h int }{maxW, w, h}
		e.memo.count++
	}
	return w, h
}
