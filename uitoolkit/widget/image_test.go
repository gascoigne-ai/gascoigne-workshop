package widget

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

func TestImageMeasuresItsPictureAndDrawsScaledToFit(t *testing.T) {
	c := testCtx(t)
	pic := ebiten.NewImage(80, 40)
	im := Image(pic)
	Layout(c, Row(AlignItems(AlignCenter), im), 200, 100)
	if im.Rect().W != 80 || im.Rect().H != 40 {
		t.Fatalf("natural size %dx%d", im.Rect().W, im.Rect().H)
	}
	for _, scale := range []float64{1, 1.5, 2} {
		if err := c.SetScale(scale); err != nil {
			t.Fatal(err)
		}
		drawn(t, c, Row(W(30), H(30), Image(pic, Grow(1))), 30, 30)     // smaller box: scaled down
		drawn(t, c, Row(W(300), H(300), Image(pic, Grow(1))), 300, 300) // larger box: not scaled up
	}
	drawn(t, c, Row(W(30), H(30), Image(nil)), 30, 30) // no picture, no crash
}
