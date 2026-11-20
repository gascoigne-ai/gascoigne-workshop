package view

import (
	"image"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestReaches(t *testing.T) {
	clip := image.Rect(100, 100, 200, 200)
	tests := []struct {
		name  string
		rect  core.Rect
		scale float64
		reach int
		want  bool
	}{
		{"inside", core.Rect{X: 120, Y: 120, W: 20, H: 20}, 1, 0, true},
		{"straddling the edge", core.Rect{X: 90, Y: 120, W: 20, H: 20}, 1, 0, true},
		{"above", core.Rect{X: 120, Y: 60, W: 20, H: 20}, 1, 0, false},
		{"below", core.Rect{X: 120, Y: 220, W: 20, H: 20}, 1, 0, false},
		{"left", core.Rect{X: 60, Y: 120, W: 20, H: 20}, 1, 0, false},
		{"right", core.Rect{X: 220, Y: 120, W: 20, H: 20}, 1, 0, false},
		{"touching the bottom edge", core.Rect{X: 120, Y: 200, W: 20, H: 20}, 1, 0, false},
		{"touching the top edge from above", core.Rect{X: 120, Y: 80, W: 20, H: 20}, 1, 0, false},
		{"only the shadow reaches in", core.Rect{X: 60, Y: 120, W: 38, H: 20}, 1, 5, true},
		{"too far for the shadow", core.Rect{X: 40, Y: 120, W: 38, H: 20}, 1, 5, false},
		{"zero size inside", core.Rect{X: 150, Y: 150}, 1, 0, true},
		{"zero size outside", core.Rect{X: 20, Y: 20}, 1, 5, false},
		{"scale 1.25 inside", core.Rect{X: 80, Y: 80, W: 20, H: 20}, 1.25, 0, true},
		{"scale 1.25 ends at device 100, touching", core.Rect{X: 60, Y: 120, W: 20, H: 20}, 1.25, 0, false},
		{"scale 1.25 ends past device 101", core.Rect{X: 60, Y: 120, W: 21, H: 20}, 1.25, 0, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := reaches(clip, test.rect, test.scale, test.reach); got != test.want {
				t.Errorf("reaches(%v, %+v, %v, %d) = %v, want %v", clip, test.rect, test.scale, test.reach, got, test.want)
			}
		})
	}
}
