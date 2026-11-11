package widget

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// edgeSlop is how far from the border, inside it, a press grabs the edge.
const edgeSlop = 6

// ResizeState is the size the user gave a Resizable element; 0 means the
// element's default. The application owns it, like a split or a scroll
// state, so it survives the per-frame rebuild and can be saved.
type ResizeState struct {
	W, H int

	active                         bool
	left, right, top, bottom       bool // the edges being dragged
	startX, startY, startW, startH int
}

type resizableB struct {
	Linear     // lays its children out top to bottom, like Col
	state      *ResizeState
	minW, minH int
	maxW, maxH int
}

// Resizable lets the user drag the edges and corners of the element it
// wraps, which is laid out and centered by its parent (a floating dialog):
// the size changes by twice the pointer's movement, so the edge stays under
// the pointer. The size is kept in state within the given minimum and
// maximum; the caller applies state.W and state.H to the element's own size,
// and a double click on an edge puts both back to 0, the default. It has
// no template attribute: its state is a Go value.
func Resizable(state *ResizeState, minW, minH, maxW, maxH int, items ...Item) *El {
	return Custom(&resizableB{state: state, minW: minW, minH: minH, maxW: maxW, maxH: maxH}, items...)
}

// edges reports which edges of r the point is on, if it is on the border.
func edges(r core.Rect, x, y int) (left, right, top, bottom bool) {
	if x < r.X-2 || x > r.X+r.W+2 || y < r.Y-2 || y > r.Y+r.H+2 {
		return
	}
	left, right = x < r.X+edgeSlop, x >= r.X+r.W-edgeSlop
	top, bottom = y < r.Y+edgeSlop, y >= r.Y+r.H-edgeSlop
	return
}

// cursorFor is the resize cursor of a set of edges.
func cursorFor(left, right, top, bottom bool) core.Cursor {
	switch {
	case (left && top) || (right && bottom):
		return core.CursorResizeNWSE
	case (right && top) || (left && bottom):
		return core.CursorResizeNESW
	case left || right:
		return core.CursorResizeH
	}
	return core.CursorResizeV
}

func (b *resizableB) Update(e *El, c *core.Ctx) {
	in, st := c.In, b.state
	if st.active {
		if !in.Held {
			st.active = false
			return
		}
		c.Cursor = cursorFor(st.left, st.right, st.top, st.bottom)
		dx, dy := in.X-st.startX, in.Y-st.startY
		if st.left || st.right {
			grow := 2 * dx
			if st.left {
				grow = -grow
			}
			st.W = min(max(st.startW+grow, b.minW), b.maxW)
		}
		if st.top || st.bottom {
			grow := 2 * dy
			if st.top {
				grow = -grow
			}
			st.H = min(max(st.startH+grow, b.minH), b.maxH)
		}
		return
	}
	left, right, top, bottom := edges(e.Rect(), in.X, in.Y)
	if !(left || right || top || bottom) {
		return
	}
	c.Cursor = cursorFor(left, right, top, bottom)
	if !in.Click {
		return
	}
	if in.Clicks >= 2 {
		st.W, st.H = 0, 0
		return
	}
	st.active, st.left, st.right, st.top, st.bottom = true, left, right, top, bottom
	st.startX, st.startY, st.startW, st.startH = in.X, in.Y, e.Rect().W, e.Rect().H
	st.W, st.H = e.Rect().W, e.Rect().H // both are the user's size from now on, whichever edge moves
}
