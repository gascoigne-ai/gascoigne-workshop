package appearance

import (
	"embed"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// App-specific color tokens, added to the widget theme.
const MethodTokenPrefix = "method."

// MethodToken is the color token of an HTTP method; unknown methods
// resolve to the text color through Theme.Color's fallback.
func MethodToken(method string) widget.Token {
	if method == "" {
		return widget.Muted
	}
	return widget.Token(MethodTokenPrefix + method)
}

//go:embed themes/*.json
var themeFiles embed.FS

// appTheme is a toolkit theme with the app's own colors (HTTP methods,
// JSON syntax) from themes/<id>.json laid over it. The toolkit's palettes
// and control styles are in uitoolkit/widget/themes; both are plain theme files.
func appTheme(id string, base widget.Theme) widget.Theme {
	data, err := themeFiles.ReadFile("themes/" + id + ".json")
	if err != nil {
		panic(err)
	}
	th, err := widget.ParseTheme(data, &base)
	if err != nil {
		panic("app theme " + id + ": " + err.Error())
	}
	return th
}

func DarkTheme() widget.Theme  { return appTheme("dark", widget.Dark()) }
func LightTheme() widget.Theme { return appTheme("light", widget.Light()) }

// ThemeChoice and SizeChoice are the entries of the settings menu.
type ThemeChoice struct {
	ID, Label string
	Build     func() widget.Theme
}

type SizeChoice struct {
	ID, Label         string
	Font, Small, Mono float64
}

var ThemeChoices = []ThemeChoice{
	{"dark", "Dark", DarkTheme},
	{"light", "Light", LightTheme},
}

var TextSizeChoices = []SizeChoice{
	{"small", "Small", 13, 11, 12},
	{"medium", "Medium", 14, 12, 13},
	{"large", "Large", 16, 13, 15},
}

// The interface scale slider's range and step, in percent.
const (
	ScaleMin  = 50
	ScaleMax  = 200
	ScaleStep = 5
)

// CornerChoices are the entries of the corner setting: the id in the
// settings file, its label, and the factor on every control's radius.
// CornerChoice is one entry of CornerChoices.
type CornerChoice struct {
	ID, Label string
	Scale     float64
}

var CornerChoices = []CornerChoice{
	{"square", "Square", 0},
	{"soft", "Soft", 1},
	{"round", "Round", 1.5},
}

// CornerScale is the radius factor of a corner id; unknown ids are soft.
func CornerScale(id string) float64 {
	for _, c := range CornerChoices {
		if c.ID == id {
			return c.Scale
		}
	}
	return 1
}

// ThemeFor builds the theme for a settings pair. Unknown ids fall back
// to the first (dark) theme and the medium text size.
func ThemeFor(themeID, textSizeID string) widget.Theme {
	th := ThemeChoices[0].Build()
	for _, c := range AllThemeChoices() {
		if c.ID == themeID {
			th = c.Build()
		}
	}
	size := TextSizeChoices[1]
	for _, c := range TextSizeChoices {
		if c.ID == textSizeID {
			size = c
		}
	}
	return th.WithTextSize(size.Font, size.Small, size.Mono)
}

// LayoutChoice is an entry of the layout chooser in the settings menu.
type LayoutChoice struct{ ID, Label string }

var LayoutChoices = []LayoutChoice{
	{"side", "Side by side"},
	{"stacked", "Stacked"},
}

// CornerIDOf is the corner id in effect: unknown ids are soft.
func CornerIDOf(id string) string {
	for _, c := range CornerChoices {
		if c.ID == id {
			return id
		}
	}
	return "soft"
}
