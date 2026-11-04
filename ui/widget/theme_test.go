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

// The palettes follow the gascoigne CLI dashboard's color tokens
// (cmd/gascoigne/pages.css in gascoigne-cli): its --text, --muted,
// --accent, --ok and --err are our Text, Muted, Accent, OK and Error,
// and its surfaces map to our Frame, Bg and Sunken. The values below are
// copied from there. Borders and rules are not pinned: they keep the
// CLI's hues but must be strong enough to see natively.
func TestPalettesFollowTheCLIDashboard(t *testing.T) {
	want := map[string]map[Token]uint32{
		"dark": {
			Text: 0xf1e8eb, Muted: 0xbfaab2, Accent: 0xe88aa6, OK: 0x5ed39a, Error: 0xff6b5e,
			Frame: 0x140f12, Sunken: 0x1c1519, Bg: 0x1c1519, Alt: 0x241b20, Panel: 0x241b20,
		},
		"light": {
			Text: 0x22161b, Muted: 0x66545b, Accent: 0x7a1f3d, OK: 0x17804a, Error: 0xd1281b,
			Frame: 0xf6f3f4, Bg: 0xffffff, Sunken: 0xffffff, Panel: 0xf5f0f2,
		},
	}
	for name, th := range map[string]Theme{"dark": Dark(), "light": Light()} {
		for tok, hex := range want[name] {
			if got := th.Color(tok); got != RGB(hex) {
				t.Errorf("%s %s = %s, want %s", name, tok, FormatHex(got), FormatHex(RGB(hex)))
			}
		}
	}
}

// Every gradient surface keeps its text readable at both ends of the fill.
func TestThemeGradientsKeepTextReadable(t *testing.T) {
	surfaces := []struct {
		control   string
		base, fgt Token
	}{
		{"hero", "hero", "hero-text"}, // the dashboard has no other gradient surface; a theme may add them
	}
	for name, th := range map[string]Theme{"dark": Dark(), "light": Light()} {
		for _, s := range surfaces {
			st := th.Style(s.control)
			if !st.HasGradient() {
				t.Errorf("%s: %s has no gradient", name, s.control)
				continue
			}
			top, bottom := GradientFor(th.Color(s.base), st)
			for end, c := range map[string]color.RGBA{"top": top, "bottom": bottom} {
				if r := ContrastRatio(c, th.Color(s.fgt)); r < 4.5 {
					t.Errorf("%s: %v text on the %s of %s: %.2f", name, s.fgt, end, s.control, r)
				}
			}
		}
	}
}
