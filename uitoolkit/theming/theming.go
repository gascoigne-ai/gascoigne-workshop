// Package theming turns a user's appearance preferences (which theme,
// text size, corner roundness, contrast, color overrides) into a
// core.Theme, and loads extra themes from a folder of theme files.
//
// It is the mechanics only. An application decides which themes exist
// (a Catalog of built-in choices), keeps its own settings and maps them
// to Prefs; the toolkit's dark and light palettes are Builtins.
// Runnable examples: package uitoolkit/examples.
package theming

import (
	"fmt"
	"image/color"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// ThemeChoice is one selectable theme.
type ThemeChoice struct {
	ID, Label string
	Build     func() core.Theme
}

// SizeChoice is a text size: the font sizes of the UI, small and mono faces.
type SizeChoice struct {
	ID, Label         string
	Font, Small, Mono float64
}

// sizeChoices are the text sizes; the second one (medium) is the default.
var sizeChoices = []SizeChoice{
	{"small", "Small", 13, 11, 12},
	{"medium", "Medium", 14, 12, 13},
	{"large", "Large", 16, 13, 15},
}

// SizeChoices are the text sizes; the second one (medium) is the default.
// The slice is a copy: changing it changes nothing here.
func SizeChoices() []SizeChoice { return slices.Clone(sizeChoices) }

// CornerChoice is a roundness: the factor applied to every control's radius.
type CornerChoice struct {
	ID, Label string
	Scale     float64
}

// cornerChoices are the roundness levels; "soft" (the theme's own radii) is the default.
var cornerChoices = []CornerChoice{
	{"square", "Square", 0},
	{"soft", "Soft", 1},
	{"round", "Round", 1.5},
}

// CornerChoices are the roundness levels; "soft" (the theme's own radii)
// is the default. The slice is a copy.
func CornerChoices() []CornerChoice { return slices.Clone(cornerChoices) }

// ContrastChoice is a contrast level.
type ContrastChoice struct{ ID, Label string }

// contrastChoices are "normal" and "high"; high strengthens the
// customizable colors until their text reaches HighRatio.
var contrastChoices = []ContrastChoice{{"normal", "Normal"}, {"high", "High"}}

// ContrastChoices are "normal" and "high". The slice is a copy.
func ContrastChoices() []ContrastChoice { return slices.Clone(contrastChoices) }

// Contrast ratios the customizable colors must reach against their text.
const (
	NormalRatio = 4.5
	HighRatio   = 7.0
)

// CustomColor is a color users can change, and the text that has to stay
// readable on it.
type CustomColor struct {
	Token core.Token
	Label string
	Text  core.Token // the text color drawn on it
}

// customColors are the tokens users may override.
var customColors = []CustomColor{
	{core.Selection, "Selection", core.Text},
	{core.FindMatch, "Find match", core.Text},
	{core.FindActive, "Active match", core.Text},
	{core.Accent, "Accent / focus", core.AccentText},
}

// CustomColors are the tokens users may override. The slice is a copy.
func CustomColors() []CustomColor { return slices.Clone(customColors) }

// CustomColorOf finds the customizable color of a token.
func CustomColorOf(tok core.Token) (CustomColor, bool) {
	for _, c := range customColors {
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
	for _, c := range cornerChoices {
		if c.ID == id {
			return c.Scale
		}
	}
	return 1
}

// CornerIDOf is the corner id in effect: unknown ids are soft.
func CornerIDOf(id string) string {
	for _, c := range cornerChoices {
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
func Builtins() []ThemeChoice {
	return []ThemeChoice{
		{"dark", "Dark", Dark},
		{"light", "Light", Light},
	}
}

// Catalog is the set of themes an application offers: its built-in
// choices, the first of which is the fallback, then the user's.
type Catalog struct {
	Builtin []ThemeChoice
	user    map[string]ThemeChoice
}

// NewCatalog makes a catalog of the given built-in choices.
func NewCatalog(builtin []ThemeChoice) *Catalog {
	return &Catalog{Builtin: builtin, user: map[string]ThemeChoice{}}
}

// SetUser sets the user's themes, as LoadUser returned them.
func (c *Catalog) SetUser(themes map[string]ThemeChoice) {
	if themes == nil {
		themes = map[string]ThemeChoice{}
	}
	c.user = themes
}

// All is the built-in themes followed by the user's, sorted by id.
func (c *Catalog) All() []ThemeChoice {
	out := append([]ThemeChoice(nil), c.Builtin...)
	ids := slices.Sorted(maps.Keys(c.user))
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
func (c *Catalog) builtinNamed(id string) (core.Theme, bool) {
	for _, ch := range c.Builtin {
		if ch.ID == id {
			return ch.Build(), true
		}
	}
	return core.Theme{}, false
}

// For builds the theme for a theme id and a text size id. Unknown ids
// fall back to the first built-in theme and the medium text size.
func (c *Catalog) For(themeID, textSizeID string) core.Theme {
	th := c.Builtin[0].Build()
	for _, ch := range c.All() {
		if ch.ID == themeID {
			th = ch.Build()
		}
	}
	size := sizeChoices[1]
	for _, s := range sizeChoices {
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
func (c *Catalog) Theme(prefs Prefs) core.Theme {
	id := c.ThemeID(prefs.Theme)
	th := c.For(id, prefs.TextSize)
	th.RadiusScale, th.NoGradients = CornerScale(prefs.Corners), prefs.Flat
	if prefs.Contrast == "high" {
		strong := map[core.Token]color.RGBA{}
		for _, cc := range customColors {
			strong[cc.Token] = core.EnsureContrast(th.Color(cc.Token), th.Color(cc.Text), HighRatio)
		}
		th = th.With(strong)
		th = th.With(strengthened(th))
	}
	need := RequiredRatio(prefs.Contrast)
	overrides := map[core.Token]color.RGBA{}
	for name, hex := range prefs.Colors[id] {
		cc, ok := CustomColorOf(core.Token(name))
		if !ok {
			continue
		}
		col, err := core.ParseHex(hex)
		if err != nil || core.ContrastRatio(col, th.Color(cc.Text)) < need {
			continue
		}
		overrides[cc.Token] = col
	}
	return th.With(overrides)
}

// highBar is one color high contrast strengthens: it must reach ratio
// against each surface it is drawn on.
type highBar struct {
	token    core.Token
	ratio    float64
	surfaces []core.Token
}

var (
	textSurfaces = []core.Token{core.Bg, core.Sunken, core.Frame}
	listSurface  = []core.Token{core.Sunken}
	paneSurface  = []core.Token{core.Bg}
)

// highBars are the colors high contrast strengthens besides the customizable
// ones. They are set well above WCAG's 4.5:1 and 3:1 so the switch is plainly
// visible in both palettes, which already meet those: 17:1 for the main text,
// 12:1 for the muted text, 8:1 for the status colors, 6:1 for the accent
// line, 7:1 for borders and rules, and for the fills that show state (the
// selected and hovered row, a chip) a clear step away from their surface.
var highBars = []highBar{
	{core.Text, 17, textSurfaces},
	{core.Muted, 12, textSurfaces},
	{core.Error, 8, textSurfaces},
	{core.OK, 8, textSurfaces},
	{core.AccentLine, 6, textSurfaces},
	{core.Border, 7, textSurfaces},
	{core.Rule, 7, textSurfaces},
	{core.BorderSoft, 3, textSurfaces},
	{core.Selected, 2.2, listSurface},
	{core.Hover, 1.6, listSurface},
	{core.Chip, 2, paneSurface},
}

// strengthened is the colors of th moved, in their own hue, until they reach
// their high-contrast ratio on every surface. A color that already reaches it
// stays as it is.
func strengthened(th core.Theme) map[core.Token]color.RGBA {
	out := map[core.Token]color.RGBA{}
	for _, bar := range highBars {
		col := th.Color(bar.token)
		for pass := 0; pass < 2; pass++ { // moving away from one surface can leave another short
			for _, surface := range bar.surfaces {
				col = core.EnsureContrast(col, th.Color(surface), bar.ratio)
			}
		}
		out[bar.token] = col
	}
	return out
}

// LoadUser reads every *.json file in dir as a theme laid over the
// built-in theme it extends (the first built-in when it says nothing). A
// file that does not parse is skipped and described in problems; a
// missing folder is no problem at all.
func (c *Catalog) LoadUser(dir string) (themes map[string]ThemeChoice, problems []string) {
	themes = map[string]ThemeChoice{}
	if dir == "" {
		return themes, nil
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		slog.Debug("skip user themes: folder name is not a valid glob pattern", "dir", dir, "err", err)
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
		baseID, err := themeExtends(data)
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
		th, err := ParseTheme(data, &base)
		if err != nil {
			problems = append(problems, fmt.Sprintf("theme %s: %v", id, err))
			continue
		}
		label := th.Name
		if label == "" || label == base.Name {
			label = id
		}
		built := th
		themes[id] = ThemeChoice{ID: id, Label: label, Build: func() core.Theme { return built }}
	}
	return themes, problems
}
