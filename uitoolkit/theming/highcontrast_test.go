package theming

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// High contrast used to move only four customizable colors, which already
// met the bar in both palettes, so switching it on changed nothing anyone
// could see. It now also strengthens the text, the muted text, the status
// colors and the lines, each against the surfaces it is drawn on.
func TestHighContrastMeetsTheBarForTextLinesAndStatusColors(t *testing.T) {
	surfaces := []core.Token{core.Bg, core.Sunken, core.Frame}
	bars := []struct {
		what  string
		token core.Token
		min   float64
	}{
		{"text", core.Text, 17},
		{"muted text", core.Muted, 12},
		{"error", core.Error, 8},
		{"ok", core.OK, 8},
		{"accent line", core.AccentLine, 6},
		{"border", core.Border, 7},
		{"rule", core.Rule, 7},
		{"soft border", core.BorderSoft, 3},
	}
	c := NewCatalog(Builtins())
	for _, theme := range []string{"dark", "light"} {
		high := c.Theme(Prefs{Theme: theme, Contrast: "high"})
		for _, bar := range bars {
			for _, surface := range surfaces {
				if got := core.ContrastRatio(high.Color(bar.token), high.Color(surface)); got < bar.min-0.01 {
					t.Errorf("%s: %s on %s is %.2f:1 under high contrast, want at least %.1f:1", theme, bar.what, surface, got, bar.min)
				}
			}
		}
	}
}

func TestHighContrastChangesTheThemeAndNormalLeavesItAlone(t *testing.T) {
	c := NewCatalog(Builtins())
	for _, theme := range []string{"dark", "light"} {
		normal := c.Theme(Prefs{Theme: theme, Contrast: "normal"})
		base := c.For(theme, "medium")
		high := c.Theme(Prefs{Theme: theme, Contrast: "high"})
		changed := 0
		for _, token := range []core.Token{core.Text, core.Muted, core.Border, core.Rule, core.AccentLine, core.Error, core.OK} {
			if normal.Color(token) != base.Color(token) {
				t.Errorf("%s: normal contrast changed %s", theme, token)
			}
			if high.Color(token) != normal.Color(token) {
				changed++
			}
		}
		if changed < 3 {
			t.Errorf("%s: high contrast changed only %d of the main tokens; it must be visible", theme, changed)
		}
	}
}

func TestHighContrastKeepsTheHueOfWhatItStrengthens(t *testing.T) {
	c := NewCatalog(Builtins())
	normal, high := c.Theme(Prefs{Theme: "dark"}), c.Theme(Prefs{Theme: "dark", Contrast: "high"})
	n, h := normal.Color(core.Error), high.Color(core.Error)
	if !(n.R >= n.G && n.R >= n.B) || !(h.R >= h.G && h.R >= h.B) {
		t.Fatalf("the error color is no longer red-ish: %v -> %v", n, h)
	}
}

// The dark palette's text already sat at the bars, so lines and the fills
// that show state (hover, selection, chips) are what make the switch
// visible: they must be told apart from their surface much more clearly.
func TestHighContrastSeparatesStateFillsFromTheirSurface(t *testing.T) {
	c := NewCatalog(Builtins())
	bars := []struct {
		what    string
		token   core.Token
		surface core.Token
		min     float64
	}{
		{"selected row", core.Selected, core.Sunken, 2.2},
		{"hovered row", core.Hover, core.Sunken, 1.6},
		{"chip", core.Chip, core.Bg, 2},
	}
	for _, theme := range []string{"dark", "light"} {
		normal, high := c.Theme(Prefs{Theme: theme}), c.Theme(Prefs{Theme: theme, Contrast: "high"})
		for _, bar := range bars {
			got := core.ContrastRatio(high.Color(bar.token), high.Color(bar.surface))
			if got < bar.min-0.01 {
				t.Errorf("%s: %s on %s is %.2f:1 under high contrast, want at least %.1f:1", theme, bar.what, bar.surface, got, bar.min)
			}
			if was := core.ContrastRatio(normal.Color(bar.token), normal.Color(bar.surface)); got <= was {
				t.Errorf("%s: %s is no clearer than in normal contrast (%.2f -> %.2f)", theme, bar.what, was, got)
			}
		}
	}
}
