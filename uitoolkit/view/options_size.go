package view

// W fixes the width in pixels.
func W(pixels int) Option { return func(element *El) { element.w = pixels } }

// H fixes the height in pixels.
func H(pixels int) Option { return func(element *El) { element.h = pixels } }

// Grow makes the element take a share of the free space in its row or
// column, proportional to weight.
func Grow(weight float64) Option { return func(element *El) { element.grow = weight } }

// Optional lets a row or column drop the element when there is not
// enough room for everything.
func Optional() Option { return func(element *El) { element.optional = true } }

// Shrink lets an element that does not grow give up room along its parent's
// axis when the row or column does not fit: the overflow is taken from
// the Shrink children first to last, each down to its MinW (MinH in a
// column). A row that fits lays out as if no child had it.
//
// Shrink is explicit rather than read from Ellipsis, because the child that
// gives room is often a container (the user's choice over a lower cap on
// chip width).
func Shrink() Option { return func(element *El) { element.shrink = true } }

// FitW turns the W given before it into a minimum: the element is as wide
// as its content when that is more, and goes back to W (shrinking its
// content) when its row is too tight. It has no template attribute: a
// template says min-w and shrink, and buttons apply FitW themselves.
func FitW() Option {
	return func(element *El) {
		if element.w == 0 {
			return
		}
		element.minW, element.w, element.shrink = element.w, 0, true
	}
}

// Pad sets the same padding on all sides.
func Pad(pixels int) Option {
	return func(element *El) {
		element.padL, element.padT, element.padR, element.padB = pixels, pixels, pixels, pixels
	}
}

// PadX sets the left and right padding.
func PadX(pixels int) Option {
	return func(element *El) { element.padL, element.padR = pixels, pixels }
}

// PadY sets the top and bottom padding.
func PadY(pixels int) Option {
	return func(element *El) { element.padT, element.padB = pixels, pixels }
}

// PadL sets the left padding.
func PadL(pixels int) Option { return func(element *El) { element.padL = pixels } }

// PadT sets the top padding.
func PadT(pixels int) Option { return func(element *El) { element.padT = pixels } }

// PadR sets the right padding.
func PadR(pixels int) Option { return func(element *El) { element.padR = pixels } }

// PadB sets the bottom padding.
func PadB(pixels int) Option { return func(element *El) { element.padB = pixels } }

// Gap sets the space between children.
func Gap(pixels int) Option { return func(element *El) { element.gap = pixels } }
