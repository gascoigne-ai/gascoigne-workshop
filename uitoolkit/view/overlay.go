package view

// How a floating element is placed in its Stack.
const (
	floatCorner = iota // top right, offset by Float
	floatPoint         // top-left at a point, kept inside (FloatAt)
	floatMiddle        // centered (FloatCenter)
)

// FloatAt takes the element out of its Stack's layout and puts its
// top-left corner at x, y inside the Stack, shifted to stay inside it:
// left of the edge when it would stick out on the right, above y when it
// would stick out at the bottom.
func FloatAt(x, y int) Option {
	return func(e *El) { e.fl.on, e.fl.mode, e.fl.top, e.fl.right = true, floatPoint, y, x }
}

// FloatCenter takes the element out of its Stack's layout and centers it.
func FloatCenter() Option {
	return func(e *El) { e.fl.on, e.fl.mode = true, floatMiddle }
}

// MoveState is where the user dragged a centered floating element: its
// offset from the center, in pixels; the zero value is the center. The
// application owns it, like a scroll state, so it survives the per-frame
// rebuild.
type MoveState struct {
	X, Y int

	active         bool
	startX, startY int // the pointer when the drag began
	fromX, fromY   int // the offset when the drag began
}

// Moved moves a FloatCenter element by the offset in state, so the user can
// drag it; the element is kept inside its Stack.
func Moved(state *MoveState) Option {
	return func(e *El) { e.fl.moved = state }
}

// FocusScope makes Tab cycle only through the focusable elements inside
// this one while it exists, as a dialog needs.
func FocusScope() Option { return func(e *El) { e.focusScope = true } }

// OnRightClick runs f with the pointer position when the element is
// right-clicked.
func OnRightClick(f func(x, y int)) Option { return func(e *El) { e.onRight = f } }

// DismissMargin makes a click within px outside a modal element's rect
// count as a click on it, so it does not dismiss it: a resizable dialog
// grabs its edge there.
func DismissMargin(px int) Option { return func(e *El) { e.dismissMargin = px } }

// dragEdge is how far from the border of a Moved element, inside it, a
// press does not start a drag: the edge is the resize grab, not the handle.
const dragEdge = 6

// RoundT rounds the top corners of the element's background to the corner
// radius of the named control (the panel it sits at the top of, such as
// "menu"), with anti-aliased edges, so a bar across the top of a rounded
// panel does not poke out past the panel's corners.
func RoundT(control string) Option { return func(e *El) { e.roundT = control } }

// DragHandle marks the element as a place to drag its Moved ancestor by:
// a dialog's title bar. Clickable and focusable elements inside it (a
// close button, a search field) keep their clicks.
func DragHandle() Option { return func(e *El) { e.dragHandle = true } }
