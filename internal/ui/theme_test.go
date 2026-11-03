package ui

import (
	"image/color"
	"math"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/jsonview"
	"github.com/gascoigne-ai/gascoigne-workshop/widget"
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

// Method labels and JSON colors are drawn on the list and code
// background; they must be readable in every theme.
func TestAppColorsAreReadableOnLists(t *testing.T) {
	for _, tc := range themeChoices {
		th := tc.build()
		bg := th.Color(widget.Sunken)
		for _, m := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
			if got := contrast(th.Color(methodToken(m)), bg); got < 3.5 {
				t.Errorf("%s: method %s contrast %.2f", tc.id, m, got)
			}
		}
		for kind, tok := range jsonTokens {
			if got := contrast(th.Color(tok), bg); got < 3.5 {
				t.Errorf("%s: json kind %d contrast %.2f", tc.id, kind, got)
			}
		}
	}
}

func TestEveryJSONKindHasAToken(t *testing.T) {
	kinds := []jsonview.Kind{jsonview.Plain, jsonview.Punct, jsonview.Key, jsonview.String, jsonview.Number, jsonview.Literal}
	for _, k := range kinds {
		if jsonTokens[k] == "" {
			t.Errorf("no token for json kind %d", k)
		}
	}
}

func TestThemeFor(t *testing.T) {
	tests := []struct {
		name, theme, size string
		wantBg            color.RGBA
		wantFont          float64
	}{
		{"dark medium", "dark", "medium", widget.Dark().Color(widget.Bg), 14},
		{"light large", "light", "large", widget.Light().Color(widget.Bg), 16},
		{"unknown theme falls back to dark", "neon", "small", widget.Dark().Color(widget.Bg), 12},
		{"unknown size falls back to medium", "light", "huge", widget.Light().Color(widget.Bg), 14},
	}
	for _, tt := range tests {
		th := ThemeFor(tt.theme, tt.size)
		if th.Color(widget.Bg) != tt.wantBg || th.FontSize != tt.wantFont {
			t.Errorf("%s: bg %v font %v", tt.name, th.Color(widget.Bg), th.FontSize)
		}
	}
}

func TestUnknownMethodFallsBackToTheTextColor(t *testing.T) {
	th := ThemeFor("dark", "medium")
	if th.Color(methodToken("PROPFIND")) != th.Color(widget.Text) {
		t.Fatal("unknown method must resolve to the text color")
	}
}

func TestCodeLinesKeepsStructureAndColors(t *testing.T) {
	lines := codeLines([]byte(`{"a":1}`))
	if len(lines) != 3 || !lines[0].Foldable || lines[1].Foldable {
		t.Fatalf("lines = %+v", lines)
	}
	var styles []widget.Token
	for _, s := range lines[1].Spans {
		styles = append(styles, s.Style)
	}
	if styles[1] != jsonTokens[jsonview.Key] || styles[len(styles)-1] != jsonTokens[jsonview.Number] {
		t.Fatalf("styles = %v", styles)
	}
}
