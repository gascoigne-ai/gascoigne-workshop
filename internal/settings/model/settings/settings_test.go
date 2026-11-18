package settings

import "testing"

func TestUIScaleDefaultsToTheDisplayScaleAndIsBounded(t *testing.T) {
	if got := Default().UIScale(); got != float64(DefaultUIScalePct)/100 {
		t.Fatalf("default %v", got)
	}
	for _, tt := range []struct {
		pct  int
		want float64
	}{{0, float64(DefaultUIScalePct) / 100}, {100, 1}, {150, 1.5}, {5, 0.5}, {9000, 3}} {
		if got := (Settings{UIScalePct: tt.pct}).UIScale(); got != tt.want {
			t.Errorf("%d%% -> %v, want %v", tt.pct, got, tt.want)
		}
	}
}
