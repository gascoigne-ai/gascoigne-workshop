package view

import (
	"image"
	"math"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// drawOverflow is how far, in logical pixels, an element may draw outside
// its rect besides its shadow: a focus ring or a mark drawn over the edge.
const drawOverflow = 4

// drawReach is how far in device pixels a child's drawing may go past its
// rect: the shadow offset, the overflow and one pixel of anti-aliasing.
func drawReach(scale float64) int {
	return int(math.Ceil((core.ShadowOffset+drawOverflow)*scale)) + 1
}

// reaches reports whether drawing rect, scaled to device pixels and grown by
// reach on every side, can touch bounds. The edges round outwards, so a
// fractional scale never hides a pixel.
func reaches(bounds image.Rectangle, rect core.Rect, scale float64, reach int) bool {
	left := int(math.Floor(float64(rect.X)*scale)) - reach
	top := int(math.Floor(float64(rect.Y)*scale)) - reach
	right := int(math.Ceil(float64(rect.X+rect.W)*scale)) + reach
	bottom := int(math.Ceil(float64(rect.Y+rect.H)*scale)) + reach
	// Compared by hand: Rectangle.Overlaps is false for an empty rectangle,
	// and a zero-size element inside the clip may still have children.
	return left < bounds.Max.X && right > bounds.Min.X && top < bounds.Max.Y && bottom > bounds.Min.Y
}
