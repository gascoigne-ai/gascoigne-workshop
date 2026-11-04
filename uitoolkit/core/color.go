package core

import "image/color"

// Shade mixes c toward white (amount > 0) or black (amount < 0); the
// alpha is kept.
func Shade(c color.RGBA, amount float64) color.RGBA {
	target, k := 0.0, -amount
	if amount > 0 {
		target, k = 255, amount
	}
	mix := func(v uint8) uint8 { return uint8(float64(v) + (target-float64(v))*k + 0.5) }
	return color.RGBA{R: mix(c.R), G: mix(c.G), B: mix(c.B), A: c.A}
}

// gradientEnds are the top and bottom colors of a vertical gradient
// around base: lighter above, darker below, by strength.
func gradientEnds(base color.RGBA, strength float64) (top, bottom color.RGBA) {
	return Shade(base, strength), Shade(base, -strength)
}
