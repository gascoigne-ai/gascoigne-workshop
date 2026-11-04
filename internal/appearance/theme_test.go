package appearance

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/respview"
	"image/color"
	"math"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/jsonview"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
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
	for _, tc := range ThemeChoices {
		th := tc.Build()
		bg := th.Color(core.Sunken)
		for _, m := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
			if got := contrast(th.Color(MethodToken(m)), bg); got < 3.5 {
				t.Errorf("%s: method %s contrast %.2f", tc.ID, m, got)
			}
		}
		for kind, tok := range respview.Tokens {
			if got := contrast(th.Color(tok), bg); got < 3.5 {
				t.Errorf("%s: json kind %d contrast %.2f", tc.ID, kind, got)
			}
		}
	}
}

func TestEveryJSONKindHasAToken(t *testing.T) {
	kinds := []jsonview.Kind{jsonview.Plain, jsonview.Punct, jsonview.Key, jsonview.String, jsonview.Number, jsonview.Literal}
	for _, k := range kinds {
		if respview.Tokens[k] == "" {
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
		{"dark medium", "dark", "medium", core.Dark().Color(core.Bg), 14},
		{"light large", "light", "large", core.Light().Color(core.Bg), 16},
		{"unknown theme falls back to dark", "neon", "small", core.Dark().Color(core.Bg), 13},
		{"unknown size falls back to medium", "light", "huge", core.Light().Color(core.Bg), 14},
	}
	for _, tt := range tests {
		th := ThemeFor(tt.theme, tt.size)
		if th.Color(core.Bg) != tt.wantBg || th.FontSize != tt.wantFont {
			t.Errorf("%s: bg %v font %v", tt.name, th.Color(core.Bg), th.FontSize)
		}
	}
}

func TestUnknownMethodFallsBackToTheTextColor(t *testing.T) {
	th := ThemeFor("dark", "medium")
	if th.Color(MethodToken("PROPFIND")) != th.Color(core.Text) {
		t.Fatal("unknown method must resolve to the text color")
	}
}

func TestCodeLinesKeepsStructureAndColors(t *testing.T) {
	lines := respview.CodeLines([]byte(`{"a":1}`))
	if len(lines) != 3 || !lines[0].Foldable || lines[1].Foldable {
		t.Fatalf("lines = %+v", lines)
	}
	var styles []core.Token
	for _, s := range lines[1].Spans {
		styles = append(styles, s.Style)
	}
	if styles[1] != respview.Tokens[jsonview.Key] || styles[len(styles)-1] != respview.Tokens[jsonview.Number] {
		t.Fatalf("styles = %v", styles)
	}
}

func TestMethodTokenForRequestsWithoutAMethod(t *testing.T) {
	if MethodToken("") != core.Muted {
		t.Fatal("a request without a method (WebSocket, MCP) uses the muted color")
	}
	if MethodToken("GET") != "method.GET" {
		t.Fatalf("MethodToken(GET) = %q", MethodToken("GET"))
	}
}

func TestThemeWithAppliesCornersAndTheFlatSwitch(t *testing.T) {
	s := settings.Default()
	th := ThemeWith(s)
	if th.RadiusScale != 1 || th.NoGradients || !th.Style("hero").HasGradient() || th.Style("button").Radius <= 0 {
		t.Fatalf("default look: scale %v flat %v %+v", th.RadiusScale, th.NoGradients, th.Style("button"))
	}
	s.Corners, s.Flat = "square", true
	th = ThemeWith(s)
	if th.Style("button").Radius != 0 || th.Style("hero").HasGradient() {
		t.Fatalf("square and flat: %+v", th.Style("button"))
	}
	s.Corners = "round"
	if ThemeWith(s).Style("button").Radius <= th.Style("button").Radius+1 && CornerScale("round") <= 1 {
		t.Fatal("round must be rounder than soft")
	}
	if CornerScale("nonsense") != 1 {
		t.Fatal("an unknown corner id is soft")
	}
}

func TestAppThemesCarryMethodAndJSONColorsOverTheToolkitStyles(t *testing.T) {
	for _, id := range []string{"dark", "light"} {
		th := ThemeFor(id, "medium")
		if th.Color(MethodToken("GET")) == th.Color(core.Text) || th.Color(respview.Tokens[jsonview.Key]) == (color.RGBA{}) {
			t.Errorf("%s: app colors missing", id)
		}
		if th.Style("primary").Radius <= 0 {
			t.Errorf("%s: toolkit control styles lost under the app overlay", id)
		}
	}
}
