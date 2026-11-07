package appearance

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/theming"
	"image/color"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func withColors(theme string, colors map[string]string) settings.Settings {
	s := settings.Default()
	s.Theme = theme
	s.Colors = map[string]map[string]string{theme: colors}
	return s
}

func TestThemeWithDefaultsEqualsThemeFor(t *testing.T) {
	for _, tc := range ThemeChoices {
		s := settings.Default()
		s.Theme = tc.ID
		want, got := ThemeFor(tc.ID, s.TextSize), ThemeWith(s)
		for _, c := range CustomColors {
			if want.Color(c.Token) != got.Color(c.Token) {
				t.Errorf("%s: %s differs without overrides", tc.ID, c.Token)
			}
		}
		if want.FontSize != got.FontSize {
			t.Errorf("%s: font size", tc.ID)
		}
	}
}

func TestAnOverrideAppliesToItsThemeOnly(t *testing.T) {
	s := withColors("dark", map[string]string{"selection": "#223344"})
	if got := ThemeWith(s).Color(core.Selection); got != core.RGB(0x223344) {
		t.Fatalf("dark selection = %v", got)
	}
	s.Theme = "light" // the override belongs to dark
	if got := ThemeWith(s).Color(core.Selection); got != ThemeFor("light", "medium").Color(core.Selection) {
		t.Fatalf("light selection = %v, want the light default", got)
	}
}

func TestBadOverridesAreIgnored(t *testing.T) {
	def := ThemeFor("dark", "medium")
	s := withColors("dark", map[string]string{
		"selection":   "not a color",
		"find-match":  "#ffffff", // white under light text: unreadable
		"nonsense":    "#123456", // not a customizable token
		"text":        "#000000", // customizable tokens only: text is not one
		"find-active": "#6e5a10", // the default value: fine
	})
	th := ThemeWith(s)
	if th.Color(core.Selection) != def.Color(core.Selection) || th.Color(core.FindMatch) != def.Color(core.FindMatch) {
		t.Fatal("unparsable and unreadable overrides must be ignored")
	}
	if th.Color(core.Text) != def.Color(core.Text) || th.Color("nonsense") != def.Color(core.Text) {
		t.Fatal("only the listed tokens can be overridden")
	}
	if th.Color(core.FindActive) != core.RGB(0x6e5a10) {
		t.Fatal("a valid override applies")
	}
}

func TestHighContrastMakesTheListedColorsReadableInEveryTheme(t *testing.T) {
	for _, tc := range ThemeChoices {
		s := settings.Default()
		s.Theme, s.Contrast = tc.ID, "high"
		th := ThemeWith(s)
		for _, c := range CustomColors {
			if r := theming.ContrastRatio(th.Color(c.Token), th.Color(c.Text)); r < HighRatio-0.01 {
				t.Errorf("%s: %s has contrast %.2f, want at least %.1f", tc.ID, c.Token, r, HighRatio)
			}
		}
		normal := ThemeFor(tc.ID, s.TextSize)
		// the surfaces stay; the text and lines drawn on them are what get stronger
		// (see theming/highcontrast_test.go)
		if th.Color(core.Bg) != normal.Color(core.Bg) || th.Color(core.Frame) != normal.Color(core.Frame) || th.Color(core.Sunken) != normal.Color(core.Sunken) {
			t.Errorf("%s: high contrast must not move the surfaces", tc.ID)
		}
	}
}

func TestHighContrastRaisesTheBarForOverrides(t *testing.T) {
	// a color readable in normal contrast (>= 4.5) but not under high
	// contrast's own, stronger text (< 7)
	hs := settings.Default()
	hs.Contrast = "high"
	highText := ThemeWith(hs).Color(core.Text)
	normalText := ThemeFor("dark", "medium").Color(core.Text)
	var hex string
	var c color.RGBA
	for v := uint8(0x30); v < 0xa0 && hex == ""; v += 2 {
		candidate := color.RGBA{v, v, v / 2, 0xff}
		if theming.ContrastRatio(candidate, normalText) >= NormalRatio && theming.ContrastRatio(candidate, highText) < HighRatio {
			c, hex = candidate, theming.FormatHex(candidate)
		}
	}
	if hex == "" {
		t.Fatal("test setup: no color is readable in normal contrast and not in high")
	}
	s := withColors("dark", map[string]string{"selection": hex})
	if got := ThemeWith(s).Color(core.Selection); got != c {
		t.Fatalf("normal contrast accepts it: %v", got)
	}
	s.Contrast = "high"
	if got := ThemeWith(s).Color(core.Selection); got == c {
		t.Fatal("high contrast must refuse it and keep the strengthened default")
	}
}

func TestUnknownThemeIDFallsBackLikeThemeFor(t *testing.T) {
	if ThemeIDOf("neon") != "dark" || ThemeIDOf("light") != "light" {
		t.Fatalf("ThemeIDOf = %q %q", ThemeIDOf("neon"), ThemeIDOf("light"))
	}
	s := withColors("dark", map[string]string{"accent": "#aaccee"})
	s.Theme = "neon" // resolves to dark, so dark's overrides apply
	if got := ThemeWith(s).Color(core.Accent); got != (color.RGBA{0xaa, 0xcc, 0xee, 0xff}) {
		t.Fatalf("accent = %v", got)
	}
}

func TestRequiredRatio(t *testing.T) {
	if RequiredRatio("normal") != NormalRatio || RequiredRatio("high") != HighRatio || RequiredRatio("") != NormalRatio || RequiredRatio("x") != NormalRatio {
		t.Fatal("normal unless high")
	}
}

func TestCustomColorLookup(t *testing.T) {
	for _, c := range CustomColors {
		got, ok := CustomColorOf(c.Token)
		if !ok || got.Token != c.Token {
			t.Errorf("CustomColorOf(%s) = %v %v", c.Token, got, ok)
		}
	}
	if _, ok := CustomColorOf(core.Text); ok {
		t.Fatal("text is not customizable")
	}
}
