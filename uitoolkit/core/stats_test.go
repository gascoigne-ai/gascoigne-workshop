package core

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestFillBounds(t *testing.T) {
	limit := image.Rect(0, 0, 100, 80)
	tests := []struct {
		name           string
		x0, y0, x1, y1 float32
		want           image.Rectangle
	}{
		{"whole pixels grow by one", 10, 10, 20, 20, image.Rect(9, 9, 21, 21)},
		{"fractional 125 percent", 12.5, 6.25, 25, 18.75, image.Rect(11, 5, 26, 20)},
		{"touching the limit clips", 0, 0, 100, 80, image.Rect(0, 0, 100, 80)},
		{"larger than the limit", -50, -50, 500, 500, limit},
		{"outside the limit is empty", 200, 10, 220, 20, image.Rectangle{}},
		{"left of the limit is empty", -30, 10, -10, 20, image.Rectangle{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := fillBounds(tc.x0, tc.y0, tc.x1, tc.y1, limit); got != tc.want {
				t.Fatalf("fillBounds = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRoundedFillIsOneAntiAliasedDraw(t *testing.T) {
	ctx, err := NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	ctx.Stats = &DrawStats{}
	img := ebiten.NewImage(20, 20)
	ctx.FillStyled(img, Rect{X: 2, Y: 2, W: 16, H: 16}, color.RGBA{255, 0, 0, 255}, color.RGBA{0, 0, 255, 255}, 4)
	if ctx.Stats.AAFills != 1 {
		t.Fatalf("AAFills = %d, want 1", ctx.Stats.AAFills)
	}
}
