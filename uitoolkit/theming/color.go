package theming

import (
	"errors"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"image/color"
	"math"
	"strings"
)

// ParseHex reads "#rrggbb", "rrggbb" or "#rgb" (case-insensitive).
func ParseHex(s string) (color.RGBA, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return color.RGBA{}, errors.New("a color is #rrggbb")
	}
	var r, g, b uint8
	if _, err := fmt.Sscanf(strings.ToLower(s), "%02x%02x%02x", &r, &g, &b); err != nil {
		return color.RGBA{}, errors.New("a color is #rrggbb")
	}
	return color.RGBA{R: r, G: g, B: b, A: 0xff}, nil
}

// FormatHex writes a color as "#rrggbb" in lower case.
func FormatHex(c color.RGBA) string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

func relLuminance(c color.RGBA) float64 {
	lin := func(v uint8) float64 {
		f := float64(v) / 255
		if f <= 0.03928 {
			return f / 12.92
		}
		return math.Pow((f+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
}

// ContrastRatio is the WCAG contrast ratio of two colors, from 1 (equal)
// to 21 (black on white).
func ContrastRatio(a, b color.RGBA) float64 {
	la, lb := relLuminance(a), relLuminance(b)
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
	if ContrastRatio(core.RGB(0xffffff), fg) > ContrastRatio(target, fg) {
		target = core.RGB(0xffffff)
	}
	mix := func(t float64) color.RGBA {
		m := func(a, b uint8) uint8 { return uint8(math.Round(float64(a) + (float64(b)-float64(a))*t)) }
		return color.RGBA{R: m(bg.R, target.R), G: m(bg.G, target.G), B: m(bg.B, target.B), A: 0xff}
	}
	for step := 1; step <= 255; step++ {
		if c := mix(float64(step) / 255); ContrastRatio(c, fg) >= min {
			return c
		}
	}
	return target
}
