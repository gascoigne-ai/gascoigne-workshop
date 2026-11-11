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
