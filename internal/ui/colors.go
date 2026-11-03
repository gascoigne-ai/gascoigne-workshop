package ui

import "image/color"

// lighten nudges a color toward white for hover states.
func lighten(c color.RGBA) color.RGBA {
	const step = 16
	up := func(v uint8) uint8 { return uint8(min(int(v)+step, 255)) }
	return color.RGBA{R: up(c.R), G: up(c.G), B: up(c.B), A: c.A}
}
