package core

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"sort"
	"strings"
)

// Theme files are JSON in the style of editor themes: named variables
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

// ControlStyle is how one kind of control is drawn. The zero value is a
// flat, square control with no outline of its own.
type ControlStyle struct {
	Radius float64 // corner radius in logical pixels; larger than half the height makes a pill
	Shade  float64 // gradient strength: the fill mixes toward white above and black below
	Top    *color.RGBA
	Bottom *color.RGBA // explicit gradient ends, replacing Shade
	Border Token       // outline color token, "" for none
}

// HasGradient reports whether the control's fill is a gradient.
func (s ControlStyle) HasGradient() bool { return s.Shade > 0 || s.Top != nil }

type controlFile struct {
	Radius   float64         `json:"radius"`
	Gradient json.RawMessage `json:"gradient"`
	Border   string          `json:"border"`
}

type themeFile struct {
	Name      string                 `json:"name"`
	Extends   string                 `json:"extends"`
	FontSize  float64                `json:"fontSize"`
	SmallSize float64                `json:"smallSize"`
	MonoSize  float64                `json:"monoSize"`
	Variables map[string]string      `json:"variables"`
	Colors    map[string]string      `json:"colors"`
	Controls  map[string]controlFile `json:"controls"`
}

// ThemeExtends is the name of the built-in theme a file says it builds on
// ("extends"), "" when it does not say. The caller picks the base theme
// and passes it to ParseTheme.
func ThemeExtends(data []byte) (string, error) {
	var f struct {
		Extends string `json:"extends"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return "", fmt.Errorf("theme file: %w", err)
	}
	return f.Extends, nil
}

// ParseColor reads "#rrggbb" or "#rrggbbaa".
func ParseColor(s string) (color.RGBA, error) {
	if len(s) == 9 {
		c, err := ParseHex(s[:7])
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
	return ParseHex(s)
}

// ParseTheme reads a theme file over base (nil for none). Unknown fields
// are errors, so a typo in a theme file is not silently ignored.
func ParseTheme(data []byte, base *Theme) (Theme, error) {
	var f themeFile
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return Theme{}, fmt.Errorf("theme file: %w", err)
	}
	var th Theme
	if base != nil {
		th = base.With(nil)
		th.Controls = copyControls(base.Controls)
	} else {
		th = Theme{Colors: map[Token]color.RGBA{}, Controls: map[string]ControlStyle{}, RadiusScale: 1}
	}
	if th.Controls == nil {
		th.Controls = map[string]ControlStyle{}
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
	if f.MonoSize > 0 {
		th.MonoSize = f.MonoSize
	}

	vars := map[string]color.RGBA{}
	for _, name := range sortedKeys(f.Variables) {
		c, err := resolveColor(f.Variables[name], vars)
		if err != nil {
			return Theme{}, fmt.Errorf("theme file: variable %q: %w", name, err)
		}
		vars[name] = c
	}
	for _, name := range sortedKeys(f.Colors) {
		c, err := resolveColor(f.Colors[name], vars)
		if err != nil {
			return Theme{}, fmt.Errorf("theme file: color %q: %w", name, err)
		}
		th.Colors[Token(name)] = c
	}
	for _, name := range sortedControlKeys(f.Controls) {
		cf := f.Controls[name]
		st := ControlStyle{Radius: cf.Radius, Border: Token(cf.Border)}
		if st.Radius < 0 {
			return Theme{}, fmt.Errorf("theme file: control %q: negative radius", name)
		}
		if err := parseGradient(cf.Gradient, vars, &st); err != nil {
			return Theme{}, fmt.Errorf("theme file: control %q: %w", name, err)
		}
		if st.Border != "" {
			if _, ok := th.Colors[st.Border]; !ok {
				return Theme{}, fmt.Errorf("theme file: control %q: border token %q has no color", name, st.Border)
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
	return ParseColor(s)
}

// parseGradient reads a control's gradient: absent or 0 for none, a
// number for a shade strength, or [top, bottom] colors.
func parseGradient(raw json.RawMessage, vars map[string]color.RGBA, st *ControlStyle) error {
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

func copyControls(m map[string]ControlStyle) map[string]ControlStyle {
	out := make(map[string]ControlStyle, len(m))
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

func builtin(name string) Theme {
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

// Style is the style of the named control with the theme's global
// switches applied: corners scaled by RadiusScale, and gradients dropped
// when NoGradients is set (the control then shows its own Bg color). A control the theme does not list is plain.
func (t Theme) Style(name string) ControlStyle {
	st := t.Controls[name]
	st.Radius *= t.RadiusScale
	if t.NoGradients {
		st.Shade = 0
		st.Top, st.Bottom = nil, nil
	}
	return st
}
