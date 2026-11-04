package vml

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// hscrollArrowW is the width of the arrow buttons at the ends of an
// overflowing HScroll.
const hscrollArrowW = 18

// Holding an arrow scrolls once at the press, then repeats every
// hscrollHoldEvery frames after hscrollHoldDelay frames.
const (
	hscrollHoldDelay = 20
	hscrollHoldEvery = 3
	hscrollPanSlop   = 4 // pixels the pointer must move before a press on the strip pans it
)

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
func KeepInView() Option { return func(e *El) { e.reveal = true } }

func (s *hscrollB) overflow(e *El) bool { return s.content > e.rect.W }

// inset is the room kept clear at each end for the arrows, so the first
// and last tab are not hidden under them; none when everything fits.
func (s *hscrollB) inset(e *El) int {
	if s.overflow(e) {
		return hscrollArrowW
	}
	return 0
}

func (s *hscrollB) maxX(e *El) int {
	if !s.overflow(e) {
		return 0
	}
	return s.content + 2*s.inset(e) - e.rect.W
}

func (s *hscrollB) layout(e *El, c *core.Ctx) {
	r := e.rect
	s.content, _ = s.flow.measure(e, c, 1<<20)
	st := s.state
	st.Overflowing = s.overflow(e)

	flagged, x := -1, e.padL
	var start, end int
	for i, k := range e.kids {
		if k.hidden || k.fl.on {
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
		if start < st.X { // the clear area starts one arrow in, where the content starts
			st.X = start
		}
		if end+2*s.inset(e) > st.X+r.W {
			st.X = end + 2*s.inset(e) - r.W
		}
	}
	st.X = max(0, min(st.X, s.maxX(e)))

	e.rect = core.Rect{X: r.X - st.X + s.inset(e), Y: r.Y, W: max(s.content, r.W), H: r.H}
	s.flow.layout(e, c)
	e.rect = r
}

func (s *hscrollB) arrows(e *El) (left, right core.Rect) {
	r := e.rect
	left = core.Rect{X: r.X, Y: r.Y, W: hscrollArrowW, H: r.H}
	right = core.Rect{X: r.X + r.W - hscrollArrowW, Y: r.Y, W: hscrollArrowW, H: r.H}
	return
}

// arrowsShown reports which arrows are drawn: only those with something
// to scroll to that way, so none when everything fits and one at an end.
func (s *hscrollB) arrowsShown(e *El) (left, right bool) {
	return s.overflow(e) && s.state.X > 0, s.overflow(e) && s.state.X < s.maxX(e)
}

// preUpdate takes clicks on the arrows before the children under them see them.
func (s *hscrollB) preUpdate(e *El, c *core.Ctx) {
	in, st, r := c.In, s.state, e.rect
	if !s.overflow(e) {
		return
	}
	left, right := s.arrows(e)
	showL, showR := s.arrowsShown(e)
	step := func(dir, by int) { st.X = max(0, min(st.X+dir*by, s.maxX(e))) }
	if !in.Held {
		st.holdDir, st.holdN = 0, 0
	}
	if !in.Held {
		st.pressed, st.panning = false, false
	}
	if in.Click {
		switch {
		case showL && left.Contains(in.X, in.Y):
			st.holdDir, st.holdN = -1, 0
			step(-1, r.W/2)
			c.In.Click = false
		case showR && right.Contains(in.X, in.Y):
			st.holdDir, st.holdN = 1, 0
			step(1, r.W/2)
			c.In.Click = false
		default:
			if in.Over(r) { // a press on the strip itself may become a pan; the tab under it still gets the click
				st.pressed, st.panning, st.panFrom, st.panX0 = true, false, in.X, st.X
			}
		}
		return
	}
	if st.pressed && in.Held { // dragging the strip: the content follows the pointer
		dx := c.PointerX - st.panFrom
		if st.panning || dx >= hscrollPanSlop || dx <= -hscrollPanSlop {
			st.panning = true
			st.X = max(0, min(st.panX0-dx, s.maxX(e)))
		}
		return
	}
	// Held on the arrow the press started on: scroll on, a little at a time.
	arrow := left
	if st.holdDir > 0 {
		arrow = right
	}
	if st.holdDir == 0 || !in.Held || !arrow.Contains(in.X, in.Y) {
		return
	}
	st.holdN++
	if st.holdN >= hscrollHoldDelay && (st.holdN-hscrollHoldDelay)%hscrollHoldEvery == 0 {
		step(st.holdDir, max(r.W/8, 1))
	}
}

// update scrolls with the wheel and with a touchpad's sideways swipe.
func (s *hscrollB) update(e *El, c *core.Ctx) {
	in, st := c.In, s.state
	if !s.overflow(e) || !in.Over(e.rect) {
		return
	}
	switch {
	case in.WheelX != 0:
		st.X = max(0, min(st.X+int(-in.WheelX*wheelStep), s.maxX(e)))
		c.In.WheelX = 0
	case in.Wheel != 0:
		st.X = max(0, min(st.X+int(-in.Wheel*wheelStep), s.maxX(e)))
		c.In.Wheel = 0
	}
}

func (s *hscrollB) drawOver(e *El, c *core.Ctx, dst *ebiten.Image) {
	left, right := s.arrows(e)
	showL, showR := s.arrowsShown(e)
	for _, a := range []struct {
		box  core.Rect
		show bool
		dir  float32
	}{{left, showL, -1}, {right, showR, 1}} {
		if !a.show {
			continue
		}
		fill := a.box
		fill.H-- // leave the strip's bottom line running under the arrow
		c.Fill(dst, fill, c.Color(core.Frame))
		clr := c.Color(core.Muted)
		if c.Hovered(a.box) {
			clr = c.Color(core.Text)
		}
		cx, cy := float32(a.box.X+a.box.W/2), float32(a.box.Y+a.box.H/2)
		c.Line(dst, cx-2*a.dir, cy-4, cx+2*a.dir, cy, 1.5, clr)
		c.Line(dst, cx+2*a.dir, cy, cx-2*a.dir, cy+4, 1.5, clr)
	}
}

// HScroll is a horizontally scrolling row: its children keep their
// natural width and may be wider than the box, which clips them and
// shows arrows at the ends that have more to scroll to. Mark the child to
// keep in view with Reveal.
func HScroll(state *ScrollState, items ...Item) *El {
	return newEl(&hscrollB{flow: flow{horiz: true}, state: state}, append([]Item{Clip()}, items...))
}
