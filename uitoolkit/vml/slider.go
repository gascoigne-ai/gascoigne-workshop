package vml

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// sliderPad is the room at each end of the track for half the thumb.
const sliderPad = 7

// SliderState is the drag of a Slider. The app owns it.
//
// A drag follows the pointer by how far it moved in device pixels since
// the press, not by where the track is: the value may resize the whole UI
// while the thumb is held, and a track that moves under the pointer must
// not feed back into the value.
type SliderState struct {
	Dragging bool
	pending  int
	startDev float64 // pointer position at the press, device pixels
	startF   float64 // thumb position at the press, 0..1
	trackDev float64 // track length at the press, device pixels
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
	mid            int // value at the middle of the track; 0 for a linear track
	value          int
	onChange       func(int)
	onEnd          func(int)
}

func (s *sliderB) track(e *El) (x, w int) {
	return e.rect.X + sliderPad, max(e.rect.W-2*sliderPad, 1)
}

// valueAt maps a track fraction (0..1) to a stepped value. With a mid
// value the two halves of the track are scaled separately, so the mid
// value sits in the middle.
func (s *sliderB) valueAt(f float64) int {
	f = max(0, min(f, 1))
	lo, hi, g := float64(s.min), float64(s.max), f
	if s.mid != 0 {
		if f < 0.5 {
			hi, g = float64(s.mid), f*2
		} else {
			lo, g = float64(s.mid), (f-0.5)*2
		}
	}
	v := lo + g*(hi-lo)
	steps := int((v-float64(s.min))/float64(s.step) + 0.5)
	return max(s.min, min(s.min+steps*s.step, s.max))
}

// fracOf is the track fraction of value v.
func (s *sliderB) fracOf(v int) float64 {
	v = max(s.min, min(v, s.max))
	if s.mid == 0 {
		return float64(v-s.min) / float64(s.max-s.min)
	}
	if v < s.mid {
		return 0.5 * float64(v-s.min) / float64(s.mid-s.min)
	}
	return 0.5 + 0.5*float64(v-s.mid)/float64(s.max-s.mid)
}

func (s *sliderB) measure(*El, *widget.Ctx, int) (int, int) { return 120, 20 }

func (s *sliderB) update(e *El, c *widget.Ctx) {
	in, st := c.In, s.state
	switch {
	case st.Dragging:
		if in.Held {
			f := st.startF + (float64(c.RawX)*c.Scale-st.startDev)/st.trackDev
			if v := s.valueAt(f); v != st.pending {
				st.pending = v
				if s.onChange != nil {
					s.onChange(v)
				}
			}
			return
		}
		st.Dragging = false
		if s.onEnd != nil {
			s.onEnd(st.pending)
		}
	case in.Click && in.Over(e.rect):
		x, w := s.track(e)
		st.Dragging = true
		st.startF = max(0, min(float64(in.X-x)/float64(w), 1))
		st.startDev, st.trackDev = float64(in.X)*c.Scale, float64(w)*c.Scale
		st.pending = s.valueAt(st.startF)
		if st.pending != s.value && s.onChange != nil {
			s.onChange(st.pending)
		}
		c.In.Click = false
	}
}

func (s *sliderB) draw(e *El, c *widget.Ctx, dst *ebiten.Image) {
	r := e.rect
	x, w := s.track(e)
	frac := s.fracOf(s.state.Shown(s.value))
	mid := r.Y + r.H/2
	pill := float64(2) * min(c.Th.RadiusScale, 1) // the track is square when corners are
	chip, accent := c.Col(widget.Chip), c.Col(widget.Accent)
	c.FillStyled(dst, widget.Rect{X: x, Y: mid - 2, W: w, H: 4}, chip, chip, pill)
	if s.mid != 0 { // a tick at the middle value
		c.Fill(dst, widget.Rect{X: x + w/2, Y: mid - 5, W: 1, H: 10}, c.Col(widget.Rule))
	}
	c.FillStyled(dst, widget.Rect{X: x, Y: mid - 2, W: int(frac*float64(w) + 0.5), H: 4}, accent, accent, pill)
	tx := x + int(frac*float64(w)+0.5)
	thumb := widget.Rect{X: tx - 5, Y: mid - 8, W: 10, H: 16}
	clr := widget.Accent
	if s.state.Dragging || c.Hovered(r) {
		clr = widget.AccentLine
	}
	tc := c.Col(clr)
	c.FillStyled(dst, thumb, tc, tc, c.Radius("slider"))
	c.OutlineStyled(dst, thumb, c.Col(widget.Border), c.Radius("slider"))
}

// Slider picks an integer from min to max in steps, by dragging. onChange
// runs with each new value while the thumb moves, so the effect can be
// seen live.
func Slider(state *SliderState, min, max, step, value int, onChange func(int), opts ...Option) *El {
	return newEl(&sliderB{state: state, min: min, max: max, step: step, value: value, onChange: onChange}, with([]Item{Grow(1), H(widget.CompactH)}, opts))
}

// SliderMid puts value mid in the middle of the slider's track, scaling
// the two halves separately.
func SliderMid(mid int) Option {
	return func(e *El) {
		if s, ok := e.b.(*sliderB); ok {
			s.mid = mid
		}
	}
}

// OnSlideEnd runs with the final value when the thumb is released.
func OnSlideEnd(f func(int)) Option {
	return func(e *El) {
		if s, ok := e.b.(*sliderB); ok {
			s.onEnd = f
		}
	}
}
