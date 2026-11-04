package ui

import (
	"image/color"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/ui/widget"
)

// customColor is a color users can change, and the text that has to
// stay readable on it.
type customColor struct {
	token widget.Token
	label string
	text  widget.Token // the text color drawn on it
}

var customColors = []customColor{
	{widget.Selection, "Selection", widget.Text},
	{widget.FindMatch, "Find match", widget.Text},
	{widget.FindActive, "Active match", widget.Text},
	{widget.Accent, "Accent / focus", widget.AccentText},
}

// Contrast ratios the colors must reach against their text.
const (
	normalRatio = 4.5
	highRatio   = 7.0
)

type contrastChoice struct{ id, label string }

var contrastChoices = []contrastChoice{{"normal", "Normal"}, {"high", "High"}}

func customColorOf(tok widget.Token) (customColor, bool) {
	for _, c := range customColors {
		if c.token == tok {
			return c, true
		}
	}
	return customColor{}, false
}

// requiredRatio is the contrast a custom color must reach: normal
// unless the contrast setting is high.
func requiredRatio(contrast string) float64 {
	if contrast == "high" {
		return highRatio
	}
	return normalRatio
}

// themeIDOf is the id of the theme a settings value resolves to:
// unknown ids fall back to the first (dark), as ThemeFor does.
func themeIDOf(id string) string {
	for _, c := range themeChoices {
		if c.id == id {
			return id
		}
	}
	return themeChoices[0].id
}

// ThemeWith is the theme for the settings: the chosen theme and text
// size, strengthened colors when contrast is high, and the user's color
// overrides for that theme. An override that does not parse, is not a
// customizable color or would make its text unreadable is ignored.
func ThemeWith(s settings.Settings) widget.Theme {
	id := themeIDOf(s.Theme)
	th := ThemeFor(id, s.TextSize)
	if s.Contrast == "high" {
		strong := map[widget.Token]color.RGBA{}
		for _, c := range customColors {
			strong[c.token] = widget.EnsureContrast(th.Color(c.token), th.Color(c.text), highRatio)
		}
		th = th.With(strong)
	}
	need := requiredRatio(s.Contrast)
	overrides := map[widget.Token]color.RGBA{}
	for name, hex := range s.Colors[id] {
		c, ok := customColorOf(widget.Token(name))
		if !ok {
			continue
		}
		col, err := widget.ParseHex(hex)
		if err != nil || widget.ContrastRatio(col, th.Color(c.text)) < need {
			continue
		}
		overrides[c.token] = col
	}
	return th.With(overrides)
}
