package core

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestDrawTooltipWaitsTheDelayAndRestartsOnClick(t *testing.T) {
	ctx, err := NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	screen := Rect{W: 200, H: 100}
	dst := ebiten.NewImage(screen.W, screen.H)
	ctx.In = Input{X: 50, Y: 10}
	ctx.Tick = 100
	draw := func() {
		ctx.TipText = "Copy the body" // the views set it every frame
		ctx.DrawTooltip(dst, screen)
	}

	draw()
	if ctx.TipShown || !ctx.TipWaiting {
		t.Fatalf("first frame: shown %v waiting %v", ctx.TipShown, ctx.TipWaiting)
	}
	ctx.Tick = 100 + TipDelayTicks - 1
	draw()
	if ctx.TipShown {
		t.Fatal("shown before the delay")
	}
	ctx.Tick = 100 + TipDelayTicks
	draw()
	if !ctx.TipShown || ctx.TipWaiting {
		t.Fatalf("after the delay: shown %v waiting %v", ctx.TipShown, ctx.TipWaiting)
	}

	ctx.In.Click = true
	ctx.Tick++
	draw()
	if ctx.TipShown || !ctx.TipWaiting {
		t.Fatalf("after a click: shown %v waiting %v", ctx.TipShown, ctx.TipWaiting)
	}

	ctx.In.Click = false
	ctx.TipText = ""
	ctx.DrawTooltip(dst, screen)
	if ctx.TipFor != "" || ctx.TipShown || ctx.TipWaiting {
		t.Fatalf("no text: for %q shown %v waiting %v", ctx.TipFor, ctx.TipShown, ctx.TipWaiting)
	}
}
