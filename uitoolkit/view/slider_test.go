package view

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// slide is a 50..200 slider with 100 in the middle, on a 200px track.
type slide struct {
	st    *SliderState
	live  []int
	ended []int
	root  *El
}

func newSlide(t *testing.T, value int, opts ...Option) (*slide, *core.Ctx) {
	s := &slide{st: &SliderState{}}
	items := append([]Option{W(200 + 2*sliderPad), H(20), OnSlideEnd(func(v int) { s.ended = append(s.ended, v) })}, opts...)
	s.root = Col(Slider(s.st, 50, 200, 5, value, append([]Option{OnSlide(func(v int) { s.live = append(s.live, v) })}, items...)...))
	c := testCtx(t)
	Layout(c, s.root, 400, 100)
	return s, c
}

func (s *slide) at(c *core.Ctx, x int, held, click bool) {
	c.In = core.Input{X: x, Y: 10, Click: click, Held: held}
	Update(c, s.root)
}

func TestSliderAppliesLiveWhileDraggingAndEndsOnRelease(t *testing.T) {
	s, c := newSlide(t, 100) // linear: 50..200 over 200px
	s.at(c, sliderPad+100, true, true)
	if len(s.live) != 1 || s.live[0] != 125 {
		t.Fatalf("live after the press: %v", s.live)
	}
	s.at(c, sliderPad+400, true, false) // far past the end
	if got := s.live[len(s.live)-1]; got != 200 {
		t.Fatalf("live at the end: %v", s.live)
	}
	if len(s.ended) != 0 {
		t.Fatalf("ended early: %v", s.ended)
	}
	s.at(c, sliderPad+400, false, false)
	if len(s.ended) != 1 || s.ended[0] != 200 {
		t.Fatalf("ended: %v", s.ended)
	}
	if s.st.Shown(120) != 120 {
		t.Fatal("after the drag the slider shows the stored value again")
	}
}

func TestSliderDragFollowsPointerMovementNotTheTrackPosition(t *testing.T) {
	s, c := newSlide(t, 100)
	s.at(c, sliderPad+100, true, true) // 125
	// The UI changes size mid-drag: the track now starts elsewhere and
	// the same pointer position would mean another fraction. Movement is
	// what counts: 20 device px right of the press is +15 on this track.
	s.root.kids[0].rect.X += 37
	c.In = core.Input{X: sliderPad + 120, Y: 10, Held: true}
	Update(c, s.root)
	if got := s.live[len(s.live)-1]; got != 140 {
		t.Fatalf("live %v, want 140", s.live)
	}
}

func TestSliderMiddleValueSitsInTheMiddleOfTheTrack(t *testing.T) {
	s, c := newSlide(t, 100, SliderMid(100))
	// Half the track is 100: pressing at the center selects it.
	s.at(c, sliderPad+100, true, true)
	s.at(c, sliderPad+100, false, false)
	if len(s.ended) != 1 || s.ended[0] != 100 {
		t.Fatalf("center = %v", s.ended)
	}
	s.ended = nil
	s.at(c, sliderPad+0, true, true) // left end
	s.at(c, sliderPad+0, false, false)
	if s.ended[0] != 50 {
		t.Fatalf("left end = %v", s.ended)
	}
	s.ended = nil
	s.at(c, sliderPad+50, true, true) // a quarter: halfway from 50 to 100 = 75
	s.at(c, sliderPad+50, false, false)
	if s.ended[0] != 75 {
		t.Fatalf("quarter = %v", s.ended)
	}
	s.ended = nil
	s.at(c, sliderPad+150, true, true) // three quarters: halfway from 100 to 200 = 150
	s.at(c, sliderPad+150, false, false)
	if s.ended[0] != 150 {
		t.Fatalf("three quarters = %v", s.ended)
	}
}

func TestSliderIgnoresClicksOutside(t *testing.T) {
	s, c := newSlide(t, 125)
	s.at(c, 390, true, true)
	s.at(c, 390, false, false)
	if len(s.live) != 0 || len(s.ended) != 0 {
		t.Fatalf("live %v ended %v", s.live, s.ended)
	}
}

func TestSliderKeepsFollowingAPointerThatLeavesItsClippingContainer(t *testing.T) {
	st := &SliderState{}
	var live []int
	slider := Slider(st, 50, 200, 5, 100, OnSlide(func(v int) { live = append(live, v) }), W(200+2*sliderPad), H(20))
	root := Col(Scroll(&ScrollState{}, H(20), W(216), slider)) // clips: a pointer outside is hidden from the slider
	c := testCtx(t)
	Layout(c, root, 400, 100)
	c.In = core.Input{X: sliderPad + 100, Y: 10, Click: true, Held: true}
	Update(c, root)
	c.In = core.Input{X: 380, Y: 90, Held: true} // far outside the scroller
	Update(c, root)
	if got := live[len(live)-1]; got != 200 {
		t.Fatalf("live %v, want the maximum", live)
	}
}
