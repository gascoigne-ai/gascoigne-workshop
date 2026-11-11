package core

import (
	"image/color"
	"testing"
)

func TestShadeMovesTowardWhiteOrBlackAndKeepsAlpha(t *testing.T) {
	c := color.RGBA{R: 100, G: 100, B: 100, A: 255}
	up, down := Shade(c, 0.5), Shade(c, -0.5)
	if up.R != 178 || up.A != 255 || down.R != 50 || down.A != 255 {
		t.Fatalf("up %+v down %+v", up, down)
	}
	if Shade(c, 0) != c {
		t.Fatal("zero is no change")
	}
	if w := Shade(color.RGBA{R: 255, G: 255, B: 255, A: 255}, 0.3); w.R != 255 {
		t.Fatalf("white stays white: %+v", w)
	}
}

func TestGradientEndsBracketTheBaseColor(t *testing.T) {
	base := color.RGBA{R: 120, G: 80, B: 90, A: 255}
	top, bottom := gradientEnds(base, 0.1)
	if !(top.R > base.R && bottom.R < base.R) {
		t.Fatalf("top %+v bottom %+v around %+v", top, bottom, base)
	}
}

func TestParseHexTakesThreeAndSixDigitColorsAndRefusesOthers(t *testing.T) {
	tests := []struct {
		in      string
		want    color.RGBA
		wantErr bool
	}{
		{"#ffc90e", color.RGBA{0xff, 0xc9, 0x0e, 0xff}, false},
		{"FFC90E", color.RGBA{0xff, 0xc9, 0x0e, 0xff}, false},
		{"  #abc  ", color.RGBA{0xaa, 0xbb, 0xcc, 0xff}, false},
		{"#000", color.RGBA{0, 0, 0, 0xff}, false},
		{"", color.RGBA{}, true},
		{"#12345", color.RGBA{}, true},
		{"#gggggg", color.RGBA{}, true},
		{"#1234567", color.RGBA{}, true},
		{"red", color.RGBA{}, true},
	}
	for _, tt := range tests {
		got, err := ParseHex(tt.in)
		if (err != nil) != tt.wantErr || (err == nil && got != tt.want) {
			t.Errorf("ParseHex(%q) = %v, %v; want %v, error %v", tt.in, got, err, tt.want, tt.wantErr)
		}
	}
}

func TestFormatHexRoundTrips(t *testing.T) {
	for _, hex := range []string{"#000000", "#ffffff", "#ffc90e", "#0a0b0c"} {
		c, err := ParseHex(hex)
		if err != nil || FormatHex(c) != hex {
			t.Errorf("%s -> %v -> %s", hex, c, FormatHex(c))
		}
	}
}

func TestContrastRatioIsSymmetricAndRunsFromOneToTwentyOne(t *testing.T) {
	black, white := RGB(0), RGB(0xffffff)
	if got := ContrastRatio(black, white); got < 20.9 || got > 21.1 {
		t.Errorf("black on white = %v", got)
	}
	if got := ContrastRatio(white, black); got < 20.9 || got > 21.1 {
		t.Errorf("order must not matter, got %v", got)
	}
	if got := ContrastRatio(RGB(0x777777), RGB(0x777777)); got != 1 {
		t.Errorf("equal colors = %v", got)
	}
	// the well-known threshold color: #767676 on white is just over 4.5
	if got := ContrastRatio(RGB(0x767676), white); got < 4.5 || got > 4.6 {
		t.Errorf("#767676 on white = %v", got)
	}
}

func TestEnsureContrastMovesABackgroundJustFarEnoughFromItsText(t *testing.T) {
	light, dark := RGB(0xdadada), RGB(0x262626)
	tests := []struct {
		name   string
		bg, fg color.RGBA
		min    float64
	}{
		{"a mid gray under light text goes darker", RGB(0x777777), light, 7},
		{"a mid gray under dark text goes lighter", RGB(0x777777), dark, 7},
		{"gold under light text", RGB(0xffc90e), light, 4.5},
		{"already fine", RGB(0x101010), light, 4.5},
	}
	for _, tt := range tests {
		got := EnsureContrast(tt.bg, tt.fg, tt.min)
		if r := ContrastRatio(got, tt.fg); r < tt.min {
			t.Errorf("%s: ratio %.2f < %.1f (%v)", tt.name, r, tt.min, got)
		}
	}
	if got := EnsureContrast(RGB(0x101010), light, 4.5); got != RGB(0x101010) {
		t.Errorf("a color that already meets the ratio must not change, got %v", got)
	}
	darker := EnsureContrast(RGB(0x777777), light, 7)
	if relLuminance(darker) >= relLuminance(RGB(0x777777)) {
		t.Errorf("under light text the background must get darker: %v", darker)
	}
	lighter := EnsureContrast(RGB(0x777777), dark, 7)
	if relLuminance(lighter) <= relLuminance(RGB(0x777777)) {
		t.Errorf("under dark text the background must get lighter: %v", lighter)
	}
	if got := EnsureContrast(RGB(0x888888), RGB(0x888888), 21); ContrastRatio(got, RGB(0x888888)) < 4 {
		t.Errorf("an impossible ratio still gets as far as it can: %v", got)
	}
}
