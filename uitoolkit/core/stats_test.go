package core

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestRoundedFillIsOneMaskDraw(t *testing.T) {
	ctx, err := NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	ctx.Stats = &DrawStats{}
	img := ebiten.NewImage(20, 20)
	ctx.FillStyled(img, Rect{X: 2, Y: 2, W: 16, H: 16}, color.RGBA{255, 0, 0, 255}, color.RGBA{0, 0, 255, 255}, 4)
	if ctx.Stats.MaskFills != 1 {
		t.Fatalf("MaskFills = %d, want 1", ctx.Stats.MaskFills)
	}
}
