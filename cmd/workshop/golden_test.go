//go:build !release

package main

import (
	"image"
	"testing"
)

const (
	shotWidth     = 1280
	shotHeight    = 800
	minShotColors = 8 // a drawn window has a background, text, borders and accents
)

// The golden scenes are only comparable if the program really renders
// without a GPU: the default window, drawn by Mesa's software rasterizer on
// a virtual display, is a full-size image with content in it.
func TestShotUnderSoftwareRenderingIsAFullWindowImage(t *testing.T) {
	img := renderShot(t)
	if got := img.Bounds().Size(); got != (image.Point{X: shotWidth, Y: shotHeight}) {
		t.Errorf("shot size = %v, want %dx%d", got, shotWidth, shotHeight)
	}
	if colors := distinctColors(img); colors < minShotColors {
		t.Errorf("shot has %d distinct colors, want at least %d: the window was not drawn", colors, minShotColors)
	}
}

func distinctColors(img image.Image) int {
	seen := map[[4]uint32]bool{}
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			seen[[4]uint32{r, g, b, a}] = true
		}
	}
	return len(seen)
}
