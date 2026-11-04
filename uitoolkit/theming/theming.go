// Package theming turns a user's appearance preferences (which theme,
// text size, corner roundness, contrast, color overrides) into a
// widget.Theme, and loads extra themes from a folder of theme files.
//
// It is the mechanics only. An application decides which themes exist
// (a Catalog of built-in choices), keeps its own settings and maps them
// to Prefs; the toolkit's dark and light palettes are Builtins.
package theming

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Choice is one selectable theme.
type Choice struct {
	ID, Label string
	Build     func() widget.Theme
}

// SizeChoice is a text size: the font sizes of the UI, small and mono faces.
type SizeChoice struct {
	ID, Label         string
	Font, Small, Mono float64
}

// SizeChoices are the text sizes; the second one (medium) is the default.
var SizeChoices = []SizeChoice{
	{"small", "Small", 13, 11, 12},
	{"medium", "Medium", 14, 12, 13},
	{"large", "Large", 16, 13, 15},
}

// CornerChoice is a roundness: the factor applied to every control's radius.
type CornerChoice struct {
	ID, Label string
	Scale     float64
}

// CornerChoices are the roundness levels; "soft" (the theme's own radii) is the default.
var CornerChoices = []CornerChoice{
	{"square", "Square", 0},
	{"soft", "Soft", 1},
	{"round", "Round", 1.5},
}

// ContrastChoice is a contrast level.
type ContrastChoice struct{ ID, Label string }

// ContrastChoices are "normal" and "high"; high strengthens the
// customizable colors until their text reaches HighRatio.
var ContrastChoices = []ContrastChoice{{"normal", "Normal"}, {"high", "High"}}

// Contrast ratios the customizable colors must reach against their text.
const (
	NormalRatio = 4.5
	HighRatio   = 7.0
)

// CustomColor is a color users can change, and the text that has to stay
// readable on it.
type CustomColor struct {
	Token widget.Token
	Label string
	Text  widget.Token // the text color drawn on it
}

// CustomColors are the tokens users may override.
var CustomColors = []CustomColor{
	{widget.Selection, "Selection", widget.Text},
	{widget.FindMatch, "Find match", widget.Text},
	{widget.FindActive, "Active match", widget.Text},
	{widget.Accent, "Accent / focus", widget.AccentText},
}

// CustomColorOf finds the customizable color of a token.
func CustomColorOf(tok widget.Token) (CustomColor, bool) {
	for _, c := range CustomColors {
		if c.Token == tok {
			return c, true
		}
	}
	return CustomColor{}, false
}

