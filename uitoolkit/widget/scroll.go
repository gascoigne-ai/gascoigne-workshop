package widget

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
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

	drag grab // the thumb being dragged
}

// scrollB lays its children out as a column taller than the box and
// shows the part at ScrollState.Y, with a scrollbar when it overflows.
type scrollB struct {
	Linear
	state   *ScrollState
	content int // height of the content, set by layout
}

// measure is the content's size, but no taller than MaxH, so a
// scroller in a floating or hugging parent shrinks to its content and
// scrolls only when the content is taller than the limit.
func (s *scrollB) track(e *El) core.Rect {
	r := e.Rect()
	return core.Rect{X: r.X + r.W - scrollbarW, Y: r.Y, W: scrollbarW, H: r.H}
}

func (s *scrollB) Layout(e *El, c *core.Ctx) {
	r := e.Rect()
	_, s.content = s.Linear.Measure(e, c, e.Inner().W)
	y := clampScroll(s.state.Y, s.content, r.H) // update writes it back

	width := r.W
	if s.content > r.H {
		width -= scrollbarW
	}
	s.LayoutIn(e, c, core.Rect{X: r.X, Y: r.Y - y, W: width, H: max(s.content, r.H)})
}

func (s *scrollB) Update(e *El, c *core.Ctx) {
	in, st, r := c.In, s.state, e.Rect()
	st.Y = clampScroll(st.Y, s.content, r.H)
	track := s.track(e)
	if st.drag.on {
		if st.drag.held(in) {
			st.Y = posFromOffset(track.H, s.content, r.H, in.Y-track.Y-st.drag.off)
		}
		return
	}
	if in.Click && track.Contains(in.X, in.Y) {
		s.press(track, r.H, in.Y-track.Y)
		return
	}
	if in.Wheel != 0 && in.Over(r) && s.content > r.H {
		st.Y = clampScroll(st.Y+int(-in.Wheel*wheelStep), s.content, r.H)
		c.In.Wheel = 0 // consumed, so an enclosing scroller does not move too
	}
}

// press starts dragging the thumb when the press is on it, else pages
// toward the press.
func (s *scrollB) press(track core.Rect, view, at int) {
	st := s.state
	grabbed, off, pos := pressThumb(track.H, s.content, view, st.Y, at)
	st.Y = pos
	if grabbed {
		st.drag.begin(off)
	}
}

func (s *scrollB) DrawOver(e *El, c *core.Ctx, dst *ebiten.Image) {
	track := s.track(e)
	drawScrollbar(c, dst, track, s.content, e.Rect().H, s.state.Y, s.state.drag.on || c.Hovered(track))
}

// Scroll is a vertically scrolling column: its children may be taller
// than the box, the wheel and a draggable scrollbar move them, and
// children scrolled out of view do not receive clicks. It grows to fill
// its parent unless given a size. It has no template attribute: its state
// is a Go value.
func Scroll(state *ScrollState, items ...Item) *El {
	b := &scrollB{state: state}
	return Custom(b, append([]Item{Grow(1), Clip()}, items...)...)
}

// ScrollTo moves the scroll position just far enough to bring target, a
// rect from this frame's layout, inside the box: what Tab moving the
// focus to a field out of view needs. The box shows it on the next frame.
func (s *scrollB) ScrollTo(e *El, target core.Rect) {
	view := e.Rect()
	switch {
	case target.Y < view.Y:
		s.state.Y = max(s.state.Y-(view.Y-target.Y), 0)
	case target.Y+target.H > view.Y+view.H:
		s.state.Y += target.Y + target.H - (view.Y + view.H)
	}
}
