package widget

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
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
	Linear
	state   *ScrollState
	content int // natural width of the children, set by layout
	origin  int // x of the shifted rect the children were laid out in
}

// overflow reports whether the children are wider than the box.
func (s *hscrollB) overflow(element *El) bool { return s.content > element.Rect().W }

// inset is the room kept clear at each end for the arrows, so the first
// and last tab are not hidden under them; none when everything fits.
func (s *hscrollB) inset(element *El) int {
	if s.overflow(element) {
		return hscrollArrowW
	}
	return 0
}

func (s *hscrollB) maxX(element *El) int {
	if !s.overflow(element) {
		return 0
	}
	return s.content + 2*s.inset(element) - element.Rect().W
}

// Layout places the children for the scroll position the state holds,
// clamped; it does not change the state (settle does, in update).
func (s *hscrollB) Layout(element *El, ctx *core.Ctx) {
	r := element.Rect()
	s.content, _ = s.Linear.Measure(element, ctx, 1<<20)
	x := max(0, min(s.state.X, s.maxX(element)))
	s.origin = r.X - x + s.inset(element)
	s.LayoutIn(element, ctx, core.Rect{X: s.origin, Y: r.Y, W: max(s.content, r.W), H: r.H})
}

// settle is what the layout found, written to the state before the frame's
// input is handled: whether the content overflows, the scroll position
// clamped, and the position that brings the child marked KeepInView into
// view when that child changed.
func (s *hscrollB) settle(element *El, ctx *core.Ctx) {
	st := s.state
	st.Overflowing = s.overflow(element)
	flagged := -1
	var start, end int
	for i, k := range element.Children() {
		if !k.InFlow() {
			continue
		}
		if kr := k.Rect(); k.KeepsInView() && flagged < 0 {
			flagged, start, end = i, kr.X-s.origin, kr.X+kr.W-s.origin
		}
	}
	switch {
	case flagged < 0:
		st.revealed = 0
	case st.revealed != flagged+1:
		st.revealed = flagged + 1
		if start < st.X { // the clear area starts one arrow in, where the content starts
			st.X = start
		}
		if end+2*s.inset(element) > st.X+element.Rect().W {
			st.X = end + 2*s.inset(element) - element.Rect().W
		}
	}
	st.X = max(0, min(st.X, s.maxX(element)))
}

func (s *hscrollB) arrows(element *El) (left, right core.Rect) {
	r := element.Rect()
	left = core.Rect{X: r.X, Y: r.Y, W: hscrollArrowW, H: r.H}
	right = core.Rect{X: r.X + r.W - hscrollArrowW, Y: r.Y, W: hscrollArrowW, H: r.H}
	return
}

// arrowsShown reports which arrows are drawn: only those with something
// to scroll to that way, so none when everything fits and one at an end.
func (s *hscrollB) arrowsShown(element *El) (left, right bool) {
	return s.overflow(element) && s.state.X > 0, s.overflow(element) && s.state.X < s.maxX(element)
}

// PreUpdate takes clicks on the arrows before the children under them see them.
func (s *hscrollB) PreUpdate(element *El, ctx *core.Ctx) {
	s.settle(element, ctx)
	in, st, r := ctx.In, s.state, element.Rect()
	if !s.overflow(element) {
		return
	}
	left, right := s.arrows(element)
	showL, showR := s.arrowsShown(element)
	step := func(dir, by int) { st.X = max(0, min(st.X+dir*by, s.maxX(element))) }
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
			ctx.In.Click = false
		case showR && right.Contains(in.X, in.Y):
			st.holdDir, st.holdN = 1, 0
			step(1, r.W/2)
			ctx.In.Click = false
		default:
			if in.Over(r) { // a press on the strip itself may become a pan; the tab under it still gets the click
				st.pressed, st.panning, st.panFrom, st.panX0 = true, false, in.X, st.X
			}
		}
		return
	}
	if st.pressed && in.Held { // dragging the strip: the content follows the pointer
		dx := ctx.PointerX - st.panFrom
		if st.panning || dx >= hscrollPanSlop || dx <= -hscrollPanSlop {
			st.panning = true
			st.X = max(0, min(st.panX0-dx, s.maxX(element)))
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

// Update scrolls with the wheel and with a touchpad's sideways swipe.
func (s *hscrollB) Update(element *El, ctx *core.Ctx) {
	in, st := ctx.In, s.state
	if !s.overflow(element) || !in.Over(element.Rect()) {
		return
	}
	switch {
	case in.WheelX != 0:
		st.X = max(0, min(st.X+int(-in.WheelX*wheelStep), s.maxX(element)))
		ctx.In.WheelX = 0
	case in.Wheel != 0:
		st.X = max(0, min(st.X+int(-in.Wheel*wheelStep), s.maxX(element)))
		ctx.In.Wheel = 0
	}
}

func (s *hscrollB) DrawOver(element *El, ctx *core.Ctx, dst *ebiten.Image) {
	left, right := s.arrows(element)
	showL, showR := s.arrowsShown(element)
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
		ctx.Fill(dst, fill, ctx.Color(core.Frame))
		clr := ctx.Color(core.Muted)
		if ctx.Hovered(a.box) {
			clr = ctx.Color(core.Text)
		}
		cx, cy := float32(a.box.X+a.box.W/2), float32(a.box.Y+a.box.H/2)
		ctx.Line(dst, cx-2*a.dir, cy-4, cx+2*a.dir, cy, 1.5, clr)
		ctx.Line(dst, cx+2*a.dir, cy, cx-2*a.dir, cy+4, 1.5, clr)
	}
}

// HScroll is a horizontally scrolling row: its children keep their
// natural width and may be wider than the box, which clips them and
// shows arrows at the ends that have more to scroll to. Mark the child to
// keep in view with KeepInView.
func HScroll(state *ScrollState, items ...Item) *El {
	return Custom(&hscrollB{Linear: Linear{Horizontal: true}, state: state}, append([]Item{Clip()}, items...)...)
}
