package view

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Behavior is what a component outside this package supplies to be an
// element: how big it wants to be, what it does when its rect is set,
// how it reacts to the frame's input and how it draws. Read the rect and
// the children through e (Rect, Children); embed Base for the methods a
// component does not need.
type Behavior interface {
	// Measure is the size the element wants when at most maxW wide.
	Measure(e *El, c *core.Ctx, maxW int) (w, h int)
	// Layout runs after the element's rect is set.
	Layout(e *El, c *core.Ctx)
	// Update handles the frame's input, children first.
	Update(e *El, c *core.Ctx)
	// Draw paints the element; its children are drawn after it.
	Draw(e *El, c *core.Ctx, dst *ebiten.Image)
}

// Base is a Behavior that measures nothing, lays nothing out, ignores
// input and draws nothing. Embed it and override what the component does.
type Base struct{}

func (Base) Measure(*El, *core.Ctx, int) (int, int) { return 0, 0 }
func (Base) Layout(*El, *core.Ctx)                  {}
func (Base) Update(*El, *core.Ctx)                  {}
func (Base) Draw(*El, *core.Ctx, *ebiten.Image)     {}

// customB runs a Behavior supplied from outside the package as the
// element's own.
type customB struct{ Behavior }

func (b customB) measure(e *El, c *core.Ctx, maxW int) (int, int) { return b.Measure(e, c, maxW) }
func (b customB) layout(e *El, c *core.Ctx)                       { b.Layout(e, c) }
func (b customB) update(e *El, c *core.Ctx)                       { b.Update(e, c) }
func (b customB) draw(e *El, c *core.Ctx, dst *ebiten.Image)      { b.Draw(e, c, dst) }

// preUpdate and drawOver pass the runtime's optional hooks on to the
// component when it implements them.
func (b customB) preUpdate(e *El, c *core.Ctx) {
	if hook, ok := b.Behavior.(PreUpdater); ok {
		hook.PreUpdate(e, c)
	}
}

func (b customB) drawOver(e *El, c *core.Ctx, dst *ebiten.Image) {
	if hook, ok := b.Behavior.(OverDrawer); ok {
		hook.DrawOver(e, c, dst)
	}
}

// Custom makes an element that behaves as b. The component's options
// configure b through Behavior.
func Custom(b Behavior, items ...Item) *El { return newEl(customB{b}, items) }

// Behavior is the behavior a Custom element was made with, or nil for
// the toolkit's own kinds of element.
func (e *El) Behavior() Behavior {
	if custom, ok := e.b.(customB); ok {
		return custom.Behavior
	}
	return nil
}

// Linear is the Behavior of a Row (Horizontal) or a Col: it lays its
// children out along one axis with the container options (Gap, Pad,
// Grow, AlignItems, JustifyContent...). A component that is a container
// embeds it and overrides what it adds, as Resizable does with Update.
type Linear struct {
	Base
	Horizontal bool
}

func (l Linear) Measure(e *El, c *core.Ctx, maxW int) (int, int) {
	return flow{horiz: l.Horizontal}.measure(e, c, maxW)
}

func (l Linear) Layout(e *El, c *core.Ctx) { flow{horiz: l.Horizontal}.layout(e, c) }

// Place gives the element the rect r and lays its own children out in
// it. A container's Layout calls it on each child.
func (e *El) Place(c *core.Ctx, r core.Rect) { e.place(c, r) }

// Describer is implemented by a Behavior that has attributes worth
// showing in Dump: Describe returns them as "name=value" pairs.
type Describer interface{ Describe() string }

// FixedSize is the size given with W and H, 0 where the element takes its
// content's size.
func (e *El) FixedSize() (w, h int) { return e.w, e.h }

// ColorToken is the token given with Color, or fallback when there is
// none: the color a component draws in.
func (e *El) ColorToken(fallback core.Token) core.Token { return e.color(nil, fallback) }

// HoverColorToken is the token given with HoverColor, "" for none.
func (e *El) HoverColorToken() core.Token { return e.hoverColor }

// PreUpdater is implemented by a Behavior that must see the input before
// its children do, to take a click away from them.
type PreUpdater interface {
	PreUpdate(e *El, c *core.Ctx)
}

// OverDrawer is implemented by a Behavior that paints over its children.
type OverDrawer interface {
	DrawOver(e *El, c *core.Ctx, dst *ebiten.Image)
}

// Scroller is implemented by a Behavior that scrolls its children. When
// Tab moves the focus, the runtime asks every Scroller above the focused
// element to bring the element's rect into view.
type Scroller interface {
	ScrollTo(e *El, target core.Rect)
}

// InFlow reports whether the element takes part in its parent's layout:
// it is neither dropped for lack of room (Optional) nor floating.
func (e *El) InFlow() bool { return !e.hidden && !e.fl.on }

// KeepsInView reports whether the element was given KeepInView.
func (e *El) KeepsInView() bool { return e.reveal }

// Inner is the element's rect inside its padding.
func (e *El) Inner() core.Rect { return e.inner() }

// LayoutIn lays the children out as if the element's rect were r, and
// leaves the element's own rect alone: a scrolling container lays its
// children out in a rect shifted by the scroll position.
func (l Linear) LayoutIn(e *El, c *core.Ctx, r core.Rect) {
	own := e.rect
	e.rect = r
	flow{horiz: l.Horizontal}.layout(e, c)
	e.rect = own
}

// KeepInView marks the child a scrolling strip must keep in view: whenever
// a different child is flagged than last time, an HScroll scrolls to show
// it once, so a selection made elsewhere is never hidden, while the
// user's own scrolling is left alone. It is an attribute of the child, set
// here and read by its parent through KeepsInView.
func KeepInView() Option { return func(e *El) { e.reveal = true } }

// Focuser is the state of a component that takes the keyboard: it holds
// the focus flag the window and the runtime read and write.
type Focuser interface {
	HasFocus() bool
	SetFocus(on bool)
}

// Focusable is implemented by a Behavior that takes keyboard focus. The
// runtime moves focus between the Focusables in tree order on Tab, keeps
// one owner, and blurs one that has left the tree, all through the Focuser.
type Focusable interface {
	// FocusTarget is the state that holds the focus; it is also what
	// identifies the owner, so it must be a pointer.
	FocusTarget() Focuser
	// KeepsTab reports whether the component uses Tab itself (an editor
	// that indents), so the runtime leaves the focus where it is.
	KeepsTab() bool
}

// ControlName is the theme control style given with Control, "" for
// none: the name a component asks the theme for its corner radius with.
func (e *El) ControlName() string { return e.control }

// OnChange runs after the element's value was edited: the text of a field
// or an area, or a checkbox that was clicked. It is an attribute of any
// element, read by the component through ChangeHandler.
func OnChange(f func()) Option { return func(e *El) { e.onChange = f } }
