package view

import "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"

// floatMode is how a floating element is placed in its Stack.
type floatMode uint8

const (
	floatCorner floatMode = iota // top right, offset by FloatTR
	floatPoint                   // top-left at a point, kept inside (FloatAt)
	floatMiddle                  // centered (FloatCenter)
)

// FloatAt takes the element out of its Stack's layout and puts its
// top-left corner at posX, posY inside the Stack, shifted to stay inside it:
// left of the edge when it would stick out on the right, above y when it
// would stick out at the bottom.
func FloatAt(posX, posY int) Option {
	return func(element *El) {
		element.fl.on, element.fl.mode, element.fl.x, element.fl.y = true, floatPoint, posX, posY
	}
}

// FloatCenter takes the element out of its Stack's layout and centers it.
func FloatCenter() Option {
	return func(element *El) { element.fl.on, element.fl.mode = true, floatMiddle }
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
	return func(element *El) { element.fl.moved = state }
}

// FocusScope makes Tab cycle only through the focusable elements inside
// this one while it exists, as a dialog needs.
func FocusScope() Option { return func(element *El) { element.focusScope = true } }

// dragEdge is how far from the border of a Moved element, inside it, a
// press does not start a drag: the edge is the resize grab, not the handle.
const dragEdge = core.EdgeGrab

// RoundT rounds the top corners of the element's background to the corner
// radius of the named control (the panel it sits at the top of, such as
// "menu"), with anti-aliased edges, so a bar across the top of a rounded
// panel does not poke out past the panel's corners. It follows the user's
// corner setting like the panel does.
//
// The Round family is not the per-side family of rule 1: it holds only the
// corner masks some panel needed (top, bottom, bottom left), and a new mask
// is added with the panel that needs it, not ahead (an accepted exception,
// listed in .harness/architecture/toolkit.md).
func RoundT(control string) Option { return roundCorners(control, core.CornersTop) }

// RoundB is RoundT for the bottom corners: a pane across the bottom of a
// rounded panel.
func RoundB(control string) Option { return roundCorners(control, core.CornersBottom) }

// RoundBL is RoundT for the bottom left corner only: a column down the left
// side of a rounded panel.
func RoundBL(control string) Option { return roundCorners(control, core.CornerBL) }

func roundCorners(control string, corners core.Corners) Option {
	return func(element *El) { element.roundCtl, element.roundCorners = control, element.roundCorners|corners }
}

// DragHandle marks the element as a place to drag its Moved ancestor by:
// a dialog's title bar. Clickable and focusable elements inside it (a
// close button, a search field) keep their clicks.
func DragHandle() Option { return func(element *El) { element.dragHandle = true } }
