package view

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// handleSlop is how many pixels each side of the divider still grab it.
const handleSlop = 2

// SplitState is the divider position of a Split. The app owns it so the
// position survives the view being rebuilt every frame (and can be saved).
type SplitState struct {
	Size float64 // size of the first part: pixels, or a fraction of the container with AsRatio; 0 = the default

	dragging bool
	grab     int // pointer offset inside the divider when the drag began
}

// splitB lays two children out around a one-pixel divider that can be dragged.
type splitB struct {
	plain
	state   *SplitState
	def     float64
	divider core.Rect
}

func (s *splitB) axis(e *El) (origin, total int) {
	if e.split.vertical {
		return e.rect.Y, e.rect.H
	}
	return e.rect.X, e.rect.W
}

// firstSize is the pixel size of the first child: the stored size or the
// default, as pixels or a fraction, held between the two minimums.
func (s *splitB) firstSize(e *El, avail int) int {
	size := s.state.Size
	if size == 0 {
		size = s.def
	}
	px := int(size)
	if e.split.ratio {
		px = int(size * float64(avail))
	}
	hi := max(avail-e.split.minSecond, e.split.minFirst)
	return min(max(px, e.split.minFirst), hi)
}

func (s *splitB) layout(e *El, c *core.Ctx) {
	if len(e.kids) != 2 {
		return
	}
	r := e.rect
	_, total := s.axis(e)
	avail := max(total-1, 0)
	px := min(s.firstSize(e, avail), avail)
	if e.split.vertical {
		s.divider = core.Rect{X: r.X, Y: r.Y + px, W: r.W, H: 1}
		e.kids[0].place(c, core.Rect{X: r.X, Y: r.Y, W: r.W, H: px})
		e.kids[1].place(c, core.Rect{X: r.X, Y: r.Y + px + 1, W: r.W, H: avail - px})
		return
	}
	s.divider = core.Rect{X: r.X + px, Y: r.Y, W: 1, H: r.H}
	e.kids[0].place(c, core.Rect{X: r.X, Y: r.Y, W: px, H: r.H})
	e.kids[1].place(c, core.Rect{X: r.X + px + 1, Y: r.Y, W: avail - px, H: r.H})
}

func (s *splitB) draw(e *El, c *core.Ctx, dst *ebiten.Image) {
	c.Fill(dst, s.divider, c.Color(core.Border))
}

// handle is the area that grabs the divider, a little wider than it.
func (s *splitB) handle(e *El) core.Rect {
	d := s.divider
	if e.split.vertical {
		return core.Rect{X: d.X, Y: d.Y - handleSlop, W: d.W, H: 1 + 2*handleSlop}
	}
	return core.Rect{X: d.X - handleSlop, Y: d.Y, W: 1 + 2*handleSlop, H: d.H}
}

func (s *splitB) update(e *El, c *core.Ctx) {
	if len(e.kids) != 2 {
		return
	}
	in, st := c.In, s.state
	cursor, pointer, divPos := core.CursorResizeH, in.X, s.divider.X
	if e.split.vertical {
		cursor, pointer, divPos = core.CursorResizeV, in.Y, s.divider.Y
	}
	origin, total := s.axis(e)
	avail := max(total-1, 1)

	if st.dragging {
		if !in.Held {
			st.dragging = false
			return
		}
		c.Cursor = cursor
		px := float64(pointer - origin - st.grab)
		if e.split.ratio {
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
	st.dragging, st.grab = true, pointer-divPos
}

// Split puts its two children side by side (or one above the other with
// Vertical) with a draggable one-pixel divider between them. def is the
// size of the first child until the user drags: pixels, or a fraction of
// the container with AsRatio. Double click on the divider resets it.
func Split(state *SplitState, def float64, items ...Item) *El {
	return newEl(&splitB{state: state, def: def}, items)
}

// Vertical makes a Split stack its children.
func Vertical() Option { return func(e *El) { e.split.vertical = true } }

// AsRatio makes a Split's size a fraction of the container instead of pixels.
func AsRatio() Option { return func(e *El) { e.split.ratio = true } }

// MinFirst is the smallest size in pixels of a Split's first child.
func MinFirst(n int) Option { return func(e *El) { e.split.minFirst = n } }

// MinSecond is the smallest size in pixels of a Split's second child.
func MinSecond(n int) Option { return func(e *El) { e.split.minSecond = n } }
