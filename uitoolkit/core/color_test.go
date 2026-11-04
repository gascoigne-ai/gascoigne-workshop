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
	top, bottom := GradientEnds(base, 0.1)
	if !(top.R > base.R && bottom.R < base.R) {
		t.Fatalf("top %+v bottom %+v around %+v", top, bottom, base)
	}
}
