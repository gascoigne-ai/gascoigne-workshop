package appearance

import (
	"embed"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/theming"
)

// App-specific color tokens, added to the widget theme.
const MethodTokenPrefix = "method."

// MethodToken is the color token of an HTTP method; unknown methods
// resolve to the text color through Theme.Color's fallback.
func MethodToken(method string) core.Token {
	if method == "" {
		return core.Muted
	}
	return core.Token(MethodTokenPrefix + method)
}

//go:embed themes/*.json
var themeFiles embed.FS

// appTheme is a toolkit theme with the app's own colors (HTTP methods,
// JSON syntax) from themes/<id>.json laid over it. The toolkit's palettes
// and control styles are in uitoolkit/core/themes; both are plain theme files.
func appTheme(id string, base core.Theme) core.Theme {
	data, err := themeFiles.ReadFile("themes/" + id + ".json")
	if err != nil {
		panic(err)
	}
	th, err := theming.ParseTheme(data, &base)
	if err != nil {
		panic("app theme " + id + ": " + err.Error())
	}
	return th
}

func DarkTheme() core.Theme  { return appTheme("dark", theming.Dark()) }
func LightTheme() core.Theme { return appTheme("light", theming.Light()) }

// The mechanics (choices, text sizes, corners, contrast, color overrides,
// the user's theme folder) are the toolkit's, in uitoolkit/theming. This
// package supplies the app's own themes and maps its settings onto them.
type (
	ThemeChoice    = theming.Choice
	SizeChoice     = theming.SizeChoice
	CornerChoice   = theming.CornerChoice
	ContrastChoice = theming.ContrastChoice
	CustomColor    = theming.CustomColor
)

const (
	NormalRatio = theming.NormalRatio
	HighRatio   = theming.HighRatio
)

var (
	TextSizeChoices = theming.SizeChoices
	CornerChoices   = theming.CornerChoices
	ContrastChoices = theming.ContrastChoices
	CustomColors    = theming.CustomColors
	CustomColorOf   = theming.CustomColorOf
	RequiredRatio   = theming.RequiredRatio
	CornerScale     = theming.CornerScale
	CornerIDOf      = theming.CornerIDOf
)

// ThemeChoices are the app's built-in themes.
var ThemeChoices = []ThemeChoice{
	{ID: "dark", Label: "Dark", Build: DarkTheme},
	{ID: "light", Label: "Light", Build: LightTheme},
}

var catalog = theming.NewCatalog(ThemeChoices)

// The interface scale slider's range and step, in percent.
const (
	ScaleMin  = 50
	ScaleMax  = 200
	ScaleStep = 5
)

// AllThemeChoices are the built-in themes followed by the user's, sorted.
func AllThemeChoices() []ThemeChoice { return catalog.All() }

// ThemeIDOf is the id of the theme a settings value resolves to.
func ThemeIDOf(id string) string { return catalog.ThemeID(id) }

// ThemeFor builds the theme for a theme id and a text size id.
func ThemeFor(themeID, textSizeID string) core.Theme { return catalog.For(themeID, textSizeID) }

// ThemeWith is the theme for the settings.
func ThemeWith(s settings.Settings) core.Theme {
	return catalog.Theme(theming.Prefs{Theme: s.Theme, TextSize: s.TextSize, Corners: s.Corners,
		Contrast: s.Contrast, Flat: s.Flat, Colors: s.Colors})
}

// LoadUserThemes reads the user's theme files in dir (see theming.Catalog.LoadUser).
func LoadUserThemes(dir string) (map[string]ThemeChoice, []string) { return catalog.LoadUser(dir) }

// SetUserThemes sets the user's themes, as LoadUserThemes returned them.
func SetUserThemes(themes map[string]ThemeChoice) { catalog.SetUser(themes) }

// LayoutChoice is an entry of the layout chooser in the settings menu.
type LayoutChoice struct{ ID, Label string }

var LayoutChoices = []LayoutChoice{
	{"stacked", "Stacked"},
	{"side", "Side by side"},
}
