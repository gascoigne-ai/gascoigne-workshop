package settings

import "time"

const (
	// DefaultTimeoutSec is used when neither the request nor the settings name one.
	DefaultTimeoutSec = 60

	// MaxTimeout caps a timeout: longer would overflow time.Duration for a
	// huge setting.
	MaxTimeout = 24 * time.Hour
)

// TimeoutFor is how long a request may run: requestSec when it sets one,
// else the default from the settings, else a minute. It is never longer
// than MaxTimeout.
func (prefs Settings) TimeoutFor(requestSec int) time.Duration {
	sec := requestSec
	if sec <= 0 {
		sec = prefs.TimeoutSec
	}
	if sec <= 0 {
		sec = DefaultTimeoutSec
	}
	return time.Duration(min(sec, int(MaxTimeout/time.Second))) * time.Second // before the multiply, which overflows
}
