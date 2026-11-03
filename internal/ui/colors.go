package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// lighten nudges a color toward white for hover states.
func lighten(c color.RGBA) color.RGBA {
	const step = 16
	up := func(v uint8) uint8 { return uint8(min(int(v)+step, 255)) }
	return color.RGBA{R: up(c.R), G: up(c.G), B: up(c.B), A: c.A}
}

func vectorLine(dst *ebiten.Image, x0, y0, x1, y1 float32, clr color.RGBA) {
	vector.StrokeLine(dst, x0, y0, x1, y1, 1.8, clr, true)
}
