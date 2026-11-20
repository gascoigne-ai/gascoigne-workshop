package widget

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
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
	Base
	state          *SliderState
	min, max, step int
	mid            int // value at the middle of the track; 0 for a linear track
	value          int
	onChange       func(int)
	onEnd          func(int)
}

// normalize makes a range that cannot be laid out usable: a step of at
// least 1, a max above the min, and a mid strictly inside the range (else
// the track is linear). Without it the thumb position divides by zero.
func (s *sliderB) normalize() {
	s.step = max(s.step, 1)
	s.max = max(s.max, s.min+1)
	if s.mid <= s.min || s.mid >= s.max {
		s.mid = 0
	}
}

func (s *sliderB) track(element *El) (x, w int) {
	return element.Rect().X + sliderPad, max(element.Rect().W-2*sliderPad, 1)
}

// valueAt maps a track fraction (0..1) to a stepped value. With a mid
// value the two halves of the track are scaled separately, so the mid
// value sits in the middle.
func (s *sliderB) valueAt(fraction float64) int {
	fraction = max(0, min(fraction, 1))
	lo, hi, g := float64(s.min), float64(s.max), fraction
	if s.mid != 0 {
		if fraction < 0.5 {
			hi, g = float64(s.mid), fraction*2
		} else {
			lo, g = float64(s.mid), (fraction-0.5)*2
		}
	}
	v := lo + g*(hi-lo)
	steps := int((v-float64(s.min))/float64(s.step) + 0.5)
	return max(s.min, min(s.min+steps*s.step, s.max))
}

// fracOf is the track fraction of value v.
func (s *sliderB) fracOf(value int) float64 {
	value = max(s.min, min(value, s.max))
	if s.mid == 0 {
		return float64(value-s.min) / float64(s.max-s.min)
	}
	if value < s.mid {
		return 0.5 * float64(value-s.min) / float64(s.mid-s.min)
	}
	return 0.5 + 0.5*float64(value-s.mid)/float64(s.max-s.mid)
}

func (s *sliderB) Measure(*El, *core.Ctx, int) (int, int) { return 120, 20 }

func (s *sliderB) Update(element *El, ctx *core.Ctx) {
	in, st := ctx.In, s.state
	switch {
	case st.Dragging:
		if in.Held {
			f := st.startF + (float64(ctx.PointerX)*ctx.Scale-st.startDev)/st.trackDev
			if v := s.valueAt(f); v != st.pending {
				st.pending = v
				if s.onChange != nil {
					ctx.Fired++
					s.onChange(v)
				}
			}
			return
		}
		st.Dragging = false
		if s.onEnd != nil {
			ctx.Fired++
			s.onEnd(st.pending)
		}
	case in.Click && in.Over(element.Rect()):
		x, w := s.track(element)
		st.Dragging = true
		st.startF = max(0, min(float64(in.X-x)/float64(w), 1))
		st.startDev, st.trackDev = float64(in.X)*ctx.Scale, float64(w)*ctx.Scale
		st.pending = s.valueAt(st.startF)
		if st.pending != s.value && s.onChange != nil {
			ctx.Fired++
			s.onChange(st.pending)
		}
		ctx.In.Click = false
	}
}

func (s *sliderB) Draw(element *El, ctx *core.Ctx, dst *ebiten.Image) {
	r := element.Rect()
	x, w := s.track(element)
	frac := s.fracOf(s.state.Shown(s.value))
	mid := r.Y + r.H/2
	pill := float64(2) * min(ctx.Th.RadiusScale, 1) // the track is square when corners are
	chip, accent := ctx.Color(core.Chip), ctx.Color(core.Accent)
	ctx.FillStyled(dst, core.Rect{X: x, Y: mid - 2, W: w, H: 4}, chip, chip, pill)
	if s.mid != 0 { // a tick at the middle value
		ctx.Fill(dst, core.Rect{X: x + w/2, Y: mid - 5, W: 1, H: 10}, ctx.Color(core.Rule))
	}
	ctx.FillStyled(dst, core.Rect{X: x, Y: mid - 2, W: int(frac*float64(w) + 0.5), H: 4}, accent, accent, pill)
	tx := x + int(frac*float64(w)+0.5)
	thumb := core.Rect{X: tx - 5, Y: mid - 8, W: 10, H: 16}
	clr := core.Accent
	if s.state.Dragging || ctx.Hovered(r) {
		clr = core.AccentLine
	}
	tc := ctx.Color(clr)
	ctx.FillStyled(dst, thumb, tc, tc, ctx.Radius("slider"))
	ctx.OutlineStyled(dst, thumb, ctx.Color(core.Border), ctx.Radius("slider"))
}

// Slider picks an integer from min to max in steps, by dragging. OnSlide
// runs with each new value while the thumb moves, so the effect can be
// seen live. In a template it is <slider state min max step value>, with
// the state and callbacks named from the scope.
func Slider(state *SliderState, low, high, step, value int, opts ...Option) *El {
	slider := &sliderB{state: state, min: low, max: high, step: step, value: value}
	slider.normalize()
	return Custom(slider, With([]Item{Grow(1), H(core.CompactH)}, opts)...)
}

// OnSlide runs with each new value of a Slider while the thumb moves.
func OnSlide(callback func(int)) Option {
	return func(element *El) {
		slider := behaviorOf[*sliderB](element, "OnSlide")
		slider.onChange = callback
	}
}

// SliderMid puts value mid in the middle of the slider's track, scaling
// the two halves separately.
func SliderMid(mid int) Option {
	return func(element *El) {
		slider := behaviorOf[*sliderB](element, "SliderMid")
		slider.mid = mid
		slider.normalize()
	}
}

// OnSlideEnd runs with the final value when the thumb is released.
func OnSlideEnd(callback func(int)) Option {
	return func(element *El) {
		slider := behaviorOf[*sliderB](element, "OnSlideEnd")
		slider.onEnd = callback
	}
}
