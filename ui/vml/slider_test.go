package vml

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/ui/widget"
)

func sliderRoot(st *SliderState, value int, got *[]int) *El {
	return Col(Slider(st, 50, 200, 5, value, func(v int) { *got = append(*got, v) }, W(200+2*sliderPad), H(20)))
}

func press(c *widget.Ctx, root *El, x int, held bool, click bool) {
	c.In = widget.Input{X: x, Y: 10, Click: click, Held: held}
	Update(c, root)
}

func TestSliderMapsPositionToSteppedValueAndCommitsOnRelease(t *testing.T) {
	var got []int
	st := &SliderState{}
	root := sliderRoot(st, 100, &got)
	c := testCtx(t)
	Layout(c, root, 400, 100)
	// The track runs from sliderPad to sliderPad+200: 50..200 over 200px.
	press(c, root, sliderPad+100, true, true) // halfway: 125
	if len(got) != 0 {
		t.Fatalf("committed while pressing: %v", got)
	}
	if st.Shown(100) != 125 {
		t.Fatalf("pending %d, want 125", st.Shown(100))
	}
	press(c, root, sliderPad+101, true, false) // drag: still 125 after rounding to the step
	press(c, root, sliderPad+400, true, false) // far past the end: clamps to 200
	if st.Shown(100) != 200 {
		t.Fatalf("pending %d, want 200", st.Shown(100))
	}
	press(c, root, sliderPad+400, false, false) // release
	if len(got) != 1 || got[0] != 200 {
		t.Fatalf("committed %v, want [200]", got)
	}
	if st.Shown(120) != 120 {
		t.Fatal("after the drag the slider shows the stored value again")
	}
}

func TestSliderClampsToTheLowEndAndSnapsToTheStep(t *testing.T) {
	var got []int
	st := &SliderState{}
	root := sliderRoot(st, 100, &got)
	c := testCtx(t)
	Layout(c, root, 400, 100)
	press(c, root, 0, true, true)
	press(c, root, 0, false, false)
	if len(got) != 1 || got[0] != 50 {
		t.Fatalf("got %v", got)
	}
	got = nil
	press(c, root, sliderPad+23, true, true) // 50+23*0.75=67.25 -> step 5 -> 65
	press(c, root, sliderPad+23, false, false)
	if len(got) != 1 || got[0] != 65 {
		t.Fatalf("got %v", got)
	}
}

func TestSliderIgnoresClicksOutsideAndDoesNotCommitUnchangedValues(t *testing.T) {
	var got []int
	st := &SliderState{}
	root := sliderRoot(st, 125, &got)
	c := testCtx(t)
	Layout(c, root, 400, 100)
	c.In = widget.Input{X: 390, Y: 10, Click: true, Held: true} // outside the 216px slider
	Update(c, root)
	c.In = widget.Input{X: 390, Y: 10}
	Update(c, root)
	if len(got) != 0 {
		t.Fatalf("got %v", got)
	}
	press(c, root, sliderPad+100, true, true) // 125 again
	press(c, root, sliderPad+100, false, false)
	if len(got) != 0 {
		t.Fatalf("an unchanged value must not be committed: %v", got)
	}
}
