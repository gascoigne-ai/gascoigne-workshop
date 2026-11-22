package keyrepeat

import (
	"os"
	"testing"
	"time"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestEachDesktopsOutputBecomesADelayAndAnInterval(t *testing.T) {
	ms := time.Millisecond
	hyprDelay, hyprInterval := parseHyprland(fixture(t, "hyprctl_repeat_delay.json"), fixture(t, "hyprctl_repeat_rate.json"))
	gnomeDelay, gnomeInterval := parseGSettings(fixture(t, "gsettings_delay.txt"), fixture(t, "gsettings_interval.txt"))
	xsetDelay, xsetInterval := parseXset("  auto repeat:    yes    key click percent:  0    LED mask:  00000000\n  auto repeat delay:  600    repeat rate:  25\n")
	macDelay, macInterval := parseMac("25\n", "2\n")
	winDelay, winInterval := parseWindows("    KeyboardDelay    REG_SZ    1\n", "    KeyboardSpeed    REG_SZ    31\n")
	cases := []struct {
		name                  string
		gotDelay, gotInterval time.Duration
		wantDelay, wantInterv time.Duration
	}{
		{"hyprland 600 ms and 40 a second", hyprDelay, hyprInterval, 600 * ms, 25 * ms},
		{"gnome 500 ms and 30 ms", gnomeDelay, gnomeInterval, 500 * ms, 30 * ms},
		{"x11 600 ms and 25 a second", xsetDelay, xsetInterval, 600 * ms, 40 * ms},
		{"macos 25 and 2 ticks", macDelay, macInterval, 375 * ms, 30 * ms},
		{"windows step 1 and fastest", winDelay, winInterval, 500 * ms, 33333333},
	}
	for _, tt := range cases {
		if tt.gotDelay != tt.wantDelay || tt.gotInterval != tt.wantInterv {
			t.Errorf("%s: got %v and %v, want %v and %v", tt.name, tt.gotDelay, tt.gotInterval, tt.wantDelay, tt.wantInterv)
		}
	}
}

func TestOutputThatIsNotASettingIsUnknown(t *testing.T) {
	zeroDelay, zeroInterval := time.Duration(0), time.Duration(0)
	for name, parse := range map[string]func() (time.Duration, time.Duration){
		"hyprland":  func() (time.Duration, time.Duration) { return parseHyprland("error", `{"int": 0}`) },
		"gsettings": func() (time.Duration, time.Duration) { return parseGSettings("", "No such key") },
		"xset":      func() (time.Duration, time.Duration) { return parseXset("no repeat here") },
		"mac":       func() (time.Duration, time.Duration) { return parseMac("The domain does not exist", "2") },
		"windows":   func() (time.Duration, time.Duration) { return parseWindows("ERROR", "REG_SZ 9") },
	} {
		if delay, interval := parse(); delay != zeroDelay || interval != zeroInterval {
			t.Errorf("%s: got %v and %v, want unknown", name, delay, interval)
		}
	}
}

func TestTimingTakesTheFirstSourceThatKnowsBoth(t *testing.T) {
	unknown := func() (time.Duration, time.Duration) { return 0, 0 }
	half := func() (time.Duration, time.Duration) { return time.Second, 0 }
	known := func() (time.Duration, time.Duration) { return 300 * time.Millisecond, 50 * time.Millisecond }
	system := System{sources: []source{unknown, half, known, func() (time.Duration, time.Duration) { return time.Hour, time.Hour }}}
	if delay, interval := system.Timing(); delay != 300*time.Millisecond || interval != 50*time.Millisecond {
		t.Fatalf("Timing = %v and %v", delay, interval)
	}
	if delay, interval := (System{}).Timing(); delay != 0 || interval != 0 {
		t.Fatalf("no sources: %v and %v, want unknown", delay, interval)
	}
}
