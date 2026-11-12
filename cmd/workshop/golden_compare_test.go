//go:build !release

package main

import (
	"image"
	"image/color"
	"testing"
)

func solid(width, height int, fill color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, fill)
		}
	}
	return img
}

func TestCompareImagesCountsPixelsBeyondTheTolerance(t *testing.T) {
	base := color.RGBA{R: 100, G: 100, B: 100, A: 255}
	tests := []struct {
		name         string
		change       func(img *image.RGBA)
		got          *image.RGBA
		wantSizeDiff bool
		wantDiffer   int
	}{
		{name: "identical", change: func(*image.RGBA) {}},
		{name: "within the tolerance", change: func(img *image.RGBA) { img.SetRGBA(3, 3, color.RGBA{R: 100 + shotTolerance, G: 100, B: 100, A: 255}) }},
		{name: "one channel over the tolerance", change: func(img *image.RGBA) {
			img.SetRGBA(3, 3, color.RGBA{R: 100, G: 100, B: 100 + shotTolerance + 1, A: 255})
		}, wantDiffer: 1},
		{name: "two pixels changed", change: func(img *image.RGBA) {
			img.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
			img.SetRGBA(9, 9, color.RGBA{G: 255, A: 255})
		}, wantDiffer: 2},
		{name: "different size", got: solid(9, 10, base), wantSizeDiff: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			want := solid(10, 10, base)
			got := test.got
			if got == nil {
				got = solid(10, 10, base)
				test.change(got)
			}
			result := compareImages(want, got)
			if result.SizeDiffers != test.wantSizeDiff || result.Differing != test.wantDiffer {
				t.Errorf("compareImages = %+v, want size differs %v and %d differing pixels", result, test.wantSizeDiff, test.wantDiffer)
			}
		})
	}
}

func TestDiffImageMarksOnlyTheDifferingPixels(t *testing.T) {
	want := solid(4, 4, color.RGBA{A: 255})
	got := solid(4, 4, color.RGBA{A: 255})
	got.SetRGBA(2, 1, color.RGBA{R: 200, A: 255})
	diff := diffImage(want, got)
	if diff.At(2, 1) != diffMark {
		t.Errorf("pixel (2,1) = %v, want the mark %v", diff.At(2, 1), diffMark)
	}
	if diff.At(0, 0) == diffMark {
		t.Error("pixel (0,0) is marked but did not change")
	}
}

func TestWithinBudgetAllowsAFewPixels(t *testing.T) {
	tests := []struct {
		name   string
		result imageDiff
		want   bool
	}{
		{"no difference", imageDiff{}, true},
		{"a few pixels", imageDiff{Differing: maxDifferingPixels}, true},
		{"too many pixels", imageDiff{Differing: maxDifferingPixels + 1}, false},
		{"size differs", imageDiff{SizeDiffers: true}, false},
	}
	for _, test := range tests {
		if got := test.result.withinBudget(); got != test.want {
			t.Errorf("%s: withinBudget = %v, want %v", test.name, got, test.want)
		}
	}
}
