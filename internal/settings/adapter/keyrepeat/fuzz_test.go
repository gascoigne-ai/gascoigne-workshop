package keyrepeat

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// checkTiming holds what every parser promises: no negative duration (a
// huge count must not overflow into one) and a repeat interval whenever
// there is a delay.
func checkTiming(t *testing.T, delay, interval time.Duration) {
	t.Helper()
	if delay < 0 || interval < 0 {
		t.Fatalf("negative timing: delay %v, interval %v", delay, interval)
	}
	if delay > 0 && interval == 0 {
		t.Fatalf("delay %v without an interval", delay)
	}
}

func seedFile(fuzz *testing.F, name string) string {
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		fuzz.Fatal(err)
	}
	return string(data)
}

// Seeds are the captured hyprctl files in testdata.
func FuzzParseHyprland(fuzz *testing.F) {
	fuzz.Add(seedFile(fuzz, "hyprctl_repeat_delay.json"), seedFile(fuzz, "hyprctl_repeat_rate.json"))
	fuzz.Add(`{"int": 9223372036854775807}`, `{"int": 1}`)
	fuzz.Fuzz(func(t *testing.T, delay, rate string) {
		first, second := parseHyprland(delay, rate)
		checkTiming(t, first, second)
	})
}

// Seeds are the captured gsettings files in testdata.
func FuzzParseGSettings(fuzz *testing.F) {
	fuzz.Add(seedFile(fuzz, "gsettings_delay.txt"), seedFile(fuzz, "gsettings_interval.txt"))
	fuzz.Add("uint32 9223372036854775807", "uint32 1")
	fuzz.Fuzz(func(t *testing.T, delay, interval string) {
		first, second := parseGSettings(delay, interval)
		checkTiming(t, first, second)
	})
}

// The seeds are the output shape the tests quote (synthetic around it).
func FuzzParseXset(fuzz *testing.F) {
	fuzz.Add("auto repeat delay: 600 repeat rate: 25")
	fuzz.Add("auto repeat delay: 9223372036854775807 repeat rate: 0")
	fuzz.Fuzz(func(t *testing.T, text string) {
		first, second := parseXset(text)
		checkTiming(t, first, second)
	})
}

func FuzzParseMac(fuzz *testing.F) {
	fuzz.Add("25\n", "6\n")
	fuzz.Add("9223372036854775807", "1")
	fuzz.Fuzz(func(t *testing.T, delay, interval string) {
		first, second := parseMac(delay, interval)
		checkTiming(t, first, second)
	})
}

func FuzzParseWindows(fuzz *testing.F) {
	fuzz.Add("    KeyboardDelay    REG_SZ    1", "    KeyboardSpeed    REG_SZ    31")
	fuzz.Add("REG_SZ 99999999999999999999", "REG_SZ 0")
	fuzz.Fuzz(func(t *testing.T, delay, speed string) {
		first, second := parseWindows(delay, speed)
		checkTiming(t, first, second)
	})
}
