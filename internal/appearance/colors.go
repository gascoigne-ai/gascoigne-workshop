package appearance

import (
	"image/color"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// CustomColor is a color users can change, and the text that has to
// stay readable on it.
type CustomColor struct {
	Token widget.Token
	Label string
	Text  widget.Token // the text color drawn on it
}

var CustomColors = []CustomColor{
	{widget.Selection, "Selection", widget.Text},
	{widget.FindMatch, "Find match", widget.Text},
	{widget.FindActive, "Active match", widget.Text},
	{widget.Accent, "Accent / focus", widget.AccentText},
}

// Contrast ratios the colors must reach against their text.
const (
	NormalRatio = 4.5
	HighRatio   = 7.0
)

type ContrastChoice struct{ ID, Label string }

var ContrastChoices = []ContrastChoice{{"normal", "Normal"}, {"high", "High"}}

func CustomColorOf(tok widget.Token) (CustomColor, bool) {
	for _, c := range CustomColors {
		if c.Token == tok {
			return c, true
		}
	}
	return CustomColor{}, false
}

// RequiredRatio is the contrast a custom color must reach: normal
// unless the contrast setting is high.
func RequiredRatio(contrast string) float64 {
	if contrast == "high" {
		return HighRatio
	}
	return NormalRatio
}

// ThemeIDOf is the id of the theme a settings value resolves to:
// unknown ids fall back to the first (dark), as ThemeFor does.
func ThemeIDOf(id string) string {
	for _, c := range AllThemeChoices() {
		if c.ID == id {
			return id
		}
	}
	return ThemeChoices[0].ID
}

// ThemeWith is the theme for the settings: the chosen theme and text
// size, strengthened colors when contrast is high, and the user's color
// overrides for that theme. An override that does not parse, is not a
// customizable color or would make its text unreadable is ignored.
func ThemeWith(s settings.Settings) widget.Theme {
	id := ThemeIDOf(s.Theme)
	th := ThemeFor(id, s.TextSize)
	th.RadiusScale, th.NoGradients = CornerScale(s.Corners), s.Flat
	if s.Contrast == "high" {
		strong := map[widget.Token]color.RGBA{}
		for _, c := range CustomColors {
			strong[c.Token] = widget.EnsureContrast(th.Color(c.Token), th.Color(c.Text), HighRatio)
		}
		th = th.With(strong)
	}
	need := RequiredRatio(s.Contrast)
	overrides := map[widget.Token]color.RGBA{}
	for name, hex := range s.Colors[id] {
		c, ok := CustomColorOf(widget.Token(name))
		if !ok {
			continue
		}
		col, err := widget.ParseHex(hex)
		if err != nil || widget.ContrastRatio(col, th.Color(c.Text)) < need {
			continue
		}
		overrides[c.Token] = col
	}
	return th.With(overrides)
}
