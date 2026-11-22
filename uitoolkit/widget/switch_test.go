package widget

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

func TestSwitchOnChangeRunsWhenTheTrackIsClicked(t *testing.T) {
	var changes int
	toggle := Switch(false, OnChange(func() { changes++ }))
	root := Row(AlignItems(AlignStart), toggle)
	c := testCtx(t)
	Layout(c, root, 200, 40)
	rect := toggle.Rect()
	c.In = core.Input{X: rect.X + rect.W + 20, Y: rect.Y + 2, Click: true} // beside the track
	Update(c, root)
	c.In = core.Input{X: rect.X + rect.W/2, Y: rect.Y + rect.H/2, Click: true}
	Update(c, root)
	if changes != 1 {
		t.Fatalf("OnChange ran %d times, want 1", changes)
	}
	if c.Fired != 1 {
		t.Fatalf("Fired %d, want 1", c.Fired)
	}
}

func TestADisabledSwitchIgnoresAClick(t *testing.T) {
	var changes int
	toggle := DisabledSwitch(true, OnChange(func() { changes++ }))
	root := Row(AlignItems(AlignStart), toggle)
	c := testCtx(t)
	Layout(c, root, 200, 40)
	rect := toggle.Rect()
	c.In = core.Input{X: rect.X + rect.W/2, Y: rect.Y + rect.H/2, Click: true}
	Update(c, root)
	if changes != 0 {
		t.Fatalf("a disabled switch ran OnChange %d times", changes)
	}
}

func TestSwitchSizeFollowsTheSizeRules(t *testing.T) {
	toggle := Switch(true)
	c := testCtx(t)
	Layout(c, Row(AlignItems(AlignStart), toggle), 200, 40)
	rect := toggle.Rect()
	if rect.W != c.Sizes.IconBtnW || rect.H != c.Sizes.Checkbox+2*core.StepHair {
		t.Fatalf("size %dx%d, want %dx%d", rect.W, rect.H, c.Sizes.IconBtnW, c.Sizes.Checkbox+2*core.StepHair)
	}
}

func TestSwitchDrawsOnAndOff(t *testing.T) {
	c := testCtx(t)
	drawn(t, c, Row(Switch(true), Switch(false), DisabledSwitch(true)), 200, 40)
}
