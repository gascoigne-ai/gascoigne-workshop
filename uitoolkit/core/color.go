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
	shaded := Mix(rgba, color.RGBA{R: uint8(target), G: uint8(target), B: uint8(target)}, k)
	shaded.A = rgba.A
	return shaded
}

// Mix blends from toward to by t (0 is from, 1 is to), every channel
// including alpha, rounded to the nearest value.
func Mix(from, to color.RGBA, t float64) color.RGBA {
	channel := func(start, end uint8) uint8 {
		return uint8(math.Round(float64(start) + (float64(end)-float64(start))*t))
	}
	return color.RGBA{R: channel(from.R, to.R), G: channel(from.G, to.G), B: channel(from.B, to.B), A: channel(from.A, to.A)}
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

// ParseHexStrict is ParseHex for text that must start with "#".
func ParseHexStrict(text string) (color.RGBA, error) {
	if !strings.HasPrefix(strings.TrimSpace(text), "#") {
		return color.RGBA{}, fmt.Errorf("%q is not #rgb or #rrggbb", text)
	}
	return ParseHex(text)
}

// ParseHexAlpha is ParseHex that also reads "#rrggbbaa".
func ParseHexAlpha(text string) (color.RGBA, error) {
	if len(text) != 9 {
		return ParseHex(text)
	}
	rgba, err := ParseHex(text[:7])
	if err != nil {
		return rgba, err
	}
	var alpha uint8
	if _, err := fmt.Sscanf(text[7:], "%02x", &alpha); err != nil {
		return rgba, fmt.Errorf("%q is not a color", text)
	}
	rgba.A = alpha
	return rgba, nil
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
func EnsureContrast(bg, fg color.RGBA, floor float64) color.RGBA {
	if ContrastRatio(bg, fg) >= floor {
		return bg
	}
	target := color.RGBA{A: 0xff}
	if ContrastRatio(RGB(0xffffff), fg) > ContrastRatio(target, fg) {
		target = RGB(0xffffff)
	}
	for step := 1; step <= 255; step++ {
		c := Mix(bg, target, float64(step)/255)
		c.A = 0xff
		if ContrastRatio(c, fg) >= floor {
			return c
		}
	}
	return target
}
