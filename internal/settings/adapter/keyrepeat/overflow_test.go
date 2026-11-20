package keyrepeat

import (
	"testing"
	"time"
)

// A desktop that reports a count beyond maxSetting (a bad or hostile
// output) overflowed time.Duration into a negative delay, which the UI
// would take as a real timing. The output is now unknown.
func TestACountBeyondTheCapIsUnknownNotANegativeDuration(t *testing.T) {
	huge := "9223372036854775807"
	cases := map[string]func() (time.Duration, time.Duration){
		"hyprland":  func() (time.Duration, time.Duration) { return parseHyprland(`{"int": `+huge+`}`, `{"int": 1}`) },
		"gsettings": func() (time.Duration, time.Duration) { return parseGSettings("uint32 "+huge, "uint32 1") },
		"xset": func() (time.Duration, time.Duration) {
			return parseXset("auto repeat delay: " + huge + " repeat rate: 25")
		},
		"mac": func() (time.Duration, time.Duration) { return parseMac(huge, "1") },
	}
	for name, parse := range cases {
		if delay, interval := parse(); delay != 0 || interval != 0 {
			t.Errorf("%s: got %v and %v, want unknown", name, delay, interval)
		}
	}
}

// A repeat rate of 0 gave a delay without an interval.
func TestAnXsetRateOfZeroIsUnknown(t *testing.T) {
	if delay, interval := parseXset("auto repeat delay: 600 repeat rate: 0"); delay != 0 || interval != 0 {
		t.Errorf("got %v and %v, want unknown", delay, interval)
	}
}
