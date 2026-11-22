package theme

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/theming"
	"image/color"
	"math"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/format"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// jsonKinds are the kinds of span a JSON view colors.
var jsonKinds = []format.Kind{format.Plain, format.Punct, format.Key, format.String, format.Number, format.Literal}

// jsonToken is the color token of a kind of JSON span: the name the kernel
// gives its style. The response view takes its tokens from the same name, so
// no feature is imported to check them.
func jsonToken(kind format.Kind) core.Token { return core.Token(kind.Style()) }

func luminance(rgba color.RGBA) float64 {
	lin := func(value uint8) float64 {
		f := float64(value) / 255
		if f <= 0.03928 {
			return f / 12.92
		}
		return math.Pow((f+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(rgba.R) + 0.7152*lin(rgba.G) + 0.0722*lin(rgba.B)
}

func contrast(rgba, other color.RGBA) float64 {
	la, lb := luminance(rgba), luminance(other)
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
		for _, kind := range jsonKinds {
			if got := contrast(th.Color(jsonToken(kind)), bg); got < 3.5 {
				t.Errorf("%s: json kind %d contrast %.2f", tc.ID, kind, got)
			}
		}
	}
}

func TestEveryJSONKindHasAToken(t *testing.T) {
	for _, k := range jsonKinds {
		if jsonToken(k) == "" {
			t.Errorf("no token for json kind %d", k)
		}
	}
}

func TestThemeFor(t *testing.T) {
	tests := []struct {
		name, theme string
		size        int
		wantBg      color.RGBA
		wantFont    float64
	}{
		{"dark default", "dark", 14, theming.Dark().Color(core.Bg), 14},
		{"light sixteen", "light", 16, theming.Light().Color(core.Bg), 16},
		{"unknown theme falls back to dark", "neon", 13, theming.Dark().Color(core.Bg), 13},
		{"zero size is the default", "light", 0, theming.Light().Color(core.Bg), 14},
	}
	for _, tt := range tests {
		th := ThemeFor(tt.theme, FontSizes{UI: tt.size})
		if th.Color(core.Bg) != tt.wantBg || th.FontSize != tt.wantFont {
			t.Errorf("%s: bg %v font %v", tt.name, th.Color(core.Bg), th.FontSize)
		}
	}
}

func TestUnknownMethodFallsBackToTheTextColor(t *testing.T) {
	th := ThemeFor("dark", FontSizes{})
	if th.Color(MethodToken("PROPFIND")) != th.Color(core.Text) {
		t.Fatal("unknown method must resolve to the text color")
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
		th := ThemeFor(id, FontSizes{})
		if th.Color(MethodToken("GET")) == th.Color(core.Text) || th.Color(jsonToken(format.Key)) == (color.RGBA{}) {
			t.Errorf("%s: app colors missing", id)
		}
		if th.Style("primary").Radius <= 0 {
			t.Errorf("%s: toolkit control styles lost under the app overlay", id)
		}
	}
}

func TestCollectionTokensCycleAndEveryThemeDefinesThem(t *testing.T) {
	if CollectionToken(0) == CollectionToken(1) {
		t.Fatal("neighbouring collections must differ")
	}
	if CollectionToken(0) != CollectionToken(CollectionColors) {
		t.Fatal("the colors repeat after CollectionColors collections")
	}
	for name, theme := range map[string]core.Theme{"dark": DarkTheme(), "light": LightTheme()} {
		for i := 0; i < CollectionColors; i++ {
			if _, ok := theme.Colors[CollectionToken(i)]; !ok {
				t.Fatalf("%s theme has no color for %s", name, CollectionToken(i))
			}
		}
	}
}

func TestThemeWithFollowsTheThreeFontSizes(t *testing.T) {
	for _, test := range []struct{ ui, small, code int }{{10, 8, 8}, {14, 12, 13}, {16, 24, 32}, {24, 8, 8}} {
		prefs := settings.Default()
		prefs.FontSize, prefs.SmallFontSize, prefs.CodeFontSize = test.ui, test.small, test.code
		th := ThemeWith(prefs)
		if th.FontSize != float64(test.ui) || th.SmallSize != float64(test.small) || th.MonoSize != float64(test.code) {
			t.Errorf("ThemeWith(%+v) sizes = %v, %v, %v", test, th.FontSize, th.SmallSize, th.MonoSize)
		}
	}
}

// ThemeWith(Default()) is the theme every default screen has.
func TestThemeWithDefaultsHasTheDefaultSizes(t *testing.T) {
	th := ThemeWith(settings.Default())
	if th.FontSize != 14 || th.SmallSize != 12 || th.MonoSize != 13 {
		t.Fatalf("sizes = %v, %v, %v; want 14, 12, 13", th.FontSize, th.SmallSize, th.MonoSize)
	}
}

// settings may not import the toolkit, so both state the ranges; this keeps
// the two from drifting apart.
func TestFontSizeRangeMatchesToolkit(t *testing.T) {
	faces := []struct {
		name                     string
		settingsMin, settingsMax int
		settingsDefault          int
		toolkitMin, toolkitMax   int
		toolkitDefault           int
	}{
		{"interface", settings.MinFontSize, settings.MaxFontSize, settings.DefaultFontSize, theming.MinFontSize, theming.MaxFontSize, theming.DefaultFontSize},
		{"small", settings.MinSmallFontSize, settings.MaxSmallFontSize, settings.DefaultSmallFontSize, theming.MinSmallSize, theming.MaxSmallSize, theming.DefaultSmallSize},
		{"code", settings.MinCodeFontSize, settings.MaxCodeFontSize, settings.DefaultCodeFontSize, theming.MinCodeSize, theming.MaxCodeSize, theming.DefaultCodeSize},
	}
	for _, face := range faces {
		if face.settingsMin != face.toolkitMin || face.settingsMax != face.toolkitMax || face.settingsDefault != face.toolkitDefault {
			t.Errorf("%s: settings %d..%d (default %d) and theming %d..%d (default %d) disagree", face.name,
				face.settingsMin, face.settingsMax, face.settingsDefault, face.toolkitMin, face.toolkitMax, face.toolkitDefault)
		}
	}
}
