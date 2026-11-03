package vml

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/widget"
)

// Layout places the tree in a w by h window.
func Layout(c *widget.Ctx, root Node, w, h int) {
	root.place(c, widget.Rect{W: w, H: h})
}

// Update delivers this frame's input (c.In) to the tree, top-most
// element first. It must follow Layout on the same tree.
func Update(c *widget.Ctx, root Node) {
	root.update(c)
	cycleFocus(c, root)
}

// focusable reports whether e takes keyboard focus: text fields and code views.
func (e *El) focusable() bool { return e.field != nil || e.view != nil || e.area != nil }

func (e *El) focused() bool {
	switch {
	case e.field != nil:
		return e.field.Focused
	case e.area != nil:
		return e.area.Focused
	}
	return e.view != nil && e.view.Focused
}

func (e *El) setFocused(on bool) {
	switch {
	case e.field != nil:
		e.field.Focused = on
		if on {
			e.field.SelectAll()
		}
	case e.area != nil:
		e.area.Focused = on
	case e.view != nil:
		e.view.Focused = on
	}
}

func collectFocusables(e *El, out []*El) []*El {
	if e.hidden {
		return out
	}
	if e.focusable() {
		out = append(out, e)
	}
	for _, k := range e.kids {
		out = collectFocusables(k, out)
	}
	return out
}

// cycleFocus moves focus to the next focusable element in tree order on
// Tab, or the previous one on Shift+Tab, wrapping at the ends. A focused
// field that handles Tab itself (a table cell editor) keeps it.
func cycleFocus(c *widget.Ctx, root Node) {
	if !c.In.Tab || c.In.Ctrl {
		return
	}
	if scope := findScope(root); scope != nil {
		root = scope
	}
	list := collectFocusables(root, nil)
	if len(list) == 0 {
		return
	}
	cur := -1
	for i, e := range list {
		if e.focused() {
			cur = i
			if e.onTab != nil || e.area != nil { // these keep Tab for themselves
				return
			}
		}
	}
	next := 0
	switch {
	case cur >= 0 && c.In.Shift:
		next = (cur - 1 + len(list)) % len(list)
	case cur >= 0:
		next = (cur + 1) % len(list)
	case c.In.Shift:
		next = len(list) - 1
	}
	for _, e := range list {
		e.setFocused(false)
	}
	list[next].setFocused(true)
}

// Draw paints the tree. It must follow Layout on the same tree.
func Draw(c *widget.Ctx, root Node, dst *ebiten.Image) {
	c.HoverOff = hasModal(root) // only the overlay's own elements react to the pointer
	root.draw(c, dst)
	c.HoverOff = false
}

// hasModal reports whether any visible element is a modal overlay.
func hasModal(e *El) bool {
	if e.hidden {
		return false
	}
	if e.modal {
		return true
	}
	for _, k := range e.kids {
		if hasModal(k) {
			return true
		}
	}
	return false
}

// update reports whether a modal element swallowed the input.
func (e *El) update(c *widget.Ctx) bool {
	if e.hidden {
		return false
	}
	saved := c.In
	masked := e.clip && !c.In.Over(e.rect)
	if masked {
		// Children of a clipping element may extend past its edge (rows
		// scrolled out of view): they must not be hit by a pointer that
		// is outside it. The click itself still reaches them, so a text
		// field there can see it was clicked away from and blur.
		c.In.X, c.In.Y, c.In.Wheel = -1<<20, -1<<20, 0
	}
	swallowed := false
	for i := len(e.kids) - 1; i >= 0; i-- {
		if e.kids[i].update(c) {
			swallowed = true
			break
		}
	}
	wheel := c.In.Wheel // a child scroller may have used it
	c.In = saved
	if !masked {
		c.In.Wheel = wheel
	}
	if swallowed {
		return true
	}
	e.b.update(e, c)
	if e.onClick != nil && c.In.Hit(e.rect) {
		e.onClick()
	}
	if e.onMiddle != nil && c.In.MiddleClick && c.In.Over(e.rect) {
		e.onMiddle()
	}
	if e.onHeldOver != nil && c.In.Held && c.In.Over(e.rect) {
		e.onHeldOver()
	}
	if e.onRight != nil && c.In.RightClick && c.In.Over(e.rect) {
		e.onRight(c.In.X, c.In.Y)
	}
	if e.modal {
		// A modal overlay owns the pointer: nothing under it gets clicks,
		// wheel or hover, and a click outside it dismisses it.
		if (c.In.Click || c.In.RightClick) && !e.rect.Contains(c.In.X, c.In.Y) && e.onDismiss != nil {
			e.onDismiss()
		}
		return true
	}
	return false
}

// overDrawer is implemented by behaviors that paint over their children,
// such as a scrollbar.
type overDrawer interface {
	drawOver(e *El, c *widget.Ctx, dst *ebiten.Image)
}

func (e *El) draw(c *widget.Ctx, dst *ebiten.Image) {
	if e.hidden {
		return
	}
	r := e.rect
	if e.modal {
		defer func(was bool) { c.HoverOff = was }(c.HoverOff)
		c.HoverOff = false // hover works inside the overlay
	}
	if e.shadow {
		c.Fill(dst, widget.Rect{X: r.X + 4, Y: r.Y + 4, W: r.W, H: r.H}, c.Col(widget.Shadow))
	}
	over := c.Hovered(r)
	e.hover = over
	if e.bg != "" {
		clr := c.Col(e.bg)
		if e.lighten && over {
			clr = widget.Lighten(clr)
		}
		c.Fill(dst, r, clr)
	}
	if e.hoverBg != "" && over {
		c.Fill(dst, r, c.Col(e.hoverBg))
	}
	if e.border != "" {
		c.Outline(dst, r, c.Col(e.border))
	}
	if e.borderTop != "" {
		c.HLine(dst, r.X, r.Y, r.W, c.Col(e.borderTop))
	}
	if e.borderBottom != "" {
		c.HLine(dst, r.X, r.Y+r.H-1, r.W, c.Col(e.borderBottom))
	}

	e.b.draw(e, c, dst)
	kidsDst := dst
	if e.clip {
		kidsDst = c.Clip(dst, r)
	}
	for _, k := range e.kids {
		k.draw(c, kidsDst)
	}
	if o, ok := e.b.(overDrawer); ok {
		o.drawOver(e, c, dst)
	}

	if e.underline != "" {
		c.Fill(dst, widget.Rect{X: r.X + e.underInset, Y: r.Y + r.H - 2, W: r.W - 2*e.underInset, H: 2}, c.Col(e.underline))
	}
}

// findScope is the last FocusScope element in tree order, the one on top.
func findScope(e *El) *El {
	if e.hidden {
		return nil
	}
	var found *El
	if e.focusScope {
		found = e
	}
	for _, k := range e.kids {
		if inner := findScope(k); inner != nil {
			found = inner
		}
	}
	return found
}
