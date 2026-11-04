package vml

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/ui/widget"
)

// hscrollArrowW is the width of the arrow buttons at the ends of an
// overflowing HScroll.
const hscrollArrowW = 18

// hscrollB lays its children out in a row at their natural width, wider
// than the box when they do not fit, and shows the part at ScrollState.X.
// When it overflows, an arrow at each end that can still move scrolls by
// half a box, and the wheel scrolls sideways.
type hscrollB struct {
	flow
	state   *ScrollState
	content int // natural width of the children, set by layout
}

// Reveal marks the child an HScroll must keep in view: whenever a
// different child is flagged than last time, the strip scrolls to show
// it once, so a selection made elsewhere is never hidden, while the
// user's own scrolling is left alone.
func Reveal() Option { return func(e *El) { e.reveal = true } }

func (s *hscrollB) overflow(e *El) bool { return s.content > e.rect.W }

func (s *hscrollB) maxX(e *El) int { return max(s.content-e.rect.W, 0) }

func (s *hscrollB) layout(e *El, c *widget.Ctx) {
	r := e.rect
	s.content, _ = s.flow.measure(e, c, 1<<20)
	st := s.state

	flagged, x := -1, e.padL
	var start, end int
	for i, k := range e.kids {
		if k.hidden || k.float {
			continue
		}
		w, _ := k.size(c, 1<<20)
		if k.reveal && flagged < 0 {
			flagged, start, end = i, x, x+w
		}
		x += w + e.gap
	}
	switch {
	case flagged < 0:
		st.revealed = 0
	case st.revealed != flagged+1:
		st.revealed = flagged + 1
		room := 0
		if s.overflow(e) {
			room = hscrollArrowW
		}
		if start-room < st.X {
			st.X = start - room
		}
		if end+room > st.X+r.W {
			st.X = end + room - r.W
		}
	}
	st.X = max(0, min(st.X, s.maxX(e)))

	e.rect = widget.Rect{X: r.X - st.X, Y: r.Y, W: max(s.content, r.W), H: r.H}
	s.flow.layout(e, c)
	e.rect = r
}

func (s *hscrollB) arrows(e *El) (left, right widget.Rect) {
	r := e.rect
	left = widget.Rect{X: r.X, Y: r.Y, W: hscrollArrowW, H: r.H}
	right = widget.Rect{X: r.X + r.W - hscrollArrowW, Y: r.Y, W: hscrollArrowW, H: r.H}
	return
}

// preUpdate takes clicks on the arrows before the children under them see them.
func (s *hscrollB) preUpdate(e *El, c *widget.Ctx) {
	in, st, r := c.In, s.state, e.rect
	if !s.overflow(e) || !in.Click {
		return
	}
	left, right := s.arrows(e)
	switch {
	case st.X > 0 && left.Contains(in.X, in.Y):
		st.X = max(st.X-r.W/2, 0)
		c.In.Click = false
	case st.X < s.maxX(e) && right.Contains(in.X, in.Y):
		st.X = min(st.X+r.W/2, s.maxX(e))
		c.In.Click = false
	}
}

func (s *hscrollB) update(e *El, c *widget.Ctx) {
	in, st := c.In, s.state
	if s.overflow(e) && in.Wheel != 0 && in.Over(e.rect) {
		st.X = max(0, min(st.X+int(-in.Wheel*wheelStep), s.maxX(e)))
		c.In.Wheel = 0
	}
}

func (s *hscrollB) drawOver(e *El, c *widget.Ctx, dst *ebiten.Image) {
	if !s.overflow(e) {
		return
	}
	left, right := s.arrows(e)
	for _, a := range []struct {
		box  widget.Rect
		show bool
		dir  float32
	}{{left, s.state.X > 0, -1}, {right, s.state.X < s.maxX(e), 1}} {
		if !a.show {
			continue
		}
		c.Fill(dst, a.box, c.Col(widget.Bg))
		clr := c.Col(widget.Muted)
		if c.Hovered(a.box) {
			clr = c.Col(widget.Text)
		}
		cx, cy := float32(a.box.X+a.box.W/2), float32(a.box.Y+a.box.H/2)
		c.Line(dst, cx-2*a.dir, cy-4, cx+2*a.dir, cy, 1.5, clr)
		c.Line(dst, cx+2*a.dir, cy, cx-2*a.dir, cy+4, 1.5, clr)
	}
}

// HScroll is a horizontally scrolling row: its children keep their
// natural width and may be wider than the box, which clips them and
// shows arrows at the ends. Mark the child to keep in view with Reveal.
func HScroll(state *ScrollState, items ...Item) *El {
	return newEl(&hscrollB{flow: flow{horiz: true}, state: state}, append([]Item{Clip()}, items...))
}
