package widget

import (
	"image/color"
	"strings"
	"testing"
)

func TestRasterizeSVGFillsShapesWithTheirColors(t *testing.T) {
	img, err := RasterizeSVG("t.svg", []byte(`<svg viewBox="0 0 10 10"><rect width="10" height="10" rx="4" fill="#ff0000"/></svg>`), 20)
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 20 || b.Dy() != 20 {
		t.Fatalf("bounds %v, want 20x20", b)
	}
	if got := img.RGBAAt(10, 10); got != (color.RGBA{255, 0, 0, 255}) {
		t.Errorf("middle %v, want solid red", got)
	}
	if got := img.RGBAAt(0, 0).A; got != 0 {
		t.Errorf("rounded corner has alpha %d, want 0", got)
	}
}

// A subpath wound the other way cuts a hole, which is how an SVG without
// the even-odd rule draws a ring.
func TestRasterizeSVGCutsHolesWoundTheOtherWay(t *testing.T) {
	src := `<svg viewBox="0 0 10 10"><path fill="#00ff00" d="M0 0H10V10H0Z M3 3V7H7V3Z"/></svg>`
	img, err := RasterizeSVG("t.svg", []byte(src), 10)
	if err != nil {
		t.Fatal(err)
	}
	if got := img.RGBAAt(5, 5).A; got != 0 {
		t.Errorf("hole has alpha %d, want 0", got)
	}
	if got := img.RGBAAt(1, 1).A; got != 255 {
		t.Errorf("ring has alpha %d, want 255", got)
	}
}

func TestRasterizeSVGLinearGradientRunsBetweenItsStops(t *testing.T) {
	src := `<svg viewBox="0 0 10 10"><defs><linearGradient id="g" x1="0" y1="0" x2="0" y2="10" gradientUnits="userSpaceOnUse"><stop offset="0" stop-color="#000000"/><stop offset="1" stop-color="#ffffff"/></linearGradient></defs><rect width="10" height="10" fill="url(#g)"/></svg>`
	img, err := RasterizeSVG("t.svg", []byte(src), 100)
	if err != nil {
		t.Fatal(err)
	}
	top, middle, bottom := img.RGBAAt(50, 1).R, img.RGBAAt(50, 50).R, img.RGBAAt(50, 98).R
	if top > 10 || bottom < 245 || middle < 118 || middle > 138 {
		t.Errorf("gradient top %d, middle %d, bottom %d, want about 0, 128, 255", top, middle, bottom)
	}
}

func TestRasterizeSVGRefusesWhatItCannotDraw(t *testing.T) {
	_, err := RasterizeSVG("t.svg", []byte(`<svg viewBox="0 0 10 10" fill="none" stroke="#fff"><path d="M0 0L9 9"/></svg>`), 10)
	if err == nil || !strings.Contains(err.Error(), "stroke") {
		t.Fatalf("error %v, want one about stroke", err)
	}
}

// Icons follow the theme, so an icon file with its own color is an error.
func TestIconFilesCannotCarryTheirOwnColors(t *testing.T) {
	_, err := parseSVG("t.svg", []byte(`<svg viewBox="0 0 10 10"><path d="M0 0L9 9" fill="#ff0000"/></svg>`))
	if err == nil || !strings.Contains(err.Error(), "token(name)") {
		t.Fatalf("error %v, want the paint rule", err)
	}
}
