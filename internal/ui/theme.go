// Package ui is the workshop window: theme, layout, widgets and the
// App that wires them to the model and protocols. Layout and widget
// state are plain values so they are tested without a GPU.
package ui

import (
	"image/color"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/jsonview"
)

// Theme holds every color and size the widgets use. Surfaces from
// back to front: Sidebar (window frame), Bg (raised panes), Sunken
// (lists, code), Alt (zebra rows), Selected (current row).
type Theme struct {
	Bg, Sidebar, Panel, Sunken, Alt, Border, Hover, Selected color.RGBA
	Chip, ChipText, Rule                                     color.RGBA
	Text, Muted, Accent, AccentText                          color.RGBA
	OK, Error                                                color.RGBA

	// Method colors a request method label; Syntax colors JSON spans.
	Method map[string]color.RGBA
	Syntax map[jsonview.Kind]color.RGBA

	FontSize, SmallSize, MonoSize float64
	RowH, Pad                     int
}

func rgb(hex uint32) color.RGBA {
	return color.RGBA{R: uint8(hex >> 16), G: uint8(hex >> 8), B: uint8(hex), A: 0xff}
}

// Sizes and the gold accent are shared by both themes.
const accentGold = 0xffc90e

func baseTheme() Theme {
	return Theme{
		Accent:     rgb(accentGold),
		AccentText: rgb(0x1a1a1a),
		FontSize:   14,
		SmallSize:  11,
		MonoSize:   13,
		RowH:       28,
		Pad:        12,
	}
}

// DarkTheme is flat mid-gray panes over a darker frame.
func DarkTheme() Theme {
	th := baseTheme()
	th.Bg, th.Sidebar, th.Panel = rgb(0x535353), rgb(0x2b2b2b), rgb(0x484848)
	th.Sunken, th.Alt, th.Border = rgb(0x3b3b3b), rgb(0x363636), rgb(0x3a3a3a)
	th.Hover, th.Selected = rgb(0x4a4a4a), rgb(0x1f1f1f)
	th.Chip, th.ChipText, th.Rule = rgb(0x2e2e2e), rgb(0xdcdcdc), rgb(0x8c8c8c)
	th.Text, th.Muted = rgb(0xdadada), rgb(0xb0b0b0)
	th.OK, th.Error = rgb(0x7fd18b), rgb(0xff7b7b)
	th.Method = map[string]color.RGBA{
		"GET": rgb(0x7fd18b), "POST": rgb(0xe6c36a), "PUT": rgb(0x7db4ea),
		"PATCH": rgb(0xd4a0e0), "DELETE": rgb(0xff7b7b),
	}
	th.Syntax = syntax(th.Text, rgb(0xf0f0f0), rgb(0xe8b070), rgb(0xe0a0ea), rgb(0xff8a8a))
	return th
}

// LightTheme is the same structure on light grays.
func LightTheme() Theme {
	th := baseTheme()
	th.Bg, th.Sidebar, th.Panel = rgb(0xe9e9e9), rgb(0xcfcfcf), rgb(0xdedede)
	th.Sunken, th.Alt, th.Border = rgb(0xf6f6f6), rgb(0xeeeeee), rgb(0xbcbcbc)
	th.Hover, th.Selected = rgb(0xdcdcdc), rgb(0xc4c4c4)
	th.Chip, th.ChipText, th.Rule = rgb(0xd2d2d2), rgb(0x262626), rgb(0x8f8f8f)
	th.Text, th.Muted = rgb(0x262626), rgb(0x5f5f5f)
	th.OK, th.Error = rgb(0x1b7f45), rgb(0xc02626)
	th.Method = map[string]color.RGBA{
		"GET": rgb(0x1b7f45), "POST": rgb(0x9a6a00), "PUT": rgb(0x1f64b0),
		"PATCH": rgb(0x8a3fa0), "DELETE": rgb(0xc02626),
	}
	th.Syntax = syntax(th.Text, rgb(0x111111), rgb(0x9c4a00), rgb(0x7a2e92), rgb(0xb02020))
	return th
}

func syntax(text, key, str, num, lit color.RGBA) map[jsonview.Kind]color.RGBA {
	return map[jsonview.Kind]color.RGBA{
		jsonview.Plain: text, jsonview.Punct: text, jsonview.Key: key,
		jsonview.String: str, jsonview.Number: num, jsonview.Literal: lit,
	}
}

// themeChoices are the selectable themes, in menu order.
var themeChoices = []struct {
	id, label string
	build     func() Theme
}{
	{"dark", "Dark", DarkTheme},
	{"light", "Light", LightTheme},
}

// textSizeChoices are the selectable font sizes, in menu order.
var textSizeChoices = []struct {
	id, label         string
	font, small, mono float64
}{
	{"small", "Small", 12, 10, 12},
	{"medium", "Medium", 14, 11, 13},
	{"large", "Large", 16, 12, 15},
}

// ThemeFor builds the theme for a settings pair. Unknown ids fall back
// to the first (dark) theme and the medium text size.
func ThemeFor(themeID, textSizeID string) Theme {
	th := themeChoices[0].build()
	for _, c := range themeChoices {
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
	th.FontSize, th.SmallSize, th.MonoSize = size.font, size.small, size.mono
	return th
}

// MethodColor returns the color for method, or the muted color for
// methods the theme does not list.
func (t Theme) MethodColor(method string) color.RGBA {
	if c, ok := t.Method[method]; ok {
		return c
	}
	return t.Muted
}
