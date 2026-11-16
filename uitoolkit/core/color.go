package core

import (
	"errors"
	"fmt"
	"image/color"
	"math"
	"strings"
)

// Shade mixes c toward white (amount > 0) or black (amount < 0); the
// alpha is kept.
func Shade(rgba color.RGBA, amount float64) color.RGBA {
	target, k := 0.0, -amount
	if amount > 0 {
		target, k = 255, amount
	}
	mix := func(value uint8) uint8 { return uint8(float64(value) + (target-float64(value))*k + 0.5) }
	return color.RGBA{R: mix(rgba.R), G: mix(rgba.G), B: mix(rgba.B), A: rgba.A}
}

// gradientEnds are the top and bottom colors of a vertical gradient
// around base: lighter above, darker below, by strength.
func gradientEnds(base color.RGBA, strength float64) (top, bottom color.RGBA) {
	return Shade(base, strength), Shade(base, -strength)
}

// RGB converts 0xRRGGBB to a color.
func RGB(hex uint32) color.RGBA {
	return color.RGBA{R: uint8(hex >> 16), G: uint8(hex >> 8), B: uint8(hex), A: 0xff}
}

// Lighten nudges a color toward white, for hover states.
func Lighten(rgba color.RGBA) color.RGBA {
	const step = 16
	up := func(value uint8) uint8 { return uint8(min(int(value)+step, 255)) }
	return color.RGBA{R: up(rgba.R), G: up(rgba.G), B: up(rgba.B), A: rgba.A}
}

// GradientFor is the top and bottom colors a control's fill takes from
// base: the style's explicit ends, else base shaded by the style's
// strength, else base twice.
func GradientFor(base color.RGBA, st ControlStyle) (top, bottom color.RGBA) {
	switch {
	case st.Top != nil && st.Bottom != nil:
		return *st.Top, *st.Bottom
	case st.Shade > 0:
		return gradientEnds(base, st.Shade)
	}
	return base, base
}

// ParseHex reads "#rrggbb", "rrggbb" or "#rgb" (case-insensitive).
func ParseHex(text string) (color.RGBA, error) {
	text = strings.TrimPrefix(strings.TrimSpace(text), "#")
	if len(text) == 3 {
		text = string([]byte{text[0], text[0], text[1], text[1], text[2], text[2]})
	}
	if len(text) != 6 {
		return color.RGBA{}, errors.New("a color is #rrggbb")
	}
	var r, g, b uint8
	if _, err := fmt.Sscanf(strings.ToLower(text), "%02x%02x%02x", &r, &g, &b); err != nil {
		return color.RGBA{}, errors.New("a color is #rrggbb")
	}
	return color.RGBA{R: r, G: g, B: b, A: 0xff}, nil
}

// FormatHex writes a color as "#rrggbb" in lower case.
func FormatHex(rgba color.RGBA) string { return fmt.Sprintf("#%02x%02x%02x", rgba.R, rgba.G, rgba.B) }

func relLuminance(rgba color.RGBA) float64 {
	lin := func(value uint8) float64 {
		f := float64(value) / 255
		if f <= 0.03928 {
			return f / 12.92
		}
		return math.Pow((f+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(rgba.R) + 0.7152*lin(rgba.G) + 0.0722*lin(rgba.B)
}

// ContrastRatio is the WCAG contrast ratio of two colors, from 1 (equal)
// to 21 (black on white).
func ContrastRatio(rgba, other color.RGBA) float64 {
	la, lb := relLuminance(rgba), relLuminance(other)
	return (math.Max(la, lb) + 0.05) / (math.Min(la, lb) + 0.05)
}

// EnsureContrast returns bg moved away from fg, toward whichever of
// black and white contrasts more with it, just far enough that the
// contrast ratio reaches min. A color that already meets it is returned
// unchanged; when min cannot be reached the extreme is returned.
func EnsureContrast(bg, fg color.RGBA, min float64) color.RGBA {
	if ContrastRatio(bg, fg) >= min {
		return bg
	}
	target := color.RGBA{A: 0xff}
	if ContrastRatio(RGB(0xffffff), fg) > ContrastRatio(target, fg) {
		target = RGB(0xffffff)
	}
	mix := func(ratio float64) color.RGBA {
		m := func(value, uint8Value uint8) uint8 {
			return uint8(math.Round(float64(value) + (float64(uint8Value)-float64(value))*ratio))
		}
		return color.RGBA{R: m(bg.R, target.R), G: m(bg.G, target.G), B: m(bg.B, target.B), A: 0xff}
	}
	for step := 1; step <= 255; step++ {
		if c := mix(float64(step) / 255); ContrastRatio(c, fg) >= min {
			return c
		}
	}
	return target
}
