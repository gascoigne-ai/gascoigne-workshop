// Package view builds widget UIs as a tree of elements, declaratively, in
// plain Go. A view is a function that returns a tree; it is called every
// frame, so the tree is always a pure function of the application state:
// no diffing, no reactive system. Runnable examples: package uitoolkit/examples.
//
//	func view(a *App) view.Node {
//		return view.Col(view.Grow(1), view.Pad(12), view.Gap(8), // options first, by convention
//			view.Text("Hello", view.Muted()),
//			view.Field(&a.name, view.Placeholder("Your name"), view.OnEnter(a.save)),
//			view.If(a.name.Text != "", view.Button("Save", view.W(80), view.OnClick(a.save))),
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
//	Size       W H MinW MaxW MinH MaxH Grow Optional Clip
//	Spacing    Pad PadX PadY PadL PadT PadR PadB   Margin MarginX MarginY MarginL MarginT MarginR MarginB   Gap
//	Layout     Row Col Stack Split Scroll HScroll Spacer   AlignItems AlignSelf JustifyContent
//	Look       Bg Border BorderX BorderY BorderL BorderT BorderR BorderB HoverBg BorderInset Shadow Control EdgeT EdgeL EdgeR HoverColor Lighten
//	Text       Text Caption Lines   Color Muted Bold Small Mono FontSource Ellipsis TextCenter
//	Input      Field Area Code Slider Checkbox   (state: TextField TextArea CodeView)   Placeholder FieldPad Segments OnChange OnEnter OnTab
//	Events     OnClick OnClickAt OnRightClick OnRightClickIn OnMiddleClick OnDragOver   Tooltip KeepInView
//	Composite  Table   OnSelect SelectHandler DismissHandler ChangeHandler OnSlide OnSlideEnd SliderMid   Separator   (Button, Tabs, Menu, Dialog and the rest of the ready-made widgets are in package widget)
//	Split      Vertical AsRatio MinFirst MinSecond
//	Icons      Icon RegisterIcons IconNames   CrossIcon TriangleIcon GripIcon DotsIcon TrashIcon PlusIcon CopyIcon FolderIcon SearchIcon PromptIcon ChevronUpIcon ChevronDownIcon
//	Flow       If When For Frag
//	Overlay    FloatTR FloatAt FloatCenter Modal OnDismiss FocusScope   Resizable (state: ResizeState)
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
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Node is an element of the view tree. Constructors return *El so
// attributes chain.
type Node = *El

// behavior is what makes one kind of element different from another:
// how big it wants to be, how it places its children, what it paints
// and how it reacts to input. Kinds are values of this interface, not
// switch cases.
type behavior interface {
	measure(e *El, c *core.Ctx, maxW int) (w, h int)
	layout(e *El, c *core.Ctx)
	draw(e *El, c *core.Ctx, dst *ebiten.Image)
	update(e *El, c *core.Ctx)
}

