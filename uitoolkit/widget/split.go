package widget

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// SplitState is the divider position of a Split. The app owns it so the
// position survives the view being rebuilt every frame (and can be saved).
type SplitState struct {
	Size float64 // size of the first part: pixels, or a fraction of the container with AsRatio; 0 = the default

	drag grab // the divider being dragged
}

// splitB lays two children out around a one-pixel divider that can be dragged.
type splitB struct {
	Base
	state               *SplitState
	def                 float64
	divider             core.Rect
	vertical, ratio     bool
	minFirst, minSecond int
}

func (s *splitB) axis(e *El) (origin, total int) {
	if s.vertical {
		return e.Rect().Y, e.Rect().H
	}
	return e.Rect().X, e.Rect().W
}

// firstSize is the pixel size of the first child: the stored size or the
// default, as pixels or a fraction, held between the two minimums.
func (s *splitB) firstSize(e *El, avail int) int {
	size := s.state.Size
	if size == 0 {
		size = s.def
	}
	px := int(size)
	if s.ratio {
		px = int(size * float64(avail))
	}
	hi := max(avail-s.minSecond, s.minFirst)
	return min(max(px, s.minFirst), hi)
}

func (s *splitB) Layout(e *El, c *core.Ctx) {
	if len(e.Children()) != 2 {
		return
	}
	r := e.Rect()
	_, total := s.axis(e)
	avail := max(total-1, 0)
	px := min(s.firstSize(e, avail), avail)
	if s.vertical {
		s.divider = core.Rect{X: r.X, Y: r.Y + px, W: r.W, H: 1}
		e.Children()[0].Place(c, core.Rect{X: r.X, Y: r.Y, W: r.W, H: px})
		e.Children()[1].Place(c, core.Rect{X: r.X, Y: r.Y + px + 1, W: r.W, H: avail - px})
		return
	}
	s.divider = core.Rect{X: r.X + px, Y: r.Y, W: 1, H: r.H}
	e.Children()[0].Place(c, core.Rect{X: r.X, Y: r.Y, W: px, H: r.H})
	e.Children()[1].Place(c, core.Rect{X: r.X + px + 1, Y: r.Y, W: avail - px, H: r.H})
}

func (s *splitB) Draw(e *El, c *core.Ctx, dst *ebiten.Image) {
	c.Fill(dst, s.divider, c.Color(core.Border))
}

// handle is the area that grabs the divider, a little wider than it.
func (s *splitB) handle(e *El) core.Rect {
	d := s.divider
	if s.vertical {
		return core.Rect{X: d.X, Y: d.Y - handleSlop, W: d.W, H: 1 + 2*handleSlop}
	}
	return core.Rect{X: d.X - handleSlop, Y: d.Y, W: 1 + 2*handleSlop, H: d.H}
}

func (s *splitB) Update(e *El, c *core.Ctx) {
	if len(e.Children()) != 2 {
		return
	}
	in, st := c.In, s.state
	cursor, pointer, divPos := core.CursorResizeH, in.X, s.divider.X
	if s.vertical {
		cursor, pointer, divPos = core.CursorResizeV, in.Y, s.divider.Y
	}
	origin, total := s.axis(e)
	avail := max(total-1, 1)

	if st.drag.on {
		if !st.drag.held(in) {
			return
		}
		c.Cursor = cursor
		px := float64(pointer - origin - st.drag.off)
		if s.ratio {
			px /= float64(avail)
		}
		st.Size = max(px, 1e-6) // 0 would mean "the default"
		return
	}
	if !in.Over(s.handle(e)) {
		return
	}
	c.Cursor = cursor
	if !in.Click {
		return
	}
	if in.Clicks >= 2 {
		st.Size = 0 // back to the default
		return
	}
	st.drag.begin(pointer - divPos)
}

// Split puts its two children side by side (or one above the other with
// Vertical) with a draggable one-pixel divider between them. def is the
// size of the first child until the user drags: pixels, or a fraction of
// the container with AsRatio. Double click on the divider resets it. It has no template attribute: its
// state is a Go value.
func Split(state *SplitState, def float64, items ...Item) *El {
	return Custom(&splitB{state: state, def: def}, items...)
}

// Vertical makes a Split stack its children.
func Vertical() Option { return func(e *El) { e.Behavior().(*splitB).vertical = true } }

// AsRatio makes a Split's size a fraction of the container instead of pixels.
func AsRatio() Option { return func(e *El) { e.Behavior().(*splitB).ratio = true } }

// MinFirst is the smallest size in pixels of a Split's first child.
func MinFirst(n int) Option { return func(e *El) { e.Behavior().(*splitB).minFirst = n } }

// MinSecond is the smallest size in pixels of a Split's second child.
func MinSecond(n int) Option { return func(e *El) { e.Behavior().(*splitB).minSecond = n } }
