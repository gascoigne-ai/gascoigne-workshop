package theming

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
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
	for name, th := range map[string]core.Theme{"dark": Dark(), "light": Light()} {
		pairs := []struct {
			what   string
			fg, bg core.Token
			min    float64
		}{
			{"text on pane", core.Text, core.Bg, 4.5},
			{"text on list", core.Text, core.Sunken, 4.5},
			{"text on zebra row", core.Text, core.Alt, 4.5},
			{"text on selected row", core.Text, core.Selected, 4.5},
			{"text on frame", core.Text, core.Frame, 4.5},
			{"text on selection", core.Text, core.Selection, 4.5},
			{"text on a find match", core.Text, core.FindMatch, 4.5},
			{"text on the active find match", core.Text, core.FindActive, 4.5},
			{"muted on pane", core.Muted, core.Bg, 3},
			{"muted on list", core.Muted, core.Sunken, 3},
			{"muted on frame", core.Muted, core.Frame, 3},
			{"chip text on chip", core.ChipText, core.Chip, 4.5},
			{"accent text on accent", core.AccentText, core.Accent, 4.5},
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
	extra := base.With(map[core.Token]color.RGBA{"app.x": core.RGB(0x123456), core.Text: core.RGB(0xffffff)})
	if _, ok := base.Colors["app.x"]; ok || base.Color(core.Text) == core.RGB(0xffffff) {
		t.Fatal("With changed the receiver's colors")
	}
	if extra.Color("app.x") != core.RGB(0x123456) || extra.Color(core.Text) != core.RGB(0xffffff) {
		t.Fatal("With did not apply the extra tokens")
	}
	if extra.Color("unknown") != extra.Color(core.Text) {
		t.Fatal("unknown token must fall back to the text color")
	}
}

// Separators, outlines and highlights are not text, but they still have
// to be seen against the surfaces they sit on (WCAG asks 3:1 for
// controls; plain dividers get a lower bar, since the change of surface
// beside them also shows the edge).
func TestNonTextElementsAreVisible(t *testing.T) {
	for name, th := range map[string]core.Theme{"dark": Dark(), "light": Light()} {
		pairs := []struct {
			what string
			a, b core.Token
			min  float64
		}{
			{"divider on a pane", core.Border, core.Bg, 2.0},
			{"divider on a list", core.Border, core.Sunken, 2.0},
			{"rule on a pane", core.Rule, core.Bg, 2.5},
			{"rule on a list", core.Rule, core.Sunken, 2.5},
			{"rule on the frame", core.Rule, core.Frame, 2.0},
			{"accent line on a pane", core.AccentLine, core.Bg, 3},
			{"accent line on a list", core.AccentLine, core.Sunken, 3},
			{"accent line on a selected row", core.AccentLine, core.Selected, 3},
			{"icon on a pane", core.Muted, core.Bg, 3},
			{"scrollbar thumb on a list", core.Muted, core.Sunken, 3},
			{"selected row on a list", core.Selected, core.Sunken, 1.3},
			{"hovered row on a list", core.Hover, core.Sunken, 1.15},
			{"zebra row on a list", core.Alt, core.Sunken, 1.05},
			{"chip on a pane", core.Chip, core.Bg, 1.5},
		}
		for _, p := range pairs {
			if got := core.ContrastRatio(th.Color(p.a), th.Color(p.b)); got < p.min {
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
	want := map[string]map[core.Token]uint32{
		"dark": {
			core.Text: 0xf1e8eb, core.Muted: 0xbfaab2, core.Accent: 0xe88aa6, core.OK: 0x5ed39a, core.Error: 0xff6b5e,
			core.Frame: 0x140f12, core.Sunken: 0x1c1519, core.Bg: 0x1c1519, core.Alt: 0x241b20, core.Panel: 0x241b20,
		},
		"light": {
			core.Text: 0x22161b, core.Muted: 0x66545b, core.Accent: 0x7a1f3d, core.OK: 0x17804a, core.Error: 0xd1281b,
			core.Frame: 0xf6f3f4, core.Bg: 0xf9f6f7, core.Sunken: 0xffffff, core.Panel: 0xf5f0f2,
		},
	}
	for name, th := range map[string]core.Theme{"dark": Dark(), "light": Light()} {
		for tok, hex := range want[name] {
			if got := th.Color(tok); got != core.RGB(hex) {
				t.Errorf("%s %s = %s, want %s", name, tok, core.FormatHex(got), core.FormatHex(core.RGB(hex)))
			}
		}
	}
}

// Every gradient surface keeps its text readable at both ends of the fill.
func TestThemeGradientsKeepTextReadable(t *testing.T) {
	surfaces := []struct {
		control   string
		base, fgt core.Token
	}{
		{"hero", "hero", "hero-text"}, // the dashboard has no other gradient surface; a theme may add them
	}
	for name, th := range map[string]core.Theme{"dark": Dark(), "light": Light()} {
		for _, s := range surfaces {
			st := th.Style(s.control)
			if !st.HasGradient() {
				t.Errorf("%s: %s has no gradient", name, s.control)
				continue
			}
			top, bottom := core.GradientFor(th.Color(s.base), st)
			for end, c := range map[string]color.RGBA{"top": top, "bottom": bottom} {
				if r := core.ContrastRatio(c, th.Color(s.fgt)); r < 4.5 {
					t.Errorf("%s: %v text on the %s of %s: %.2f", name, s.fgt, end, s.control, r)
				}
			}
		}
	}
}