// RequiredRatio is the contrast a custom color must reach: normal
// unless the contrast is high.
func RequiredRatio(contrast string) float64 {
	if contrast == "high" {
		return HighRatio
	}
	return NormalRatio
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

// CornerIDOf is the corner id in effect: unknown ids are soft.
func CornerIDOf(id string) string {
	for _, c := range CornerChoices {
		if c.ID == id {
			return id
		}
	}
	return "soft"
}

// Prefs are the appearance preferences, as an application stores them.
type Prefs struct {
	Theme, TextSize, Corners, Contrast string
	Flat                               bool                         // no gradients
	Colors                             map[string]map[string]string // theme id, token name, "#rrggbb"
}

// Builtins are the toolkit's own dark and light themes.
func Builtins() []Choice {
	return []Choice{
		{"dark", "Dark", widget.Dark},
		{"light", "Light", widget.Light},
	}
}

// Catalog is the set of themes an application offers: its built-in
// choices, the first of which is the fallback, then the user's.
type Catalog struct {
	Builtin []Choice
	user    map[string]Choice
}

// NewCatalog makes a catalog of the given built-in choices.
func NewCatalog(builtin []Choice) *Catalog {
	return &Catalog{Builtin: builtin, user: map[string]Choice{}}
}

// SetUser sets the user's themes, as LoadUser returned them.
func (c *Catalog) SetUser(themes map[string]Choice) {
	if themes == nil {
		themes = map[string]Choice{}
	}
	c.user = themes
}

// All is the built-in themes followed by the user's, sorted by id.
func (c *Catalog) All() []Choice {
	out := append([]Choice(nil), c.Builtin...)
	ids := make([]string, 0, len(c.user))
	for id := range c.user {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		out = append(out, c.user[id])
	}
	return out
}

// ThemeID is the id a preference resolves to: unknown ids fall back to
// the first built-in theme.
func (c *Catalog) ThemeID(id string) string {
	for _, ch := range c.All() {
		if ch.ID == id {
			return id
		}
	}
	return c.Builtin[0].ID
}

// builtinNamed is the built-in theme a theme file may extend.
func (c *Catalog) builtinNamed(id string) (widget.Theme, bool) {
	for _, ch := range c.Builtin {
		if ch.ID == id {
			return ch.Build(), true
		}
	}
	return widget.Theme{}, false
}

// For builds the theme for a theme id and a text size id. Unknown ids
// fall back to the first built-in theme and the medium text size.
func (c *Catalog) For(themeID, textSizeID string) widget.Theme {
	th := c.Builtin[0].Build()
	for _, ch := range c.All() {
		if ch.ID == themeID {
			th = ch.Build()
		}
	}
	size := SizeChoices[1]
	for _, s := range SizeChoices {
		if s.ID == textSizeID {
			size = s
		}
	}
	return th.WithTextSize(size.Font, size.Small, size.Mono)
}

// Theme is the theme for the preferences: the chosen theme and text size,
// the corner roundness and flat switch, high contrast (the customizable
// colors moved until their text reaches HighRatio), then the user's color
// overrides for that theme. An override is ignored unless it parses as a
// hex color, names a customizable token and reaches the required ratio.
func (c *Catalog) Theme(p Prefs) widget.Theme {
	id := c.ThemeID(p.Theme)
	th := c.For(id, p.TextSize)
	th.RadiusScale, th.NoGradients = CornerScale(p.Corners), p.Flat
	if p.Contrast == "high" {
		strong := map[widget.Token]color.RGBA{}
		for _, cc := range CustomColors {
			strong[cc.Token] = widget.EnsureContrast(th.Color(cc.Token), th.Color(cc.Text), HighRatio)
		}
		th = th.With(strong)
	}
	need := RequiredRatio(p.Contrast)
	overrides := map[widget.Token]color.RGBA{}
	for name, hex := range p.Colors[id] {
		cc, ok := CustomColorOf(widget.Token(name))
		if !ok {
			continue
		}
		col, err := widget.ParseHex(hex)
		if err != nil || widget.ContrastRatio(col, th.Color(cc.Text)) < need {
			continue
		}
		overrides[cc.Token] = col
	}
	return th.With(overrides)
}

// LoadUser reads every *.json file in dir as a theme laid over the
// built-in theme it extends (the first built-in when it says nothing). A
// file that does not parse is skipped and described in problems; a
// missing folder is no problem at all.
func (c *Catalog) LoadUser(dir string) (themes map[string]Choice, problems []string) {
	themes = map[string]Choice{}
	if dir == "" {
		return themes, nil
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return themes, nil
	}
	sort.Strings(files)
	for _, file := range files {
		id := strings.TrimSuffix(filepath.Base(file), ".json")
		data, err := os.ReadFile(file)
		if err != nil {
			problems = append(problems, fmt.Sprintf("theme %s: %v", id, err))
			continue
		}
		baseID, err := widget.ThemeExtends(data)
		if err != nil {
			problems = append(problems, fmt.Sprintf("theme %s: %v", id, err))
			continue
		}
		if baseID == "" {
			baseID = c.Builtin[0].ID
		}
		base, ok := c.builtinNamed(baseID)
		if !ok {
			problems = append(problems, fmt.Sprintf("theme %s extends unknown theme %q", id, baseID))
			continue
		}
		th, err := widget.ParseTheme(data, &base)
		if err != nil {
			problems = append(problems, fmt.Sprintf("theme %s: %v", id, err))
			continue
		}
		label := th.Name
		if label == "" || label == base.Name {
			label = id
		}
		built := th
		themes[id] = Choice{ID: id, Label: label, Build: func() widget.Theme { return built }}
	}
	return themes, problems
}
