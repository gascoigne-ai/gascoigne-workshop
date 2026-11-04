package vml

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

// FocusScope makes Tab cycle only through the focusable elements inside
// this one while it exists, as a dialog needs.
func FocusScope() Option { return func(e *El) { e.focusScope = true } }

// OnRightClick runs f with the pointer position when the element is
// right-clicked.
func OnRightClick(f func(x, y int)) Option { return func(e *El) { e.onRight = f } }
