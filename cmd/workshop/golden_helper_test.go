//go:build !release

package main

import (
	"flag"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

const (
	// shotTolerance is how far a channel may move before a pixel counts as
	// different: another Mesa version rounds blends differently.
	shotTolerance = 4
	// maxDifferingPixels is the budget of changed pixels per shot (about
	// 0.01% of a 1280x800 window); a color or layout change blows past it.
	maxDifferingPixels = 100
	goldenDir          = "testdata/golden"
)

var updateGolden = flag.Bool("update", false, "rewrite the golden shots with what the program draws now")

// diffMark paints a differing pixel in the diff image.
var diffMark = color.RGBA{R: 255, A: 255}

// imageDiff is how two images differ.
type imageDiff struct {
	SizeDiffers bool
	Differing   int // pixels with a channel off by more than shotTolerance
}

func (diff imageDiff) withinBudget() bool {
	return !diff.SizeDiffers && diff.Differing <= maxDifferingPixels
}

func compareImages(want, got image.Image) imageDiff {
	if want.Bounds() != got.Bounds() {
		return imageDiff{SizeDiffers: true}
	}
	differing := 0
	forEachPixel(want, func(x, y int) {
		if pixelsDiffer(want.At(x, y), got.At(x, y)) {
			differing++
		}
	})
	return imageDiff{Differing: differing}
}

// diffImage has the size of want and marks the pixels that differ.
func diffImage(want, got image.Image) *image.RGBA {
	diff := image.NewRGBA(want.Bounds())
	forEachPixel(want, func(x, y int) {
		if pixelsDiffer(want.At(x, y), got.At(x, y)) {
			diff.SetRGBA(x, y, diffMark)
		}
	})
	return diff
}

func forEachPixel(img image.Image, visit func(x, y int)) {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			visit(x, y)
		}
	}
}

func pixelsDiffer(want, got color.Color) bool {
	wantRGBA, gotRGBA := color.RGBAModel.Convert(want).(color.RGBA), color.RGBAModel.Convert(got).(color.RGBA)
	return channelDiffers(wantRGBA.R, gotRGBA.R) || channelDiffers(wantRGBA.G, gotRGBA.G) ||
		channelDiffers(wantRGBA.B, gotRGBA.B) || channelDiffers(wantRGBA.A, gotRGBA.A)
}

func channelDiffers(want, got uint8) bool {
	if want > got {
		return want-got > shotTolerance
	}
	return got-want > shotTolerance
}

// assertGolden compares img with testdata/golden/<name>.png. On a mismatch it
// writes <name>.got.png and <name>.diff.png next to the golden for a look;
// `go test ./cmd/workshop -update` rewrites the golden.
func assertGolden(t *testing.T, name string, img image.Image) {
	t.Helper()
	path := filepath.Join(goldenDir, name+".png")
	if *updateGolden {
		writePNG(t, path, img)
		return
	}
	want := readGolden(t, path)
	diff := compareImages(want, img)
	if diff.withinBudget() {
		return
	}
	gotPath := filepath.Join(goldenDir, name+".got.png")
	writePNG(t, gotPath, img)
	if !diff.SizeDiffers {
		writePNG(t, filepath.Join(goldenDir, name+".diff.png"), diffImage(want, img))
	}
	t.Errorf("%s differs from its golden: size differs %v, %d pixels off (budget %d); see %s, then -update if the change is wanted",
		name, diff.SizeDiffers, diff.Differing, maxDifferingPixels, gotPath)
}

func readGolden(t *testing.T, path string) image.Image {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("no golden at %s: run with -update and look at it before committing", path)
	}
	return readPNG(t, path)
}

func writePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create %s: %v", filepath.Dir(path), err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}
}
