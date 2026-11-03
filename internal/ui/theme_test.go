package ui

import (
	"image/color"
	"math"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/jsonview"
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

func contrast(a, b color.RGBA) float64 {
	la, lb := luminance(a), luminance(b)
	return (max(la, lb) + 0.05) / (min(la, lb) + 0.05)
}

// Text must stay readable on every surface it is drawn on, in each theme.
func TestThemesAreReadable(t *testing.T) {
	for _, tc := range themeChoices {
		th := tc.build()
		pairs := []struct {
			name   string
			fg, bg color.RGBA
			min    float64
		}{
			{"text on pane", th.Text, th.Bg, 4.5},
			{"text on list", th.Text, th.Sunken, 4.5},
			{"text on zebra row", th.Text, th.Alt, 4.5},
			{"text on selected row", th.Text, th.Selected, 4.5},
			{"text on frame", th.Text, th.Sidebar, 4.5},
			{"muted on pane", th.Muted, th.Bg, 3},
			{"muted on list", th.Muted, th.Sunken, 3},
			{"chip text on chip", th.ChipText, th.Chip, 4.5},
			{"accent text on accent", th.AccentText, th.Accent, 4.5},
		}
		for _, p := range pairs {
			if got := contrast(p.fg, p.bg); got < p.min {
				t.Errorf("%s: %s contrast %.2f < %.1f", tc.id, p.name, got, p.min)
			}
		}
		for kind, clr := range th.Syntax {
			if got := contrast(clr, th.Sunken); got < 3.5 {
				t.Errorf("%s: syntax kind %d contrast %.2f on code background", tc.id, kind, got)
			}
		}
		for m, clr := range th.Method {
			if got := contrast(clr, th.Sunken); got < 3.5 {
				t.Errorf("%s: method %s contrast %.2f on list", tc.id, m, got)
			}
		}
	}
}

func TestThemesDefineEveryKind(t *testing.T) {
	kinds := []jsonview.Kind{jsonview.Plain, jsonview.Punct, jsonview.Key, jsonview.String, jsonview.Number, jsonview.Literal}
	for _, tc := range themeChoices {
		th := tc.build()
		for _, k := range kinds {
			if _, ok := th.Syntax[k]; !ok {
				t.Errorf("%s: no color for kind %d", tc.id, k)
			}
		}
		for m := range DarkTheme().Method {
			if _, ok := th.Method[m]; !ok {
				t.Errorf("%s: no color for method %s", tc.id, m)
			}
		}
	}
}

func TestThemeFor(t *testing.T) {
	tests := []struct {
		name, theme, size string
		wantBg            color.RGBA
		wantFont          float64
	}{
		{"dark medium", "dark", "medium", DarkTheme().Bg, 14},
		{"light large", "light", "large", LightTheme().Bg, 16},
		{"unknown theme falls back to dark", "neon", "small", DarkTheme().Bg, 12},
		{"unknown size falls back to medium", "light", "huge", LightTheme().Bg, 14},
	}
	for _, tt := range tests {
		th := ThemeFor(tt.theme, tt.size)
		if th.Bg != tt.wantBg || th.FontSize != tt.wantFont {
			t.Errorf("%s: bg %v font %v", tt.name, th.Bg, th.FontSize)
		}
	}
}
