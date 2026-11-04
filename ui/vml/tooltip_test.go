package vml

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/ui/widget"
)

func TestTooltipAppearsAfterTheHoverDelayAndOnlyThenIsShown(t *testing.T) {
	a := box(W(100), Tooltip("Copy the body"))
	b := box(W(100), Tooltip("Other"))
	root := Row(a, b)
	c := testCtx(t)
	c.In = widget.Input{X: 50, Y: 10}
	c.Tick = 100
	drawn(t, c, root, 200, 50)
	if c.TipShown || !c.TipWaiting || c.TipText != "Copy the body" {
		t.Fatalf("first frame: shown %v waiting %v text %q", c.TipShown, c.TipWaiting, c.TipText)
	}
	c.Tick = 100 + TipDelayTicks - 1
	drawn(t, c, root, 200, 50)
	if c.TipShown {
		t.Fatal("shown before the delay")
	}
	c.Tick = 100 + TipDelayTicks
	drawn(t, c, root, 200, 50)
	if !c.TipShown || c.TipWaiting {
		t.Fatalf("after the delay: shown %v waiting %v", c.TipShown, c.TipWaiting)
	}
	// Moving to another element starts the wait again.
	c.In = widget.Input{X: 150, Y: 10}
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
	c.In = widget.Input{X: 50, Y: 10}
	c.Tick = 10
	drawn(t, c, root, 200, 50)
	c.Tick = 10 + TipDelayTicks
	drawn(t, c, root, 200, 50)
	if !c.TipShown {
		t.Fatal("setup: the tip must be shown")
	}
	c.In = widget.Input{X: 50, Y: 10, Click: true, Held: true}
	c.Tick++
	drawn(t, c, root, 200, 50)
	if c.TipShown {
		t.Fatal("a click hides the tooltip")
	}
	c.In = widget.Input{X: 150, Y: 10}
	c.Tick++
	drawn(t, c, root, 200, 50)
	if c.TipText != "" || c.TipShown || c.TipWaiting {
		t.Fatalf("off the element: %q %v %v", c.TipText, c.TipShown, c.TipWaiting)
	}
}

func TestNoTooltipForElementsUnderAModalOverlay(t *testing.T) {
	under := box(W(100), Tooltip("under"))
	overlay := box(W(10), H(10), FloatAt(150, 0), Modal(func() {}))
	root := Stack(Row(under), overlay)
	c := testCtx(t)
	c.In = widget.Input{X: 50, Y: 30}
	drawn(t, c, root, 200, 50)
	if c.TipText != "" {
		t.Fatalf("tooltip %q under a modal", c.TipText)
	}
}

func TestEveryLineIconHasASizeAHoverColorAndDrawsWithoutHelp(t *testing.T) {
	makers := map[string]func(...Option) *El{
		"trash": Trash, "copy": CopyIcon, "search": SearchIcon, "plus": Plus, "prompt": PromptIcon,
		"up": ChevronUp, "down": ChevronDown,
	}
	for name, mk := range makers {
		e := mk(Grow(1))
		if _, ok := iconSizes[e.icon]; !ok {
			t.Errorf("%s has no size", name)
		}
		if glyphs[e.icon] == nil {
			t.Errorf("%s has no drawing", name)
		}
		if _, ok := glyphHover[e.icon]; !ok {
			t.Errorf("%s has no hover color", name)
		}
		c := testCtx(t)
		root := Row(W(30), H(30), e)
		c.In = widget.Input{X: 10, Y: 10} // hovered, so the hover color path runs too
		drawn(t, c, root, 30, 30)
	}
}

func TestIconButtonIsAnOutlinedChipAroundTheGlyph(t *testing.T) {
	g := Trash(Grow(1))
	b := IconButton(g, W(40), H(28))
	if b.bg != widget.Chip || b.border != widget.Border {
		t.Fatalf("bg %q border %q", b.bg, b.border)
	}
	Layout(testCtx(t), Col(b), 100, 100)
	if g.rect.W != 40 {
		t.Fatalf("the glyph must fill the button so hover and clicks land on it, W=%d", g.rect.W)
	}
}
