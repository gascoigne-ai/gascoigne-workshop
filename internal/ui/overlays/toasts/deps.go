package toasts

import "time"

// Deps is what the component asks of the application.
type Deps struct {
	Clock  Clock
	Window Window
}

// Clock tells the time of the frame.
type Clock interface {
	Now() time.Time
}

// Window is the window the toasts are drawn over.
type Window interface {
	// Size is the window's size in logical pixels.
	Size() (width, height int)
	// Stale says the view built this frame no longer shows what changed.
	Stale()
	// Redraw says the screen must be drawn again.
	Redraw()
}
