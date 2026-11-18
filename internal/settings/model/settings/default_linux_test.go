package settings

import "testing"

func TestLinuxDefaultsToTenPercentBiggerInterface(t *testing.T) {
	if got := Default().UIScalePct; got != 110 {
		t.Fatalf("default scale %d%%, want 110%%", got)
	}
	if got := (Settings{}).UIScale(); got != 1.1 {
		t.Fatalf("unset scale %v, want 1.1", got)
	}
}
