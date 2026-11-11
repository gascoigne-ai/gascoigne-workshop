package theming

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"image/color"
	"sort"
	"strings"
)

// core.Theme files are JSON in the style of editor themes: named variables
// that colors refer to as "$name", a flat token-to-color table, and a
// table of per-control styles (corner radius, gradient, outline).
//
//	{
//	  "name": "dark",
//	  "variables": { "rose": "#e88aa6" },
//	  "colors":    { "accent": "$rose", "shadow": "#00000066" },
//	  "controls":  { "button": { "radius": 6, "gradient": 0.08, "border": "border" },
//	                 "hero":   { "gradient": ["#4f1428", "#2a0d18"] } }
//	}
//
// A file may leave out whatever a base theme already defines; colors and
// controls are merged over it.

type controlFile struct {
	Radius   float64         `json:"radius"`
	Gradient json.RawMessage `json:"gradient"`
	Border   string          `json:"border"`
}

type themeFile struct {
	Name      string  `json:"name"`
	Extends   string  `json:"extends"`
	FontSize  float64 `json:"fontSize"`
	SmallSize float64 `json:"smallSize"`
	MonoSize  float64 `json:"monoSize"`
	// IconStroke multiplies the stroke width of every icon: 0.8 for thin
	// icons, 1.3 for bold ones; absent or 0 keeps the base theme's.
	IconStroke float64                `json:"iconStroke"`
	Variables  map[string]string      `json:"variables"`
	Colors     map[string]string      `json:"colors"`
	Controls   map[string]controlFile `json:"controls"`
}

// themeExtends is the name of the built-in theme a file says it builds on
// ("extends"), "" when it does not say. The caller picks the base theme
// and passes it to ParseTheme.
func themeExtends(data []byte) (string, error) {
	var f struct {
		Extends string `json:"extends"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return "", fmt.Errorf("theme file: %w", err)
	}
	return f.Extends, nil
}

// parseColor reads "#rrggbb" or "#rrggbbaa".
func parseColor(s string) (color.RGBA, error) {
	if len(s) == 9 {
		c, err := core.ParseHex(s[:7])
		if err != nil {
			return c, err
		}
		var a uint8
		if _, err := fmt.Sscanf(s[7:], "%02x", &a); err != nil {
			return c, fmt.Errorf("%q is not a color", s)
		}
		c.A = a
		return c, nil
	}
	return core.ParseHex(s)
}

// ParseTheme reads a theme file over base (nil for none). Unknown fields
// are errors, so a typo in a theme file is not silently ignored.
func ParseTheme(data []byte, base *core.Theme) (core.Theme, error) {
	var f themeFile
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return core.Theme{}, fmt.Errorf("theme file: %w", err)
	}
	var th core.Theme
	if base != nil {
		th = base.With(nil)
		th.Controls = copyControls(base.Controls)
	} else {
		th = core.Theme{Colors: map[core.Token]color.RGBA{}, Controls: map[string]core.ControlStyle{}, RadiusScale: 1}
	}
	if th.Controls == nil {
		th.Controls = map[string]core.ControlStyle{}
	}
	if th.RadiusScale == 0 {
		th.RadiusScale = 1
	}
	if f.Name != "" {
		th.Name = f.Name
	}
	if f.FontSize > 0 {
		th.FontSize = f.FontSize
	}
	if f.SmallSize > 0 {
		th.SmallSize = f.SmallSize
	}
	if f.IconStroke > 0 {
		th.IconStroke = f.IconStroke
	}
	if f.MonoSize > 0 {
		th.MonoSize = f.MonoSize
	}

	vars := map[string]color.RGBA{}
	for _, name := range sortedKeys(f.Variables) {
		c, err := resolveColor(f.Variables[name], vars)
		if err != nil {
			return core.Theme{}, fmt.Errorf("theme file: variable %q: %w", name, err)
		}
		vars[name] = c
	}
	for _, name := range sortedKeys(f.Colors) {
		c, err := resolveColor(f.Colors[name], vars)
		if err != nil {
			return core.Theme{}, fmt.Errorf("theme file: color %q: %w", name, err)
		}
		th.Colors[core.Token(name)] = c
	}
	for _, name := range sortedControlKeys(f.Controls) {
		cf := f.Controls[name]
		st := core.ControlStyle{Radius: cf.Radius, Border: core.Token(cf.Border)}
		if st.Radius < 0 {
			return core.Theme{}, fmt.Errorf("theme file: control %q: negative radius", name)
		}
		if err := parseGradient(cf.Gradient, vars, &st); err != nil {
			return core.Theme{}, fmt.Errorf("theme file: control %q: %w", name, err)
		}
		if st.Border != "" {
			if _, ok := th.Colors[st.Border]; !ok {
				return core.Theme{}, fmt.Errorf("theme file: control %q: border token %q has no color", name, st.Border)
			}
		}
		th.Controls[name] = st
	}
	return th, nil
}

// resolveColor is a "$variable" or a literal color.
func resolveColor(s string, vars map[string]color.RGBA) (color.RGBA, error) {
	if name, ok := strings.CutPrefix(s, "$"); ok {
		c, ok := vars[name]
		if !ok {
			return color.RGBA{}, fmt.Errorf("unknown variable $%s", name)
		}
		return c, nil
	}
	return parseColor(s)
}

// parseGradient reads a control's gradient: absent or 0 for none, a
// number for a shade strength, or [top, bottom] colors.
func parseGradient(raw json.RawMessage, vars map[string]color.RGBA, st *core.ControlStyle) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var strength float64
	if err := json.Unmarshal(raw, &strength); err == nil {
		if strength < 0 || strength > 0.5 {
			return errors.New("gradient strength must be within 0 and 0.5")
		}
		st.Shade = strength
		return nil
	}
	var ends []string
	if err := json.Unmarshal(raw, &ends); err != nil || len(ends) != 2 {
		return errors.New("gradient must be a number or [top, bottom]")
	}
	top, err := resolveColor(ends[0], vars)
	if err != nil {
		return err
	}
	bottom, err := resolveColor(ends[1], vars)
	if err != nil {
		return err
	}
	st.Top, st.Bottom = &top, &bottom
	return nil
}

func copyControls(m map[string]core.ControlStyle) map[string]core.ControlStyle {
	out := make(map[string]core.ControlStyle, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedControlKeys(m map[string]controlFile) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

//go:embed themes/*.json
var builtinThemes embed.FS

func builtin(name string) core.Theme {
	data, err := builtinThemes.ReadFile("themes/" + name + ".json")
	if err != nil {
		panic(err)
	}
	th, err := ParseTheme(data, nil)
	if err != nil {
		panic(fmt.Sprintf("built-in theme %s: %v", name, err))
	}
	return th
}

// Dark is warm near-black surfaces with a rose accent. The palette and
// control styles live in themes/dark.json.
func Dark() core.Theme { return builtin("dark") }

// Light is cream surfaces, white lists and a burgundy accent
// (themes/light.json).
func Light() core.Theme { return builtin("light") }
