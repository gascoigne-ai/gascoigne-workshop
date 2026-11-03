package widget

import "image/color"

// Token names a color role. A Theme maps tokens to colors, so switching
// theme recolors everything. Applications add their own tokens (for
// example "method.GET") with Theme.With.
type Token string

// Core tokens, from back to front: Frame is the window chrome, Bg the
// raised panes, Sunken lists and code, Alt zebra rows, Selected the
// current row.
const (
	Bg         Token = "bg"
	Frame      Token = "frame"
	Panel      Token = "panel"
	Sunken     Token = "sunken"
	Alt        Token = "alt"
	Border     Token = "border"
	Hover      Token = "hover"
	Selected   Token = "selected"
	Chip       Token = "chip"
	ChipText   Token = "chip-text"
	Rule       Token = "rule"
	Text       Token = "text"
	Muted      Token = "muted"
	Accent     Token = "accent"
	AccentLine Token = "accent-line" // the accent as a line: outlines, underlines, marks
	AccentText Token = "accent-text"
	OK         Token = "ok"
	Error      Token = "error"
	Shadow     Token = "shadow"
	Selection  Token = "selection"
	FindMatch  Token = "find-match"
	FindActive Token = "find-active"
)

// Fixed sizes shared by all widgets, in logical pixels.
const (
	RowH = 28
	Pad  = 12
)

// Theme holds the colors and font sizes the widgets draw with.
type Theme struct {
	Colors                        map[Token]color.RGBA
	FontSize, SmallSize, MonoSize float64
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

// Dark is warm near-black surfaces with a rose accent.
func Dark() Theme {
	return Theme{
		FontSize: 15, SmallSize: 12, MonoSize: 14,
		Colors: map[Token]color.RGBA{
			Bg: RGB(0x241b20), Frame: RGB(0x140f12), Panel: RGB(0x2d2228),
			Sunken: RGB(0x1c1519), Alt: RGB(0x251c21), Border: RGB(0x5f4d57),
			Hover: RGB(0x302429), Selected: RGB(0x4a2234),
			Chip: RGB(0x4a3a42), ChipText: RGB(0xf1e8eb), Rule: RGB(0x76626c),
			Text: RGB(0xf1e8eb), Muted: RGB(0xbfaab2),
			Accent: RGB(0xe88aa6), AccentLine: RGB(0xe88aa6), AccentText: RGB(0x2a0d18),
			OK: RGB(0x5ed39a), Error: RGB(0xff6b5e),
			Shadow: {A: 0x66}, Selection: RGB(0x8a4a66),
			FindMatch: RGB(0x4a3a1a), FindActive: RGB(0x8a5d12),
		},
	}
}

// Light is cream surfaces, white lists and a burgundy accent.
func Light() Theme {
	return Theme{
		FontSize: 15, SmallSize: 12, MonoSize: 14,
		Colors: map[Token]color.RGBA{
			Bg: RGB(0xf6f3f4), Frame: RGB(0xece5e8), Panel: RGB(0xf5f0f2),
			Sunken: RGB(0xffffff), Alt: RGB(0xf5f0f2), Border: RGB(0xa89aa1),
			Hover: RGB(0xece5e8), Selected: RGB(0xecd3dc),
			Chip: RGB(0xcfc2c8), ChipText: RGB(0x22161b), Rule: RGB(0x8c7b83),
			Text: RGB(0x22161b), Muted: RGB(0x66545b),
			Accent: RGB(0x7a1f3d), AccentLine: RGB(0x7a1f3d), AccentText: RGB(0xffffff),
			OK: RGB(0x17804a), Error: RGB(0xd1281b),
			Shadow: {A: 0x33}, Selection: RGB(0xf0c9d6),
			FindMatch: RGB(0xf6e3b5), FindActive: RGB(0xf0c26a),
		},
	}
}
