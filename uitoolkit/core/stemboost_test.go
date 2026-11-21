package core

import (
	"math"
	"testing"
)

// composite is the coverage a pixel ends at when the passes are drawn one
// over the other with source-over blending, as the GPU blends them.
func composite(passes []float32, coverage float64) float64 {
	uncovered := 1.0
	for _, alpha := range passes {
		uncovered *= 1 - coverage*float64(alpha)
	}
	return 1 - uncovered
}

func TestStemBoostDarkensPartialCoverageOnly(t *testing.T) {
	tests := []struct {
		name     string
		coverage float64
		want     float64
	}{
		{"an empty pixel stays empty", 0, 0},
		{"a full pixel stays full", 1, 1},
		{"a half covered edge gets heavier", 0.5, 0.5 + StemBoost*0.25},
		{"a faint edge gets heavier", 0.2, 0.2 + StemBoost*0.16},
		{"a nearly full edge gets a little heavier", 0.9, 0.9 + StemBoost*0.09},
	}
	passes := textPasses(StemBoost)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := composite(passes, test.coverage)
			if math.Abs(got-test.want) > 1e-6 {
				t.Errorf("coverage %.2f ends at %.4f, want %.4f", test.coverage, got, test.want)
			}
		})
	}
}

// The boost makes text slightly heavier, not bold: a half covered pixel
// stays well under the coverage a bold stem would give it.
func TestStemBoostIsSlightNotBold(t *testing.T) {
	half := composite(textPasses(StemBoost), 0.5)
	if half < 0.55 || half > 0.65 {
		t.Errorf("a half covered pixel ends at %.3f, want between 0.55 and 0.65", half)
	}
}

func TestStemBoostKeepsCoverageOrder(t *testing.T) {
	passes := textPasses(StemBoost)
	previous := 0.0
	for step := 1; step <= 255; step++ {
		got := composite(passes, float64(step)/255)
		if got < previous {
			t.Fatalf("coverage %d/255 ends at %.4f, below the step before (%.4f)", step, got, previous)
		}
		previous = got
	}
}
