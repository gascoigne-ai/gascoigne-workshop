package widget

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

func TestTooltipAppearsAfterTheHoverDelayAndOnlyThenIsShown(t *testing.T) {
	a := Row(W(100), Tooltip("Copy the body"))
	b := Row(W(100), Tooltip("Other"))
	state := &TooltipState{}
	root := Stack(Row(a, b), TooltipLayer(state))
	c := testCtx(t)
	c.In = core.Input{X: 50, Y: 10}
	c.Tick = 100
	drawn(t, c, root, 200, 50)
	if state.Shown() || !state.Waiting() || c.TipText != "Copy the body" {
		t.Fatalf("first frame: shown %v waiting %v text %q", state.Shown(), state.Waiting(), c.TipText)
	}
	c.Tick = 100 + tipDelayTicks - 1
	drawn(t, c, root, 200, 50)
	if state.Shown() {
		t.Fatal("shown before the delay")
	}
	c.Tick = 100 + tipDelayTicks
	drawn(t, c, root, 200, 50)
	if !state.Shown() || state.Waiting() {
		t.Fatalf("after the delay: shown %v waiting %v", state.Shown(), state.Waiting())
	}
	// Moving to another element starts the wait again.
	c.In = core.Input{X: 150, Y: 10}
	c.Tick++
	drawn(t, c, root, 200, 50)
	if state.Shown() || c.TipText != "Other" {
		t.Fatalf("on the other element: shown %v text %q", state.Shown(), c.TipText)
	}
}

func TestTooltipGoesAwayOffTheElementAndOnClick(t *testing.T) {
	a := Row(W(100), Tooltip("tip"))
	state := &TooltipState{}
	root := Stack(Row(a, Row(W(100))), TooltipLayer(state))
	c := testCtx(t)
	c.In = core.Input{X: 50, Y: 10}
	c.Tick = 10
	drawn(t, c, root, 200, 50)
	c.Tick = 10 + tipDelayTicks
	drawn(t, c, root, 200, 50)
	if !state.Shown() {
		t.Fatal("setup: the tip must be shown")
	}
	c.In = core.Input{X: 50, Y: 10, Click: true, Held: true}
	c.Tick++
	drawn(t, c, root, 200, 50)
	if state.Shown() {
		t.Fatal("a click hides the tooltip")
	}
	c.In = core.Input{X: 150, Y: 10}
	c.Tick++
	drawn(t, c, root, 200, 50)
	if c.TipText != "" || state.Shown() || state.Waiting() {
		t.Fatalf("off the element: %q %v %v", c.TipText, state.Shown(), state.Waiting())
	}
}
