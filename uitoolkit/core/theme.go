package core

import "image/color"

// Token names a color role. A Theme maps tokens to colors, so switching
// theme recolors everything. Applications add their own tokens (for
// example "method.GET") with Theme.With.
type Token string

// Core tokens, from back to front: Frame is the window chrome, Bg the
// raised panes, Sunken lists and code, Alt zebra rows, Selected the
// current row.
const (
	Bg           Token = "bg"
	Frame        Token = "frame"
	Panel        Token = "panel"
	Sunken       Token = "sunken"
	Alt          Token = "alt"
	Border       Token = "border"
	Hover        Token = "hover"
	Selected     Token = "selected"
	Chip         Token = "chip"
	ChipText     Token = "chip-text"
	Rule         Token = "rule"
	Text         Token = "text"
	Muted        Token = "muted"
	Accent       Token = "accent"
	AccentLine   Token = "accent-line" // the accent as a line: outlines, underlines, marks
	AccentText   Token = "accent-text"
	OK           Token = "ok"
	Error        Token = "error"
	Shadow       Token = "shadow"
	Selection    Token = "selection"
	FindMatch    Token = "find-match"
	FindActive   Token = "find-active"
	BorderSoft   Token = "border-soft"   // hairlines between table rows, as the dashboard draws them
	BorderStrong Token = "border-strong" // control outlines (buttons, fields, menus)
	Info         Token = "info"
	InfoSoft     Token = "info-soft"
	Warn         Token = "warn"
	OKSoft       Token = "ok-soft" // pale backgrounds for status pills
	WarnSoft     Token = "warn-soft"
	ErrorSoft    Token = "error-soft"
	AccentSoft   Token = "accent-soft"
	Track        Token = "track"      // the empty part of a bar
	Faint        Token = "faint"      // the quietest text
	Hero         Token = "hero"       // the banner behind the window title, as a flat fallback
	HeroText     Token = "hero-text"  // text on the banner
	HeroMuted    Token = "hero-muted" // icons on the banner
)

// Size rules shared by every component, in logical pixels. Heights come
// in three steps so neighbours line up: RowH for list and table rows and
// standard controls, CompactH for chips, icon buttons and inline fields
// that sit in a row of that height, LabelH and NoteH for section captions
// and one-line notes. Icon buttons are IconBtnW wide. Gutter is the inset
// between a container and its content.
const (
	RowH     = 28
	CompactH = 24
	LabelH   = 22
	NoteH    = 18
	IconBtnW = 32
	Gutter   = 12
)

// Theme holds the colors and font sizes the widgets draw with.
type Theme struct {
	Colors                        map[Token]color.RGBA
	FontSize, SmallSize, MonoSize float64

	// Name is the theme's name, from its file.
	Name string

	// Controls are the per-control styles (corner radius, gradient,
	// outline) by control name; see ControlStyle and Theme.Style.
	Controls map[string]ControlStyle

	// RadiusScale multiplies every control's corner radius: 0 gives square
	// corners, 1 the theme's own.
	RadiusScale float64

	// NoGradients turns every gradient into a flat fill.
	NoGradients bool
}

// Color returns the color of tok, or the text color for unknown tokens
// so a typo stays visible but readable.
func (t Theme) Color(tok Token) color.RGBA {
	if c, ok := t.Colors[tok]; ok {
		return c
	}
	return t.Colors[Text]
}

// With returns a copy of t with extra tokens added or replaced.
func (t Theme) With(extra map[Token]color.RGBA) Theme {
	colors := make(map[Token]color.RGBA, len(t.Colors)+len(extra))
	for k, v := range t.Colors {
		colors[k] = v
	}
	for k, v := range extra {
		colors[k] = v
	}
	t.Colors = colors
	return t
}

// WithTextSize returns a copy of t with the given font sizes in pixels.
func (t Theme) WithTextSize(font, small, mono float64) Theme {
	t.FontSize, t.SmallSize, t.MonoSize = font, small, mono
	return t
}

// RGB converts 0xRRGGBB to a color.
func RGB(hex uint32) color.RGBA {
	return color.RGBA{R: uint8(hex >> 16), G: uint8(hex >> 8), B: uint8(hex), A: 0xff}
}

// Dark and Light follow the gascoigne CLI dashboard's palette
// (cmd/gascoigne/pages.css in gascoigne-cli): burgundy accent, warm
// near-black or cream surfaces. Its surface tokens map to Frame (the
// window chrome), Bg (raised panes) and Sunken (lists and code). Border,
// Rule, Chip, Hover and Selected keep the CLI's hues but are stronger
// than its soft web borders, so a native window's edges can be seen
// (see TestNonTextElementsAreVisible).

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