// El is one element: a behavior plus the attributes shared by all.
type El struct {
	b     behavior
	kids  []*El
	rect  core.Rect // set by layout
	hover bool      // the pointer was over this element at the last draw, and hover was not blocked

	w, h             int // fixed size; 0 = content size
	grow             float64
	optional, hidden bool // optional elements are dropped when the row is too narrow
	padL, padT       int
	padR, padB       int
	gap              int

	bg, border, borderT, borderB, borderL, borderR core.Token
	borderInset                                    int // BorderL and BorderR stop this far from the top and bottom
	hoverBg                                        core.Token
	fontSource                                     *text.GoTextFaceSource // draws a text in this font instead of the interface one
	hoverColor                                     core.Token             // an icon's color under the pointer
	shadow, clip, lighten                          bool

	text                             string
	fg                               core.Token
	small, mono, bold, ellipsis, ctr bool
	placeholder                      string
	segments                         func(string) []Span
	field                            *TextField
	view                             *CodeView
	area                             *TextArea
	iconName                         string
	on                               bool

	onClick, onChange, onEnter func()
	onClickAt                  func(x, y int)
	onRightIn                  func(c *core.Ctx, x, y int) // a right click that needs the context, as TextMenu does
	onTab                      func(back bool)
	onMiddle, onDragOver       func()
	onSelect                   func(i int)

	edgeTok                      core.Token // Edges: the color of the outline
	edgeTop, edgeLeft, edgeRight bool
	fieldInset                   int    // Field padding when not the default
	control                      string // the theme control style that shapes and shades this element
	tip                          string // tooltip text, shown after the pointer rests on the element
	reveal                       bool   // an HScroll keeps this child in view when it becomes the flagged one

	alignItems, alignSelf  Align
	justify                Justify
	minW, maxW, minH, maxH int
	mL, mT, mR, mB         int       // margins
	split                  splitOpts // set by Split

	fl         floatOpts // set by Float, FloatAt and FloatCenter
	focusScope bool
	onRight    func(x, y int)
	modal      bool
	onDismiss  func()
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

// PadX sets the left and right padding.
func PadX(n int) Option { return func(e *El) { e.padL, e.padR = n, n } }

// PadY sets the top and bottom padding.
func PadY(n int) Option { return func(e *El) { e.padT, e.padB = n, n } }

// PadL sets the left padding.
func PadL(n int) Option { return func(e *El) { e.padL = n } }

// PadT sets the top padding.
func PadT(n int) Option { return func(e *El) { e.padT = n } }

// PadR sets the right padding.
func PadR(n int) Option { return func(e *El) { e.padR = n } }

// PadB sets the bottom padding.
func PadB(n int) Option { return func(e *El) { e.padB = n } }

// Gap sets the space between children.
func Gap(n int) Option { return func(e *El) { e.gap = n } }

// Appearance options.

// Bg fills the element with a theme color.
func Bg(t core.Token) Option { return func(e *El) { e.bg = t } }

// Border outlines the element.
func Border(t core.Token) Option { return func(e *El) { e.border = t } }

// BorderT draws a hairline along the top edge.
func BorderT(t core.Token) Option { return func(e *El) { e.borderT = t } }

// BorderB draws a hairline along the bottom edge, under the
// background and below the children.
// BorderL draws a one-pixel line down the element's left edge, to
// separate neighbours such as tabs or table columns; BorderInset shortens it.
func BorderL(t core.Token) Option { return func(e *El) { e.borderL = t } }

// Separator draws the soft hairline between neighbours (tabs, table
// columns) on the element's left edge; inset keeps it off the top and
// bottom. The first of a row has none: When(i > 0, Separator(10)). It has
// no template attribute because the template language has no conditional
// options; use border-l where the line is always there.
func Separator(inset int) Option {
	return func(e *El) { e.borderL, e.borderInset = core.BorderSoft, inset }
}

// When applies opt only if cond holds: the option form of If, so a
// condition is never a bool parameter of the option itself.
func When(cond bool, opt Option) Option {
	if cond {
		return opt
	}
	return func(*El) {}
}

// BorderInset keeps the BorderL and BorderR lines this many pixels from the top and bottom.
func BorderInset(n int) Option { return func(e *El) { e.borderInset = n } }

// BorderB draws a hairline along the bottom edge.
func BorderB(t core.Token) Option { return func(e *El) { e.borderB = t } }

// BorderR draws a hairline along the right edge.
func BorderR(t core.Token) Option { return func(e *El) { e.borderR = t } }

// BorderX draws hairlines along the left and right edges.
func BorderX(t core.Token) Option { return func(e *El) { e.borderL, e.borderR = t, t } }

// BorderY draws hairlines along the top and bottom edges.
func BorderY(t core.Token) Option { return func(e *El) { e.borderT, e.borderB = t, t } }

// Control names the theme style that draws this element: its corner
// radius, gradient fill and outline come from Theme.Controls[name], so
// every control of a kind looks the same and the theme file decides how.
func Control(name string) Option { return func(e *El) { e.control = name } }

// EdgeT, EdgeL and EdgeR outline the element on that side with a
// two-pixel line of the token's color; all three make the look of the open
// tab, which runs into the pane below it. The color of the last one wins.
func EdgeT(t core.Token) Option { return func(e *El) { e.edgeTok, e.edgeTop = t, true } }

// EdgeL outlines the left side (see EdgeT).
func EdgeL(t core.Token) Option { return func(e *El) { e.edgeTok, e.edgeLeft = t, true } }

// EdgeR outlines the right side (see EdgeT).
func EdgeR(t core.Token) Option { return func(e *El) { e.edgeTok, e.edgeRight = t, true } }

// Tooltip shows text near the pointer once it has rested on the element
// for tipDelayTicks.
func Tooltip(text string) Option { return func(e *El) { e.tip = text } }

// HoverBg fills the element while the pointer is over it.
func HoverBg(t core.Token) Option { return func(e *El) { e.hoverBg = t } }

// FontSource draws a Text in the font of source, at the size the text's other
// options choose (Small, Bold), instead of the interface font. Use it to show
// a font in its own face, as a font list does. It has no template attribute: a font source is a Go value.
func FontSource(source *text.GoTextFaceSource) Option { return func(e *El) { e.fontSource = source } }

// HoverColor is the color an icon takes while the pointer is over it, in
// place of the hover token of the icon itself. Use it where that token
// does not contrast with the surface behind the icon, such as on the
// banner.
func HoverColor(t core.Token) Option { return func(e *El) { e.hoverColor = t } }

// Lighten brightens the background while the pointer is over it.
func Lighten() Option { return func(e *El) { e.lighten = true } }

// Shadow draws a soft offset shadow behind the element.
func Shadow() Option { return func(e *El) { e.shadow = true } }

// Clip cuts the children off at the element's edge.
func Clip() Option { return func(e *El) { e.clip = true } }

// Events and overlays.

// OnClick runs f when the element is clicked.
func OnClick(f func()) Option { return func(e *El) { e.onClick = f } }

// OnClickAt is OnClick with the pointer position, for what opens at the
// click (a menu under the cursor); the app does not read the pointer itself.
// It has no template attribute: a template names a func(), not a func(x, y).
func OnClickAt(f func(x, y int)) Option { return func(e *El) { e.onClickAt = f } }

// OnMiddleClick runs f when the element is middle-clicked.
func OnMiddleClick(f func()) Option { return func(e *El) { e.onMiddle = f } }

// OnDragOver runs f every frame the left or middle button is held with the pointer
// over the element, for drag and drop: the element under the dragged
// item learns the item is over it.
func OnDragOver(f func()) Option { return func(e *El) { e.onDragOver = f } }

// FloatTR takes the element out of the layout of its Stack and pins it
// top and right, this many pixels from those edges.
func FloatTR(top, right int) Option {
	return func(e *El) { e.fl.on, e.fl.top, e.fl.right = true, top, right }
}

// Modal makes a floating element swallow every click; a click outside it
// runs the OnDismiss handler.
func Modal() Option { return func(e *El) { e.modal = true } }

// OnDismiss runs when a click lands outside a Modal element.
func OnDismiss(f func()) Option { return func(e *El) { e.onDismiss = f } }

// If is Vue's v-if: the node when cond holds, else nothing.
func If(cond bool, n Node) Node {
	if cond {
		return n
	}
	return nil
}

// splitOpts are the settings of a Split element.
type splitOpts struct {
	vertical, ratio     bool
	minFirst, minSecond int
}

// floatOpts are the settings of a floating element.
type floatOpts struct {
	on         bool
	top, right int
	mode       int // floatCorner, floatPoint or floatMiddle
}

// SelectHandler is the function given with OnSelect, nil for none. Widgets
// built on view read it when the user picks something, because the option
// is applied after they build their children.
func (e *El) SelectHandler() func(index int) { return e.onSelect }

// DismissHandler is the function given with OnDismiss, nil for none.
func (e *El) DismissHandler() func() { return e.onDismiss }

// ChangeHandler is the function given with OnChange, nil for none.
func (e *El) ChangeHandler() func() { return e.onChange }

// OnRightClickIn runs f when the element is right-clicked, with the context
// and the pointer position. It is OnRightClick for a handler that needs
// the context, such as one that opens a popup menu.
func OnRightClickIn(f func(c *core.Ctx, x, y int)) Option {
	return func(e *El) { e.onRightIn = f }
}

// OnSelect runs with the index of the item the user picked: the tab of a
// Tabs strip, the row of a Menu.
func OnSelect(f func(i int)) Option { return func(e *El) { e.onSelect = f } }

// Rect is where the last Layout put the element, in window pixels.
func (e *El) Rect() core.Rect { return e.rect }

// Children are the element's child elements in tree order. It is for
// code that looks at a laid-out tree, such as tests of widgets.
func (e *El) Children() []*El { return e.kids }
