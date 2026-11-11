package view

import (
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Layout places the tree in a w by h window.
func Layout(c *core.Ctx, root Node, w, h int) {
	root.place(c, core.Rect{W: w, H: h})
}

// Update delivers this frame's input (c.In) to the tree, top-most
// element first. It must follow Layout on the same tree.
func Update(c *core.Ctx, root Node) {
	c.PointerX, c.PointerY = c.In.X, c.In.Y
	settleFocus(c, root)
	root.update(c)
	cycleFocus(c, root)
	if c.In.Tab {
		revealFocused(root)
	}
}

// revealFocused scrolls every Scroll that holds the focused element so the
// element is inside its box, which is what Tab moving focus into a field
// that is out of view needs. It uses this frame's layout; the container
// shows the new position on the next frame.
func revealFocused(root Node) {
	var path []*El
	var find func(e *El) bool
	find = func(e *El) bool {
		if e.hidden {
			return false
		}
		path = append(path, e)
		if e.focusable() && e.focused() {
			return true
		}
		for _, k := range e.kids {
			if find(k) {
				return true
			}
		}
		path = path[:len(path)-1]
		return false
	}
	if !find(root) {
		return
	}
	target := path[len(path)-1].rect
	for _, e := range path {
		if scroller, ok := e.Behavior().(Scroller); ok {
			scroller.ScrollTo(e, target)
		}
	}
}

func (f *TextField) HasFocus() bool { return f.Focused }
func (f *TextField) SetFocus(on bool) {
	f.Focused = on
	if on {
		f.SelectAll()
	}
}

func (a *TextArea) HasFocus() bool   { return a.Focused }
func (a *TextArea) SetFocus(on bool) { a.Focused = on }
func (v *CodeView) HasFocus() bool   { return v.Focused }
func (v *CodeView) SetFocus(on bool) { v.Focused = on }

// state is what e edits and focuses, nil for an element that is not
// focusable. It is also what identifies the focus owner.
func (e *El) state() Focuser {
	if focusable, ok := e.Behavior().(Focusable); ok {
		return focusable.FocusTarget()
	}
	switch {
	case e.field != nil:
		return e.field
	case e.area != nil:
		return e.area
	case e.view != nil:
		return e.view
	}
	return nil
}

func (e *El) focusable() bool { return e.state() != nil }

func (e *El) focused() bool { return e.focusable() && e.state().HasFocus() }

func (e *El) setFocused(on bool) {
	if state := e.state(); state != nil {
		state.SetFocus(on)
	}
}

// keepsTab reports whether the focused element uses Tab itself, so Tab
// does not move the focus away from it.
func (e *El) keepsTab() bool {
	if focusable, ok := e.Behavior().(Focusable); ok && focusable.KeepsTab() {
		return true
	}
	return e.onTab != nil || e.area != nil
}

// focusKey identifies the focus owner: the state's pointer.
func (e *El) focusKey() any {
	if state := e.state(); state != nil {
		return state
	}
	return nil
}

// blurKey takes the keyboard from a state that may no longer be in the tree.
func blurKey(key any) {
	if state, ok := key.(Focuser); ok {
		state.SetFocus(false)
	}
}

// settleFocus keeps the one-owner rule before the frame's input is
// delivered. The app asks for focus by setting a state's Focused, so the
// owner is the focused state that was not the owner before (the last one in
// tree order if several asked); every other state is blurred, and so is an
// owner that left the tree.
func settleFocus(c *core.Ctx, root Node) {
	list := collectFocusables(root, nil)
	var asked any
	for _, e := range list {
		if key := e.focusKey(); e.focused() && key != c.FocusOwner {
			asked = key
		}
	}
	switch {
	case asked != nil:
		blurKey(c.FocusOwner)
		c.FocusOwner = asked
	case c.FocusOwner != nil && !slices.ContainsFunc(list, func(e *El) bool { return e.focusKey() == c.FocusOwner && e.focused() }):
		blurKey(c.FocusOwner)
		c.FocusOwner = nil
	}
	for _, e := range list {
		if key := e.focusKey(); key != c.FocusOwner && e.focused() {
			e.setFocused(false)
		}
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
func cycleFocus(c *core.Ctx, root Node) {
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
			if e.keepsTab() { // these keep Tab for themselves
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
func Draw(c *core.Ctx, root Node, dst *ebiten.Image) {
	c.HoverOff = hasModal(root) // only the overlay's own elements react to the pointer
	c.TipText = ""
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

// preUpdater is implemented by behaviors that must see the input before
// their children do, to take a click away from them.
type preUpdater interface {
	preUpdate(e *El, c *core.Ctx)
}

// update reports whether a modal element swallowed the input.
func (e *El) update(c *core.Ctx) bool {
	if e.hidden {
		return false
	}
	if p, ok := e.b.(preUpdater); ok {
		p.preUpdate(e, c)
	}
	saved := c.In
	masked := e.clip && !c.In.Over(e.rect)
	if masked {
		// Children of a clipping element may extend past its edge (rows
		// scrolled out of view): they must not be hit by a pointer that
		// is outside it. The click itself still reaches them, so a text
		// field there can see it was clicked away from and blur.
		c.In.X, c.In.Y, c.In.Wheel, c.In.WheelX = -1<<20, -1<<20, 0, 0
	}
	swallowed := false
	for i := len(e.kids) - 1; i >= 0; i-- {
		if e.kids[i].update(c) {
			swallowed = true
			break
		}
	}
	wheel, wheelX, rightClick := c.In.Wheel, c.In.WheelX, c.In.RightClick // a child may have used them
	c.In = saved
	if !masked {
		c.In.Wheel, c.In.WheelX, c.In.RightClick = wheel, wheelX, rightClick
	}
	if swallowed {
		return true
	}
	e.b.update(e, c)
	if e.onClick != nil && c.In.Hit(e.rect) {
		e.onClick()
	}
	if e.onClickAt != nil && c.In.Hit(e.rect) {
		e.onClickAt(c.In.X, c.In.Y)
	}
	if e.onMiddle != nil && c.In.MiddleClick && c.In.Over(e.rect) {
		e.onMiddle()
	}
	if e.onDragOver != nil && (c.In.Held || c.In.MiddleHeld) && c.In.Over(e.rect) {
		e.onDragOver()
	}
	if e.onRight != nil && c.In.RightClick && c.In.Over(e.rect) {
		e.onRight(c.In.X, c.In.Y)
		c.In.RightClick = false // one right click opens one menu: the innermost wins
	}
	if e.onRightIn != nil && c.In.RightClick && c.In.Over(e.rect) {
		e.onRightIn(c, c.In.X, c.In.Y)
		c.In.RightClick = false
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
	drawOver(e *El, c *core.Ctx, dst *ebiten.Image)
}

func (e *El) draw(c *core.Ctx, dst *ebiten.Image) {
	if e.hidden {
		return
	}
	r := e.rect
	if e.modal {
		defer func(was bool) { c.HoverOff = was }(c.HoverOff)
		c.HoverOff = false // hover works inside the overlay
	}
	style := c.Th.Style(e.control)
	if e.shadow {
		sh := c.Color(core.Shadow)
		c.FillStyled(dst, core.Rect{X: r.X + 4, Y: r.Y + 4, W: r.W, H: r.H}, sh, sh, style.Radius)
	}
	over := c.Hovered(r)
	e.hover = over
	if over && e.tip != "" {
		c.TipText = e.tip // children draw later, so the innermost tooltip wins
	}
	if e.bg != "" {
		clr := c.Color(e.bg)
		if e.lighten && over {
			clr = core.Lighten(clr)
		}
		top, bottom := core.GradientFor(clr, style)
		c.FillStyled(dst, r, top, bottom, style.Radius)
	}
	if e.hoverBg != "" && over {
		hv := c.Color(e.hoverBg)
		c.FillStyled(dst, r, hv, hv, style.Radius)
	}
	if e.borderT != "" {
		c.HLine(dst, r.X, r.Y, r.W, c.Color(e.borderT))
	}
	if e.borderL != "" {
		c.VLine(dst, r.X, r.Y+e.borderInset, r.H-2*e.borderInset, c.Color(e.borderL))
	}
	if e.borderR != "" {
		c.VLine(dst, r.X+r.W-1, r.Y+e.borderInset, r.H-2*e.borderInset, c.Color(e.borderR))
	}
	if e.borderB != "" {
		c.HLine(dst, r.X, r.Y+r.H-1, r.W, c.Color(e.borderB))
	}

	e.b.draw(e, c, dst)
	kidsDst := dst
	if e.clip {
		kidsDst = c.Clip(dst, r)
	}
	for _, k := range e.kids {
		k.draw(c, kidsDst)
	}
	border := e.border
	if border == "" {
		border = style.Border
	}
	if border != "" { // after the children, so striped rows do not paint over it
		c.OutlineStyled(dst, r, c.Color(border), style.Radius)
	}
	if o, ok := e.b.(overDrawer); ok {
		o.drawOver(e, c, dst)
	}

	if e.edgeTok != "" { // the open tab's outline: top, left and right, bottom left open
		col := c.Color(e.edgeTok)
		if e.edgeTop {
			c.Fill(dst, core.Rect{X: r.X, Y: r.Y, W: r.W, H: 2}, col)
		}
		if e.edgeLeft {
			c.Fill(dst, core.Rect{X: r.X, Y: r.Y, W: 2, H: r.H}, col)
		}
		if e.edgeRight {
			c.Fill(dst, core.Rect{X: r.X + r.W - 2, Y: r.Y, W: 2, H: r.H}, col)
		}
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
