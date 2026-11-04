package vml

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Image shows a picture scaled down to fit its box, centered, keeping its
// proportions. It is never scaled up past its own size. Give it Grow or
// W and H to decide the box; without them it asks for the picture's size.
func Image(img *ebiten.Image, opts ...Option) *El {
	e := newEl(imageB{img: img}, with(nil, opts))
	return e
}

type imageB struct {
	plain
	img *ebiten.Image
}

func (b imageB) measure(*El, *core.Ctx, int) (int, int) {
	if b.img == nil {
		return 0, 0
	}
	r := b.img.Bounds()
	return r.Dx(), r.Dy()
}

func (b imageB) draw(e *El, c *core.Ctx, dst *ebiten.Image) {
	if b.img == nil {
		return
	}
	r := e.rect
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
