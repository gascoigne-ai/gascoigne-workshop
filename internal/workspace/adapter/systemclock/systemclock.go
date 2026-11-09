// Package systemclock implements port.Clock with the wall clock.
package systemclock

import "time"

// Clock is the wall clock.
type Clock struct{}

// Now is the current local time.
func (Clock) Now() time.Time { return time.Now() }
