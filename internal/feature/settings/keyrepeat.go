package settings

import "time"

// KeyRepeat tells how the desktop repeats a held key.
type KeyRepeat interface {
	// Timing is the wait before the first repeat and the time between
	// repeats; both are 0 when the desktop's setting is unknown.
	Timing() (delay, interval time.Duration)
}
