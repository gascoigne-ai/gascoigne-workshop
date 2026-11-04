package vml

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// wheelStep is how many pixels one wheel notch scrolls.
const wheelStep = 48

// ScrollState is the scroll position of a Scroll container. The app
// owns it, so the position survives the view being rebuilt every frame.
type ScrollState struct {
	Y int
	X int // horizontal position, for HScroll

	// Overflowing is set by HScroll's layout: its content is wider than
	// the box. The app uses it to tell panning from reordering.
	Overflowing bool

	pressed, panning bool // HScroll: the left button went down on the strip; it moved far enough to pan
	panFrom, panX0   int  // pointer x and X at that press

	holdDir, holdN int // HScroll: the arrow held down (-1 left, 1 right) and for how many frames

	revealed int // HScroll: index+1 of the child last brought into view, 0 for none

	dragging bool // the thumb is being dragged
	grab     int  // pointer offset inside the thumb when the drag began
}

// scrollB lays its children out as a column taller than the box and
// shows the part at ScrollState.Y, with a scrollbar when it overflows.
type scrollB struct {
	flow
	state   *ScrollState
	content int // height of the content, set by layout
}

// measure is the content's size, but no taller than MaxH, so a
// scroller in a floating or hugging parent shrinks to its content and
// scrolls only when the content is taller than the limit.
func (s *scrollB) measure(e *El, c *core.Ctx, maxW int) (int, int) {
	w, h := s.flow.measure(e, c, maxW)
	return w, h
}

func (s *scrollB) track(e *El) core.Rect {
	r := e.rect
	return core.Rect{X: r.X + r.W - ScrollbarW, Y: r.Y, W: ScrollbarW, H: r.H}
}

func (s *scrollB) layout(e *El, c *core.Ctx) {
	r := e.rect
	_, s.content = s.flow.measure(e, c, e.inner().W)
	s.state.Y = ClampScroll(s.state.Y, s.content, r.H)

	width := r.W
	if s.content > r.H {
		width -= ScrollbarW
	}
	e.rect = core.Rect{X: r.X, Y: r.Y - s.state.Y, W: width, H: max(s.content, r.H)}
	s.flow.layout(e, c)
	e.rect = r
}

func (s *scrollB) update(e *El, c *core.Ctx) {
	in, st, r := c.In, s.state, e.rect
	track := s.track(e)
	if st.dragging {
		if !in.Held {
			st.dragging = false
			return
		}
		st.Y = PosFromOffset(track.H, s.content, r.H, in.Y-track.Y-st.grab)
		return
	}
	if in.Click && track.Contains(in.X, in.Y) {
		s.press(track, r.H, in.Y-track.Y)
		return
	}
	if in.Wheel != 0 && in.Over(r) && s.content > r.H {
		st.Y = ClampScroll(st.Y+int(-in.Wheel*wheelStep), s.content, r.H)
		c.In.Wheel = 0 // consumed, so an enclosing scroller does not move too
	}
}

// press starts dragging the thumb when the press is on it, else pages
// toward the press.
func (s *scrollB) press(track core.Rect, view, at int) {
	st := s.state
	off, length, ok := Thumb(track.H, s.content, view, st.Y)
	switch {
	case !ok:
	case at >= off && at < off+length:
		st.dragging, st.grab = true, at-off
	case at < off:
		st.Y = ClampScroll(st.Y-view, s.content, view)
	default:
		st.Y = ClampScroll(st.Y+view, s.content, view)
	}
}

func (s *scrollB) drawOver(e *El, c *core.Ctx, dst *ebiten.Image) {
	track := s.track(e)
	drawScrollbar(c, dst, track, s.content, e.rect.H, s.state.Y, s.state.dragging || c.Hovered(track))
}

// Scroll is a vertically scrolling column: its children may be taller
// than the box, the wheel and a draggable scrollbar move them, and
// children scrolled out of view do not receive clicks. It grows to fill
// its parent unless given a size.
func Scroll(state *ScrollState, items ...Item) *El {
	b := &scrollB{state: state}
	return newEl(b, append([]Item{Grow(1), Clip()}, items...))
}
