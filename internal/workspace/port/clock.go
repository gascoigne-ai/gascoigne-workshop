package port

import "time"

// Clock tells the current time, so toasts, debounces and the caret blink
// can be driven by a fake one.
type Clock interface {
	Now() time.Time
}
