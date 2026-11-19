package settings

import (
	"math"
	"testing"
	"time"
)

func TestTimeoutFallsBackToTheSettingsDefault(t *testing.T) {
	tests := []struct {
		name         string
		req, setting int
		want         time.Duration
	}{
		{"the request's own value wins", 5, 60, 5 * time.Second},
		{"zero uses the default", 0, 30, 30 * time.Second},
		{"no default either uses a minute", 0, 0, 60 * time.Second},
		{"a negative value is not a timeout", -3, 30, 30 * time.Second},
	}
	for _, test := range tests {
		got := Settings{TimeoutSec: test.setting}.TimeoutFor(test.req)
		if got != test.want {
			t.Errorf("%s: got %v, want %v", test.name, got, test.want)
		}
	}
}

// A huge timeout (from a request or the settings file) overflowed
// time.Duration to a negative value, so the request failed at once.
func TestTimeoutIsCappedInsteadOfOverflowing(t *testing.T) {
	for _, sec := range []int{1 << 40, math.MaxInt} {
		for _, prefs := range []Settings{{}, {TimeoutSec: sec}} {
			got := prefs.TimeoutFor(sec)
			if got <= 0 || got > MaxTimeout {
				t.Errorf("TimeoutFor(%d s) = %v, want 1..%v", sec, got, MaxTimeout)
			}
		}
	}
}
