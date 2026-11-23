// Package appicon draws the program's icon: a rounded burgundy square with
// a light spanner, at any size. The picture is appicon.svg, rasterized in
// software when asked, because the window icon is needed before any GPU
// image exists; there is no image file to keep in step with it.
package appicon

import (
	_ "embed"
	"image"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

//go:embed appicon.svg
var source []byte

// Image returns the icon at size x size pixels.
func Image(size int) *image.RGBA {
	img, err := widget.RasterizeSVG("appicon.svg", source, size)
	if err != nil {
		panic(err) // the file is part of the program, so this is a bug in it
	}
	return img
}
