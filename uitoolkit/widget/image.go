package widget

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Image shows a picture scaled down to fit its box, centered, keeping its
// proportions; a picture smaller than the box is not scaled up.
func Image(img *ebiten.Image, opts ...Option) *El {
	return Custom(imageB{img: img}, with(nil, opts)...)
}

type imageB struct {
	Base
	img *ebiten.Image
}

func (b imageB) Measure(*El, *core.Ctx, int) (int, int) {
	if b.img == nil {
		return 0, 0
	}
	r := b.img.Bounds()
	return r.Dx(), r.Dy()
}

func (b imageB) Draw(e *El, c *core.Ctx, dst *ebiten.Image) {
	if b.img == nil {
		return
	}
	r := e.Rect()
	iw, ih := float64(b.img.Bounds().Dx()), float64(b.img.Bounds().Dy())
	if iw == 0 || ih == 0 || r.W <= 0 || r.H <= 0 {
		return
	}
	// fit in logical pixels, then to device pixels
	k := math.Min(1, math.Min(float64(r.W)/iw, float64(r.H)/ih))
	ox := float64(r.X) + (float64(r.W)-iw*k)/2
	oy := float64(r.Y) + (float64(r.H)-ih*k)/2
	var op ebiten.DrawImageOptions
	op.Filter = ebiten.FilterLinear
	op.GeoM.Scale(k*c.Scale, k*c.Scale)
	op.GeoM.Translate(ox*c.Scale, oy*c.Scale)
	dst.DrawImage(b.img, &op)
}
