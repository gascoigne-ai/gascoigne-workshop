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

	drag barDrag // the thumb being dragged
}

// scrollB lays its children out as a column taller than the box and
// shows the part at ScrollState.Y, with a scrollbar when it overflows.
type scrollB struct {
	Linear
	state   *ScrollState
	content int // height of the content, set by layout
}

// Layout places the children at their natural height, shifted up by the
// scroll position the state holds, clamped to the content.
func (s *scrollB) Layout(element *El, ctx *core.Ctx) {
	r := element.Rect()
	_, s.content = s.Linear.Measure(element, ctx, element.Inner().W)
	y := clampScroll(s.state.Y, s.content, r.H) // update writes it back

	width := r.W
	if s.content > r.H {
		width -= scrollbarW
	}
	s.LayoutIn(element, ctx, core.Rect{X: r.X, Y: r.Y - y, W: width, H: max(s.content, r.H)})
}

func (s *scrollB) Update(element *El, ctx *core.Ctx) {
	in, st, r := ctx.In, s.state, element.Rect()
	st.Y = clampScroll(st.Y, s.content, r.H)
	sizes := s.bars(element)
	wasDragging := st.drag.active()
	st.drag.update(ctx, sizes, 0, s.content, &st.X, &st.Y)
	if wasDragging || in.Click && sizes.vTrack.Contains(in.X, in.Y) {
		return
	}
	if in.Wheel != 0 && in.Over(r) && s.content > r.H {
		st.Y = clampScroll(st.Y+int(-in.Wheel*wheelStep), s.content, r.H)
		ctx.In.Wheel = 0 // consumed, so an enclosing scroller does not move too
	}
}

// bars is the scrollbar layout of the box: a vertical bar along its right edge.
func (s *scrollB) bars(element *El) bars { return layoutBars(element.Rect(), 0, s.content) }

func (s *scrollB) DrawOver(element *El, ctx *core.Ctx, dst *ebiten.Image) {
	drawBars(ctx, dst, s.bars(element), s.state.drag, 0, s.content, 0, s.state.Y)
}

// Scroll is a vertically scrolling column: its children may be taller
// than the box, the wheel and a draggable scrollbar move them, and
// children scrolled out of view do not receive clicks. It grows to fill
// its parent unless given a size. In a template it is <scroll state>, with
// the state named from the scope.
func Scroll(state *ScrollState, items ...Item) *El {
	b := &scrollB{state: state}
	return Custom(b, append([]Item{Grow(1), Clip()}, items...)...)
}

// ScrollTo moves the scroll position just far enough to bring target, a
// rect from this frame's layout, inside the box: what Tab moving the
// focus to a field out of view needs. The box shows it on the next frame.
func (s *scrollB) ScrollTo(element *El, target core.Rect) {
	view := element.Rect()
	switch {
	case target.Y < view.Y:
		s.state.Y = max(s.state.Y-(view.Y-target.Y), 0)
	case target.Y+target.H > view.Y+view.H:
		s.state.Y += target.Y + target.H - (view.Y + view.H)
	}
}
