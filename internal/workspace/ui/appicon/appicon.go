// Package appicon draws the program's icon: a rounded burgundy square with
// a light spanner, at any size. It is drawn, not stored, so there is no image
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

// markCoverage draws a spanner lying on the diagonal: an open-end jaw at the
// top right, a ring at the bottom left, a round handle between them. Every
// part is a signed distance, so the edge is smooth at any size.
func markCoverage(x, y, s float64) float64 {
	px, py := x/s, y/s
	const k = 0.7071067811865476 // 1/sqrt 2
	handle := distSegment(px, py, 0.30, 0.70, 0.62, 0.38) - 0.058
	// the jaw: a disc with a slot cut along the diagonal
	jx, jy := px-0.665, py-0.335
	jaw := math.Hypot(jx, jy) - 0.185
	along, perp := (jx-jy)*k, (jx+jy)*k // along points to the top right
	slot := math.Max(math.Abs(perp)-0.07, 0.02-along)
	jaw = math.Max(jaw, -slot)
	// the ring: a disc, with the hole cut after the handle joins it
	rx, ry := px-0.285, py-0.715
	ring := math.Hypot(rx, ry) - 0.12
	hole := math.Hypot(rx, ry) - 0.05
	d := math.Max(math.Min(handle, math.Min(jaw, ring)), -hole) * s
	return clamp(0.5 - d)
}

// distSegment is the distance from p to the line from a to b.
func distSegment(px, py, ax, ay, bx, by float64) float64 {
	vx, vy := bx-ax, by-ay
	t := clamp(((px-ax)*vx + (py-ay)*vy) / (vx*vx + vy*vy))
	return math.Hypot(px-(ax+t*vx), py-(ay+t*vy))
}

func clamp(v float64) float64 { return math.Min(math.Max(v, 0), 1) }
