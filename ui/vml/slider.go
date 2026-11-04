package vml

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/ui/widget"
)

// sliderPad is the room at each end of the track for half the thumb.
const sliderPad = 7

// SliderState is the drag of a Slider. The app owns it. While the thumb
// is held the slider shows and tracks a pending value and commits it on
// release, so a value that resizes the UI does not move the slider under
// the pointer.
type SliderState struct {
	Dragging bool
	pending  int
}

// Shown is the value to display: the pending one while dragging,
// stored otherwise.
func (s *SliderState) Shown(stored int) int {
	if s.Dragging {
		return s.pending
	}
	return stored
}

type sliderB struct {
	plain
	state          *SliderState
	min, max, step int
	value          int
	onChange       func(int)
}

func (s *sliderB) track(e *El) (x, w int) {
	return e.rect.X + sliderPad, max(e.rect.W-2*sliderPad, 1)
}

// at is the stepped, clamped value under pointer position px.
func (s *sliderB) at(e *El, px int) int {
	x, w := s.track(e)
	f := float64(px-x) / float64(w)
	f = max(0, min(f, 1))
	v := float64(s.min) + f*float64(s.max-s.min)
	steps := int((v-float64(s.min))/float64(s.step) + 0.5)
	return max(s.min, min(s.min+steps*s.step, s.max))
}

func (s *sliderB) measure(*El, *widget.Ctx, int) (int, int) { return 120, 20 }

func (s *sliderB) update(e *El, c *widget.Ctx) {
	in, st := c.In, s.state
	switch {
	case st.Dragging:
		if in.Held {
			st.pending = s.at(e, in.X)
			return
		}
		st.Dragging = false
		if st.pending != s.value && s.onChange != nil {
			s.onChange(st.pending)
		}
	case in.Click && in.Over(e.rect):
		st.Dragging, st.pending = true, s.at(e, in.X)
		c.In.Click = false
	}
}

func (s *sliderB) draw(e *El, c *widget.Ctx, dst *ebiten.Image) {
	r := e.rect
	x, w := s.track(e)
	v := s.state.Shown(s.value)
	frac := float64(v-s.min) / float64(s.max-s.min)
	mid := r.Y + r.H/2
	c.Fill(dst, widget.Rect{X: x, Y: mid - 2, W: w, H: 4}, c.Col(widget.Chip))
	c.Fill(dst, widget.Rect{X: x, Y: mid - 2, W: int(frac*float64(w) + 0.5), H: 4}, c.Col(widget.Accent))
	tx := x + int(frac*float64(w)+0.5)
	thumb := widget.Rect{X: tx - 5, Y: mid - 8, W: 10, H: 16}
	clr := widget.Accent
	if s.state.Dragging || c.Hovered(r) {
		clr = widget.AccentLine
	}
	c.Fill(dst, thumb, c.Col(clr))
	c.Outline(dst, thumb, c.Col(widget.Border))
}

// Slider picks an integer from min to max in steps, by dragging. onChange
// runs with the new value when the thumb is released.
func Slider(state *SliderState, min, max, step, value int, onChange func(int), opts ...Option) *El {
	return newEl(&sliderB{state: state, min: min, max: max, step: step, value: value, onChange: onChange}, with([]Item{Grow(1), H(24)}, opts))
}
