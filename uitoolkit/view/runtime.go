package view

import (
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Layout places the tree in a width by height window.
func Layout(ctx *core.Ctx, root Node, width, height int) {
	root.place(ctx, core.Rect{W: width, H: height})
}

// Update delivers this frame's input (c.In) to the tree, top-most
// element first. It must follow Layout on the same tree.
func Update(ctx *core.Ctx, root Node) {
	ctx.PointerX, ctx.PointerY = ctx.In.X, ctx.In.Y
	settleFocus(ctx, root)
	root.update(ctx)
	cycleFocus(ctx, root)
	if ctx.In.Tab {
		revealFocused(root)
	}
}

// revealFocused scrolls every Scroll that holds the focused element so the
// element is inside its box, which is what Tab moving focus into a field
// that is out of view needs. It uses this frame's layout; the container
// shows the new position on the next frame.
func revealFocused(root Node) {
	var path []*El
	var find func(element *El) bool
	find = func(element *El) bool {
		if element.hidden {
			return false
		}
		path = append(path, element)
		if element.focusable() && element.focused() {
			return true
		}
		for _, k := range element.kids {
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

// state is what the element edits and focuses, nil for an element that is not
// focusable. It is also what identifies the focus owner.
func (e *El) state() Focuser {
	if focusable, ok := e.Behavior().(Focusable); ok {
		return focusable.FocusTarget()
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
	focusable, ok := e.Behavior().(Focusable)
	return ok && focusable.KeepsTab()
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
func settleFocus(ctx *core.Ctx, root Node) {
	list := collectFocusables(root, nil)
	var asked any
	for _, e := range list {
		if key := e.focusKey(); e.focused() && key != ctx.FocusOwner {
			asked = key
		}
	}
	switch {
	case asked != nil:
		blurKey(ctx.FocusOwner)
		ctx.FocusOwner = asked
	case ctx.FocusOwner != nil && !slices.ContainsFunc(list, func(element *El) bool { return element.focusKey() == ctx.FocusOwner && element.focused() }):
		blurKey(ctx.FocusOwner)
		ctx.FocusOwner = nil
	}
	for _, e := range list {
		if key := e.focusKey(); key != ctx.FocusOwner && e.focused() {
			e.setFocused(false)
		}
	}
}

func collectFocusables(element *El, out []*El) []*El {
	if element.hidden {
		return out
	}
	if element.focusable() {
		out = append(out, element)
	}
	for _, k := range element.kids {
		out = collectFocusables(k, out)
	}
	return out
}

// cycleFocus moves focus to the next focusable element in tree order on
// Tab, or the previous one on Shift+Tab, wrapping at the ends. A focused
// field that handles Tab itself (a table cell editor) keeps it.
func cycleFocus(ctx *core.Ctx, root Node) {
	if !ctx.In.Tab || ctx.In.Ctrl {
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
	case cur >= 0 && ctx.In.Shift:
		next = (cur - 1 + len(list)) % len(list)
	case cur >= 0:
		next = (cur + 1) % len(list)
	case ctx.In.Shift:
		next = len(list) - 1
	}
	for _, e := range list {
		e.setFocused(false)
	}
	list[next].setFocused(true)
}

// Draw paints the tree. It must follow Layout on the same tree.
func Draw(ctx *core.Ctx, root Node, dst *ebiten.Image) {
	ctx.HoverOff = hasModal(root) // only the overlay's own elements react to the pointer
	ctx.TipText = ""
	root.draw(ctx, dst)
	ctx.HoverOff = false
}

// hasModal reports whether any visible element is a modal overlay.
func hasModal(element *El) bool {
	if element.hidden {
		return false
	}
	if element.modal {
		return true
	}
	for _, k := range element.kids {
		if hasModal(k) {
			return true
		}
	}
	return false
}

// update reports whether a modal element swallowed the input.
func (e *El) update(ctx *core.Ctx) bool {
	if e.hidden {
		return false
	}
	if p, ok := e.b.(PreUpdater); ok {
		p.PreUpdate(e, ctx)
	}
	saved := ctx.In
	masked := e.clip && !ctx.In.Over(e.rect)
	if masked {
		// Children of a clipping element may extend past its edge (rows
		// scrolled out of view): they must not be hit by a pointer that
		// is outside it. The click itself still reaches them, so a text
		// field there can see it was clicked away from and blur.
		ctx.In.X, ctx.In.Y, ctx.In.Wheel, ctx.In.WheelX = -1<<20, -1<<20, 0, 0
	}
	swallowed := false
	for i := len(e.kids) - 1; i >= 0; i-- {
		if e.kids[i].update(ctx) {
			swallowed = true
			break
		}
	}
	wheel, wheelX, rightClick := ctx.In.Wheel, ctx.In.WheelX, ctx.In.RightClick // a child may have used them
	ctx.In = saved
	if !masked {
		ctx.In.Wheel, ctx.In.WheelX, ctx.In.RightClick = wheel, wheelX, rightClick
	}
	if swallowed {
		return true
	}
	e.b.Update(e, ctx)
	e.dragByHandle(ctx)
	if e.onClick != nil && ctx.In.Hit(e.rect) {
		e.onClick(ctx.In.X, ctx.In.Y)
	}
	if e.onDouble != nil && ctx.In.Clicks == 2 && ctx.In.Hit(e.rect) {
		e.onDouble()
	}
	if e.onMiddle != nil && ctx.In.MiddleClick && ctx.In.Over(e.rect) {
		e.onMiddle()
	}
	if e.onDragOver != nil && (ctx.In.Held || ctx.In.MiddleHeld) && ctx.In.Over(e.rect) {
		e.onDragOver()
	}
	if e.onRight != nil && ctx.In.RightClick && ctx.In.Over(e.rect) {
		e.onRight(ctx, ctx.In.X, ctx.In.Y)
		ctx.In.RightClick = false // one right click opens one menu: the innermost wins
	}
	if e.modal {
		// A modal overlay owns the pointer: nothing under it gets clicks,
		// wheel or hover, and a click outside it dismisses it.
		if (ctx.In.Click || ctx.In.RightClick) && !e.rect.Contains(ctx.In.X, ctx.In.Y) && !e.nearRect(ctx.In.X, ctx.In.Y) && e.onDismiss != nil {
			e.onDismiss()
		}
		return true
	}
	return false
}

func (e *El) draw(ctx *core.Ctx, dst *ebiten.Image) {
	if e.hidden {
		return
	}
	r := e.rect
	if e.modal {
		defer func(was bool) { ctx.HoverOff = was }(ctx.HoverOff)
		ctx.HoverOff = false // hover works inside the overlay
	}
	style := ctx.Th.Style(e.control)
	if e.shadow {
		sh := ctx.Color(core.Shadow)
		ctx.FillStyled(dst, core.Rect{X: r.X + core.ShadowOffset, Y: r.Y + core.ShadowOffset, W: r.W, H: r.H}, sh, sh, style.Radius)
	}
	over := ctx.Hovered(r)
	e.hover = over
	if over && e.tip != "" {
		ctx.TipText = e.tip // children draw later, so the innermost tooltip wins
	}
	if e.bg != "" {
		clr := ctx.Color(e.bg)
		if e.lighten && over {
			clr = core.Lighten(clr)
		}
		top, bottom := core.GradientFor(clr, style)
		if e.roundCorners != 0 {
			ctx.FillStyledCorners(dst, r, top, bottom, ctx.Radius(e.roundCtl), e.roundCorners)
		} else {
			ctx.FillStyled(dst, r, top, bottom, style.Radius)
		}
	}
	if e.hoverBg != "" && over {
		hv := ctx.Color(e.hoverBg)
		ctx.FillStyled(dst, r, hv, hv, style.Radius)
	}
	if e.borderT != "" {
		ctx.HLine(dst, r.X, r.Y, r.W, ctx.Color(e.borderT))
	}
	if e.borderL != "" {
		ctx.VLine(dst, r.X, r.Y+e.borderInset, r.H-2*e.borderInset, ctx.Color(e.borderL))
	}
	if e.borderR != "" {
		ctx.VLine(dst, r.X+r.W-1, r.Y+e.borderInset, r.H-2*e.borderInset, ctx.Color(e.borderR))
	}
	if e.borderB != "" {
		ctx.HLine(dst, r.X, r.Y+r.H-1, r.W, ctx.Color(e.borderB))
	}

	e.b.Draw(e, ctx, dst)
	kidsDst := dst
	if e.clip {
		kidsDst = ctx.Clip(dst, r)
	}
	for _, k := range e.kids {
		k.draw(ctx, kidsDst)
	}
	border := e.border
	if border == "" {
		border = style.Border
	}
	if border != "" { // after the children, so striped rows do not paint over it
		ctx.OutlineStyled(dst, r, ctx.Color(border), style.Radius)
	}
	if o, ok := e.b.(OverDrawer); ok {
		o.DrawOver(e, ctx, dst)
	}

	if e.edgeTok != "" { // the open tab's outline: top, left and right, bottom left open
		col := ctx.Color(e.edgeTok)
		if e.edgeTop {
			ctx.Fill(dst, core.Rect{X: r.X, Y: r.Y, W: r.W, H: core.EdgeW}, col)
		}
		if e.edgeLeft {
			ctx.Fill(dst, core.Rect{X: r.X, Y: r.Y, W: core.EdgeW, H: r.H}, col)
		}
		if e.edgeRight {
			ctx.Fill(dst, core.Rect{X: r.X + r.W - core.EdgeW, Y: r.Y, W: core.EdgeW, H: r.H}, col)
		}
	}
}

// findScope is the last FocusScope element in tree order, the one on top.
func findScope(element *El) *El {
	if element.hidden {
		return nil
	}
	var found *El
	if element.focusScope {
		found = element
	}
	for _, k := range element.kids {
		if inner := findScope(k); inner != nil {
			found = inner
		}
	}
	return found
}

// dragByHandle moves a Moved element while the pointer drags one of its
// handles: a press on a handle starts it (and shows the grab cursor, only
// from then until the release), the offset follows the pointer
// until the button is released, and a double click puts the element back in
// the center.
func (e *El) dragByHandle(ctx *core.Ctx) {
	m := e.fl.moved
	if m == nil {
		return
	}
	if m.active {
		if !ctx.In.Held {
			m.active = false
			return
		}
		ctx.Cursor = core.CursorMove
		m.X, m.Y = m.fromX+ctx.In.X-m.startX, m.fromY+ctx.In.Y-m.startY
		return
	}
	if !e.rect.Contains(ctx.In.X, ctx.In.Y) || e.onBorder(ctx.In.X, ctx.In.Y) || !e.onHandle(ctx.In.X, ctx.In.Y) {
		return
	}
	if !ctx.In.Click {
		return
	}
	if ctx.In.Clicks >= 2 {
		m.X, m.Y = 0, 0
		return
	}
	m.active, m.startX, m.startY, m.fromX, m.fromY = true, ctx.In.X, ctx.In.Y, m.X, m.Y
	ctx.Cursor = core.CursorMove // grabbed: shown from the press, not while hovering
}

// onHandle reports whether the point is on a drag handle of the element that no
// clickable or focusable element inside the handle covers.
func (e *El) onHandle(posX, posY int) bool {
	if e.hidden {
		return false
	}
	if e.dragHandle && e.rect.Contains(posX, posY) {
		return !e.coversWithControl(posX, posY)
	}
	for _, k := range e.kids {
		if k.onHandle(posX, posY) {
			return true
		}
	}
	return false
}

// coversWithControl reports whether a descendant that takes clicks or focus
// is at the point.
func (e *El) coversWithControl(posX, posY int) bool {
	for _, k := range e.kids {
		if k.hidden || !k.rect.Contains(posX, posY) {
			continue
		}
		if k.onClick != nil || k.onDouble != nil || k.focusable() || k.coversWithControl(posX, posY) {
			return true
		}
	}
	return false
}

// nearRect reports whether the point is outside e's rect but within its
// dismiss margin.
func (e *El) nearRect(posX, posY int) bool {
	m := e.dismissMargin
	return m > 0 && posX >= e.rect.X-m && posX < e.rect.X+e.rect.W+m && posY >= e.rect.Y-m && posY < e.rect.Y+e.rect.H+m
}

// onBorder reports whether the point is within dragEdge of e's border.
func (e *El) onBorder(posX, posY int) bool {
	r := e.rect
	return posX < r.X+dragEdge || posX >= r.X+r.W-dragEdge || posY < r.Y+dragEdge || posY >= r.Y+r.H-dragEdge
}
