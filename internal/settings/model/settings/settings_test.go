package settings

import "testing"

func TestUIScaleDefaultsToTheDisplayScaleAndIsBounded(t *testing.T) {
	if got := Default().UIScale(); got != 1 {
		t.Fatalf("default %v", got)
	}
	for _, tt := range []struct {
		pct  int
		want float64
	}{{0, 1}, {100, 1}, {150, 1.5}, {5, 0.5}, {9000, 3}} {
		if got := (Settings{UIScalePct: tt.pct}).UIScale(); got != tt.want {
			t.Errorf("%d%% -> %v, want %v", tt.pct, got, tt.want)
		}
	}
}

// The Linux desktop text and controls read small, so Linux has a base
// factor under the user's scale; the setting itself stays at 100%.
func TestBaseScaleIsTenPercentOnLinuxOnly(t *testing.T) {
	for goos, want := range map[string]float64{"linux": 1.1, "windows": 1, "darwin": 1, "": 1} {
		if got := BaseScale(goos); got != want {
			t.Errorf("%q: %v, want %v", goos, got, want)
		}
	}
}
