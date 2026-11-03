// Package ui is the workshop window: theme, layout, widgets and the
// App that wires them to the model and protocols. Layout and widget
// state are plain values so they are tested without a GPU.
package ui

import (
	"image/color"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/jsonview"
)

// Theme holds every color and size the widgets use.
type Theme struct {
	Bg, Sidebar, Panel, Border, Hover, Selected color.RGBA
	Text, Muted, Accent, AccentText             color.RGBA
	OK, Error                                   color.RGBA

	// Method colors a request method label; Syntax colors JSON spans.
	Method map[string]color.RGBA
	Syntax map[jsonview.Kind]color.RGBA

	FontSize, SmallSize, MonoSize float64
	RowH, Pad                     int
}

func rgb(hex uint32) color.RGBA {
	return color.RGBA{R: uint8(hex >> 16), G: uint8(hex >> 8), B: uint8(hex), A: 0xff}
}

// DarkTheme is the default dark palette.
func DarkTheme() Theme {
	th := Theme{
		Bg:         rgb(0x1f1f1f),
		Sidebar:    rgb(0x1a1a1a),
		Panel:      rgb(0x242424),
		Border:     rgb(0x333333),
		Hover:      rgb(0x2a2a2a),
		Selected:   rgb(0x2e2e2e),
		Text:       rgb(0xcccccc),
		Muted:      rgb(0x858585),
		Accent:     rgb(0xefb74e),
		AccentText: rgb(0x1a1a1a),
		OK:         rgb(0x5ec98a),
		Error:      rgb(0xeb5757),
		Method: map[string]color.RGBA{
			"GET":    rgb(0x5ec98a),
			"POST":   rgb(0xd7ba7d),
			"PUT":    rgb(0x569cd6),
			"PATCH":  rgb(0xc586c0),
			"DELETE": rgb(0xf14c4c),
		},
		FontSize:  14,
		SmallSize: 11,
		MonoSize:  13,
		RowH:      28,
		Pad:       12,
	}
	th.Syntax = map[jsonview.Kind]color.RGBA{
		jsonview.Plain:   th.Text,
		jsonview.Punct:   th.Text,
		jsonview.Key:     rgb(0xe6e6e6),
		jsonview.String:  rgb(0xd9a566),
		jsonview.Number:  rgb(0xd68de0),
		jsonview.Literal: rgb(0xf07178),
	}
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
