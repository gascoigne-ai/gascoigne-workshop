package widget

import (
	"image/color"
	"math"
	"testing"
)

func luminance(c color.RGBA) float64 {
	lin := func(v uint8) float64 {
		f := float64(v) / 255
		if f <= 0.03928 {
			return f / 12.92
		}
		return math.Pow((f+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
}

// Contrast is the WCAG contrast ratio between two colors.
func contrast(a, b color.RGBA) float64 {
	la, lb := luminance(a), luminance(b)
	return (max(la, lb) + 0.05) / (min(la, lb) + 0.05)
}

// Text must stay readable on every surface it is drawn on, in each theme.
func TestThemesAreReadable(t *testing.T) {
	for name, th := range map[string]Theme{"dark": Dark(), "light": Light()} {
		pairs := []struct {
			what   string
			fg, bg Token
			min    float64
		}{
			{"text on pane", Text, Bg, 4.5},
			{"text on list", Text, Sunken, 4.5},
			{"text on zebra row", Text, Alt, 4.5},
			{"text on selected row", Text, Selected, 4.5},
			{"text on frame", Text, Frame, 4.5},
			{"muted on pane", Muted, Bg, 3},
			{"muted on list", Muted, Sunken, 3},
			{"muted on frame", Muted, Frame, 3},
			{"chip text on chip", ChipText, Chip, 4.5},
			{"accent text on accent", AccentText, Accent, 4.5},
		}
		for _, p := range pairs {
			if got := contrast(th.Color(p.fg), th.Color(p.bg)); got < p.min {
				t.Errorf("%s: %s contrast %.2f < %.1f", name, p.what, got, p.min)
			}
		}
	}
}

func TestThemeWithDoesNotMutateTheOriginal(t *testing.T) {
	base := Dark()
	extra := base.With(map[Token]color.RGBA{"app.x": RGB(0x123456), Text: RGB(0xffffff)})
	if _, ok := base.Colors["app.x"]; ok || base.Color(Text) == RGB(0xffffff) {
		t.Fatal("With changed the receiver's colors")
	}
	if extra.Color("app.x") != RGB(0x123456) || extra.Color(Text) != RGB(0xffffff) {
		t.Fatal("With did not apply the extra tokens")
	}
	if extra.Color("unknown") != extra.Color(Text) {
		t.Fatal("unknown token must fall back to the text color")
	}
}
