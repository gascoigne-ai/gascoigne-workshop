// Package theme is the app's part of theming: its overlay themes
// (HTTP-method, JSON and collection colors), the color tokens for them
// and the mapping from settings to a toolkit theme.
//
// It sits in the ui zone, which may import the toolkit, so importing
// uitoolkit/core and uitoolkit/theming (core.Theme, core.Token: theme
// data is toolkit-bound) needs no written exception. The user rejected
// putting it in uitoolkit/theming (the toolkit would learn the app's
// colors). Revisit when the toolkit-free data (tokens as strings) can be
// split from the build.
package theme

import (
	"embed"
	"strconv"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
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

// CollectionColors is how many collection colors there are; more open
// collections reuse them in turn.
const CollectionColors = 6

// CollectionToken is the color that marks the collection at index.
func CollectionToken(index int) core.Token {
	return core.Token("collection." + strconv.Itoa(index%CollectionColors))
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
	ThemeChoice    = theming.ThemeChoice
	CornerChoice   = theming.CornerChoice
	ContrastChoice = theming.ContrastChoice
	CustomColor    = theming.CustomColor
	FontSizes      = theming.FontSizes
)

const (
	NormalRatio = theming.NormalRatio
	HighRatio   = theming.HighRatio
)

var (
	CornerChoices   = theming.CornerChoices()
	ContrastChoices = theming.ContrastChoices()
	CustomColors    = theming.CustomColors()
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

// AllThemeChoices are the built-in themes followed by the user's, sorted.
func AllThemeChoices() []ThemeChoice { return catalog.All() }

// ThemeIDOf is the id of the theme a settings value resolves to.
func ThemeIDOf(id string) string { return catalog.ThemeID(id) }

// ThemeFor builds the theme for a theme id and the three face sizes in pixels.
func ThemeFor(themeID string, sizes FontSizes) core.Theme { return catalog.For(themeID, sizes) }

// ThemeWith is the theme for the settings.
func ThemeWith(prefs settings.Settings) core.Theme {
	return catalog.Theme(theming.Prefs{Theme: prefs.Theme, Fonts: FontSizes{UI: prefs.FontSize, Small: prefs.SmallFontSize, Code: prefs.CodeFontSize}, Corners: prefs.Corners,
		Contrast: prefs.Contrast, Flat: prefs.Flat, Colors: prefs.Colors})
}

// LoadUserThemes reads the user's theme files in dir (see theming.Catalog.LoadUser).
func LoadUserThemes(dir string) (map[string]ThemeChoice, []string) { return catalog.LoadUser(dir) }

// SetUserThemes sets the user's themes, as LoadUserThemes returned them.
func SetUserThemes(themes map[string]ThemeChoice) { catalog.SetUser(themes) }
