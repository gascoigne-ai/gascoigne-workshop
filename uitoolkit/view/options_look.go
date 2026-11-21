package view

import (
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Bg fills the element with a theme color.
func Bg(token core.Token) Option { return func(element *El) { element.bg = token } }

// Border outlines the element.
func Border(token core.Token) Option { return func(element *El) { element.border = token } }

// BorderT draws a hairline along the top edge.
func BorderT(token core.Token) Option { return func(element *El) { element.borderT = token } }

// BorderL draws a one-pixel line down the element's left edge, to
// separate neighbours such as tabs or table columns; BorderInset shortens it.
func BorderL(token core.Token) Option { return func(element *El) { element.borderL = token } }

// Separator draws the soft hairline between neighbours (tabs, table
// columns) on the element's left edge; inset keeps it off the top and
// bottom. The first of a row has none: When(i > 0, Separator(10)); a
// template writes separator="10" on the elements after the first.
func Separator(inset int) Option {
	return func(element *El) { element.borderL, element.borderInset, element.insetOf = core.BorderSoft, inset, nil }
}

// SeparatorOf is Separator with an inset that follows the context's Sizes,
// as HOf does for a height.
func SeparatorOf(inset func(sizes core.Sizes) int) Option {
	return func(element *El) {
		element.borderL, element.insetOf = core.BorderSoft, inset
		element.borderInset = inset(core.DefaultSizes())
	}
}

// BorderInset keeps the BorderL and BorderR lines this many pixels from the top and bottom.
func BorderInset(pixels int) Option {
	return func(element *El) { element.borderInset, element.insetOf = pixels, nil }
}

// BorderB draws a hairline along the bottom edge, under the
// background and below the children.
func BorderB(token core.Token) Option { return func(element *El) { element.borderB = token } }

// BorderR draws a hairline along the right edge.
func BorderR(token core.Token) Option { return func(element *El) { element.borderR = token } }

// BorderX draws hairlines along the left and right edges.
func BorderX(token core.Token) Option {
	return func(element *El) { element.borderL, element.borderR = token, token }
}

// BorderY draws hairlines along the top and bottom edges.
func BorderY(token core.Token) Option {
	return func(element *El) { element.borderT, element.borderB = token, token }
}

// Control names the theme style that draws this element: its corner
// radius, gradient fill and outline come from Theme.Controls[name], so
// every control of a kind looks the same and the theme file decides how.
func Control(name string) Option { return func(element *El) { element.control = name } }

// EdgeT, EdgeL, EdgeR and EdgeB outline the element on that side with a
// two-pixel line of the token's color; the first three make the look of the
// open tab, which runs into the pane below it. The color of the last one wins.
func EdgeT(token core.Token) Option {
	return func(element *El) { element.edgeTok, element.edgeTop = token, true }
}

// EdgeL outlines the left side (see EdgeT).
func EdgeL(token core.Token) Option {
	return func(element *El) { element.edgeTok, element.edgeLeft = token, true }
}

// EdgeR outlines the right side (see EdgeT).
func EdgeR(token core.Token) Option {
	return func(element *El) { element.edgeTok, element.edgeRight = token, true }
}

// EdgeB outlines the bottom side (see EdgeT).
func EdgeB(token core.Token) Option {
	return func(element *El) { element.edgeTok, element.edgeBottom = token, true }
}

// Tooltip shows text near the pointer once it has rested on the element
// for a moment. The text is only recorded in the context (TipText): the
// window shows it by mounting widget.TooltipLayer last in its Stack.
func Tooltip(content string) Option { return func(element *El) { element.tip = content } }

// HoverBg fills the element while the pointer is over it.
func HoverBg(token core.Token) Option { return func(element *El) { element.hoverBg = token } }

// FontSource draws a Text in the font of source, at the size the text's other
// options choose (Small, Bold), instead of the interface font. Use it to show
// a font in its own face, as a font list does. A nil source keeps the
// interface font. In a template, font-source names the source in the scope.
func FontSource(source *text.GoTextFaceSource) Option {
	return func(element *El) { element.fontSource = source }
}

// HoverColor is the color an icon takes while the pointer is over it, in
// place of the hover token of the icon itself. Use it where that token
// does not contrast with the surface behind the icon, such as on the
// banner.
func HoverColor(token core.Token) Option { return func(element *El) { element.hoverColor = token } }

// Lighten brightens the background while the pointer is over it.
func Lighten() Option { return func(element *El) { element.lighten = true } }

// Shadow draws a soft offset shadow behind the element.
func Shadow() Option { return func(element *El) { element.shadow = true } }

// Clip cuts the children off at the element's edge.
func Clip() Option { return func(element *El) { element.clip = true } }
