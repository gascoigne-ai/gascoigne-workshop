package displayscale

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

// The first seed is the captured hyprctl output in testdata; the others
// are synthetic edge cases (sway's shape, odd scales).
func FuzzParseMonitors(fuzz *testing.F) {
	if data, err := os.ReadFile(filepath.Join("testdata", "hyprctl_monitors.json")); err == nil {
		fuzz.Add(data)
	}
	fuzz.Add([]byte(`[{"name":"eDP-1","scale":1.5,"focused":true}]`))
	fuzz.Add([]byte(`[{"scale":-2},{"scale":1e999}]`))
	fuzz.Add([]byte(`[]`))
	fuzz.Fuzz(func(t *testing.T, data []byte) {
		scale := parseMonitors(data)
		if scale < 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
			t.Fatalf("scale %v", scale)
		}
	})
}
