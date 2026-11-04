// Package appicon draws the program's icon: a rounded burgundy square with
// a light mark, at any size. It is drawn, not stored, so there is no image
// file to keep in step with the theme.
package appicon

import (
	"image"
	"image/color"
	"math"
)

// Image returns the icon at size x size pixels.
func Image(size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	s := float64(size)
	radius := s * 0.22
	top, bottom := color.RGBA{0x6a, 0x1b, 0x36, 0xff}, color.RGBA{0x2a, 0x0d, 0x18, 0xff}
	light := color.RGBA{0xf4, 0xd9, 0xe2, 0xff}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			cov := roundedCoverage(float64(x)+0.5, float64(y)+0.5, s, radius)
			if cov == 0 {
				continue
			}
			t := float64(y) / s
			c := mix(top, bottom, t)
			if m := markCoverage(float64(x)+0.5, float64(y)+0.5, s); m > 0 {
				c = mix(c, light, m)
			}
			c.A = uint8(math.Round(float64(c.A) * cov))
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	f := func(x, y uint8) uint8 { return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t)) }
	return color.RGBA{f(a.R, b.R), f(a.G, b.G), f(a.B, b.B), 0xff}
}

// roundedCoverage is how much of the pixel at x, y lies inside a square of
// side s with the given corner radius (0 outside, 1 inside, soft at the edge).
func roundedCoverage(x, y, s, r float64) float64 {
	dx := math.Max(math.Abs(x-s/2)-(s/2-r), 0)
	dy := math.Max(math.Abs(y-s/2)-(s/2-r), 0)
	d := math.Hypot(dx, dy) - r
	return clamp(0.5 - d)
}

// markCoverage draws a ">_" prompt, the same glyph as the toolbar's, as
// two strokes and a cursor, in the middle of the square.
func markCoverage(x, y, s float64) float64 {
	w := s * 0.075 // stroke half width
	u := func(v float64) float64 { return v * s }
	cov := 0.0
	cov = math.Max(cov, segment(x, y, u(0.25), u(0.34), u(0.45), u(0.5), w))
	cov = math.Max(cov, segment(x, y, u(0.45), u(0.5), u(0.25), u(0.66), w))
	cov = math.Max(cov, segment(x, y, u(0.52), u(0.68), u(0.76), u(0.68), w))
	return cov
}

// segment is the coverage of a round-capped line from a to b, half width w.
func segment(px, py, ax, ay, bx, by, w float64) float64 {
	vx, vy := bx-ax, by-ay
	t := clamp(((px-ax)*vx + (py-ay)*vy) / (vx*vx + vy*vy))
	d := math.Hypot(px-(ax+t*vx), py-(ay+t*vy)) - w
	return clamp(0.5 - d)
}

func clamp(v float64) float64 { return math.Min(math.Max(v, 0), 1) }
