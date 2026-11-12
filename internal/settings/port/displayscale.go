package port

// DisplayScale tells the desktop's scale for the display when the window
// system does not pass it on.
type DisplayScale interface {
	// Factor is the scale (2 for a 2x display), 0 when unknown.
	Factor() float64
}
