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
	AccentText Token = "accent-text"
	OK         Token = "ok"
	Error      Token = "error"
	Shadow     Token = "shadow"
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

const accentGold = 0xffc90e

// Dark is flat mid-gray panes over a darker frame, with a gold accent.
func Dark() Theme {
	return Theme{
		FontSize: 14, SmallSize: 11, MonoSize: 13,
		Colors: map[Token]color.RGBA{
			Bg: RGB(0x535353), Frame: RGB(0x2b2b2b), Panel: RGB(0x484848),
			Sunken: RGB(0x3b3b3b), Alt: RGB(0x363636), Border: RGB(0x3a3a3a),
			Hover: RGB(0x4a4a4a), Selected: RGB(0x1f1f1f),
			Chip: RGB(0x2e2e2e), ChipText: RGB(0xdcdcdc), Rule: RGB(0x8c8c8c),
			Text: RGB(0xdadada), Muted: RGB(0xb0b0b0),
			Accent: RGB(accentGold), AccentText: RGB(0x1a1a1a),
			OK: RGB(0x7fd18b), Error: RGB(0xff7b7b),
			Shadow: {A: 0x66},
		},
	}
}

// Light is the same structure on light grays.
func Light() Theme {
	return Theme{
		FontSize: 14, SmallSize: 11, MonoSize: 13,
		Colors: map[Token]color.RGBA{
			Bg: RGB(0xe9e9e9), Frame: RGB(0xcfcfcf), Panel: RGB(0xdedede),
			Sunken: RGB(0xf6f6f6), Alt: RGB(0xeeeeee), Border: RGB(0xbcbcbc),
			Hover: RGB(0xdcdcdc), Selected: RGB(0xc4c4c4),
			Chip: RGB(0xd2d2d2), ChipText: RGB(0x262626), Rule: RGB(0x8f8f8f),
			Text: RGB(0x262626), Muted: RGB(0x5f5f5f),
			Accent: RGB(accentGold), AccentText: RGB(0x1a1a1a),
			OK: RGB(0x1b7f45), Error: RGB(0xc02626),
			Shadow: {A: 0x33},
		},
	}
}
