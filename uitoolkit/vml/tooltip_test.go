package vml

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestTooltipAppearsAfterTheHoverDelayAndOnlyThenIsShown(t *testing.T) {
	a := box(W(100), Tooltip("Copy the body"))
	b := box(W(100), Tooltip("Other"))
	root := Row(a, b)
	c := testCtx(t)
	c.In = core.Input{X: 50, Y: 10}
	c.Tick = 100
	drawn(t, c, root, 200, 50)
	if c.TipShown || !c.TipWaiting || c.TipText != "Copy the body" {
		t.Fatalf("first frame: shown %v waiting %v text %q", c.TipShown, c.TipWaiting, c.TipText)
	}
	c.Tick = 100 + tipDelayTicks - 1
	drawn(t, c, root, 200, 50)
	if c.TipShown {
		t.Fatal("shown before the delay")
	}
	c.Tick = 100 + tipDelayTicks
	drawn(t, c, root, 200, 50)
	if !c.TipShown || c.TipWaiting {
		t.Fatalf("after the delay: shown %v waiting %v", c.TipShown, c.TipWaiting)
	}
	// Moving to another element starts the wait again.
	c.In = core.Input{X: 150, Y: 10}
	c.Tick++
	drawn(t, c, root, 200, 50)
	if c.TipShown || c.TipText != "Other" {
		t.Fatalf("on the other element: shown %v text %q", c.TipShown, c.TipText)
	}
}

func TestTooltipGoesAwayOffTheElementAndOnClick(t *testing.T) {
	a := box(W(100), Tooltip("tip"))
	root := Row(a, box(W(100)))
	c := testCtx(t)
	c.In = core.Input{X: 50, Y: 10}
	c.Tick = 10
	drawn(t, c, root, 200, 50)
	c.Tick = 10 + tipDelayTicks
	drawn(t, c, root, 200, 50)
	if !c.TipShown {
		t.Fatal("setup: the tip must be shown")
	}
	c.In = core.Input{X: 50, Y: 10, Click: true, Held: true}
	c.Tick++
	drawn(t, c, root, 200, 50)
	if c.TipShown {
		t.Fatal("a click hides the tooltip")
	}
	c.In = core.Input{X: 150, Y: 10}
	c.Tick++
	drawn(t, c, root, 200, 50)
	if c.TipText != "" || c.TipShown || c.TipWaiting {
		t.Fatalf("off the element: %q %v %v", c.TipText, c.TipShown, c.TipWaiting)
	}
}

func TestNoTooltipForElementsUnderAModalOverlay(t *testing.T) {
	under := box(W(100), Tooltip("under"))
	overlay := box(W(10), H(10), FloatAt(150, 0), Modal(), OnDismiss(func() {}))
	root := Stack(Row(under), overlay)
	c := testCtx(t)
	c.In = core.Input{X: 50, Y: 30}
	drawn(t, c, root, 200, 50)
	if c.TipText != "" {
		t.Fatalf("tooltip %q under a modal", c.TipText)
	}
}

func TestIconButtonIsAnOutlinedChipAroundTheGlyph(t *testing.T) {
	g := TrashIcon(Grow(1))
	b := IconButton(g, W(40), H(28))
	if b.bg != core.Sunken || b.control != "icon-button" {
		t.Fatalf("bg %q control %q", b.bg, b.control)
	}
	Layout(testCtx(t), Col(b), 100, 100)
	if g.rect.W != 40 {
		t.Fatalf("the glyph must fill the button so hover and clicks land on it, W=%d", g.rect.W)
	}
}
