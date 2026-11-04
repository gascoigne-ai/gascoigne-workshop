package ui

import (
	"embed"
	"strings"
	"unicode/utf8"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// App-specific color tokens, added to the widget theme.
const methodTokenPrefix = "method."

// methodToken is the color token of an HTTP method; unknown methods
// resolve to the text color through Theme.Color's fallback.
func methodToken(method string) widget.Token {
	if method == "" {
		return widget.Muted
	}
	return widget.Token(methodTokenPrefix + method)
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

func darkTheme() widget.Theme  { return appTheme("dark", widget.Dark()) }
func lightTheme() widget.Theme { return appTheme("light", widget.Light()) }

// themeChoice and sizeChoice are the entries of the settings menu.
type themeChoice struct {
	id, label string
	build     func() widget.Theme
}

type sizeChoice struct {
	id, label         string
	font, small, mono float64
}

var themeChoices = []themeChoice{
	{"dark", "Dark", darkTheme},
	{"light", "Light", lightTheme},
}

var textSizeChoices = []sizeChoice{
	{"small", "Small", 13, 11, 12},
	{"medium", "Medium", 14, 12, 13},
	{"large", "Large", 16, 13, 15},
}

// The interface scale slider's range and step, in percent.
const (
	scaleMin  = 50
	scaleMax  = 200
	scaleStep = 5
)

// cornerChoices are the entries of the corner setting: the id in the
// settings file, its label, and the factor on every control's radius.
var cornerChoices = []struct {
	id, label string
	scale     float64
}{
	{"square", "Square", 0},
	{"soft", "Soft", 1},
	{"round", "Round", 1.5},
}

// cornerScale is the radius factor of a corner id; unknown ids are soft.
func cornerScale(id string) float64 {
	for _, c := range cornerChoices {
		if c.id == id {
			return c.scale
		}
	}
	return 1
}

// ThemeFor builds the theme for a settings pair. Unknown ids fall back
// to the first (dark) theme and the medium text size.
func ThemeFor(themeID, textSizeID string) widget.Theme {
	th := themeChoices[0].build()
	for _, c := range allThemeChoices() {
		if c.id == themeID {
			th = c.build()
		}
	}
	size := textSizeChoices[1]
	for _, c := range textSizeChoices {
		if c.id == textSizeID {
			size = c
		}
	}
	return th.WithTextSize(size.font, size.small, size.mono)
}

// kvRows turns key/value rows into table rows with two or three cells.
func kvRows(rows []model.KV, withDesc, flagDuplicates bool) []vml.TableRow {
	out := make([]vml.TableRow, len(rows))
	var dup []bool
	if flagDuplicates {
		dup = model.DuplicateNames(rows)
	}
	for i, kv := range rows {
		cells := []string{kv.Name, kv.Value}
		if kv.Secret {
			cells[1] = strings.Repeat("•", min(utf8.RuneCountInString(kv.Value), 12))
		}
		if withDesc {
			cells = append(cells, kv.Description)
		}
		out[i] = vml.TableRow{Cells: cells, Checked: kv.Enabled, Muted: !kv.Enabled, Flagged: dup != nil && dup[i]}
	}
	return out
}

// layoutChoice is an entry of the layout chooser in the settings menu.
type layoutChoice struct{ id, label string }

var layoutChoices = []layoutChoice{
	{"side", "Side by side"},
	{"stacked", "Stacked"},
}

// cornerIDOf is the corner id in effect: unknown ids are soft.
func cornerIDOf(id string) string {
	for _, c := range cornerChoices {
		if c.id == id {
			return id
		}
	}
	return "soft"
}
