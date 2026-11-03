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
			{"text on selection", Text, Selection, 4.5},
			{"text on a find match", Text, FindMatch, 4.5},
			{"text on the active find match", Text, FindActive, 4.5},
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

// Separators, outlines and highlights are not text, but they still have
// to be seen against the surfaces they sit on (WCAG asks 3:1 for
// controls; plain dividers get a lower bar, since the change of surface
// beside them also shows the edge).
func TestNonTextElementsAreVisible(t *testing.T) {
	for name, th := range map[string]Theme{"dark": Dark(), "light": Light()} {
		pairs := []struct {
			what string
			a, b Token
			min  float64
		}{
			{"divider on a pane", Border, Bg, 2.0},
			{"divider on a list", Border, Sunken, 2.0},
			{"rule on a pane", Rule, Bg, 2.5},
			{"rule on a list", Rule, Sunken, 2.5},
			{"rule on the frame", Rule, Frame, 2.0},
			{"accent line on a pane", AccentLine, Bg, 3},
			{"accent line on a list", AccentLine, Sunken, 3},
			{"accent line on a selected row", AccentLine, Selected, 3},
			{"icon on a pane", Muted, Bg, 3},
			{"scrollbar thumb on a list", Muted, Sunken, 3},
			{"selected row on a list", Selected, Sunken, 1.3},
			{"hovered row on a list", Hover, Sunken, 1.15},
			{"zebra row on a list", Alt, Sunken, 1.05},
			{"chip on a pane", Chip, Bg, 1.5},
		}
		for _, p := range pairs {
			if got := ContrastRatio(th.Color(p.a), th.Color(p.b)); got < p.min {
				t.Errorf("%s: %s: %s on %s is %.2f:1, want at least %.2f:1", name, p.what, p.a, p.b, got, p.min)
			}
		}
	}
}
