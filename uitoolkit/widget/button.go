package widget

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// buttonPadX is the inset between the edge of a button and its label.
const buttonPadX = 10

// Button is an outlined chip with a centered label; it brightens on hover.
// A W is a minimum: the button widens to show its label when the row has
// room, so a label is cut with an ellipsis only when there is none.
func Button(label string, opts ...Option) *El {
	return Row(With([]Item{
		Bg(core.Sunken), Control("button"), Lighten(), PadX(buttonPadX),
		Text(label, TextCenter(), Ellipsis(), Grow(1), Color(core.ChipText)),
	}, fitted(opts))...)
}

// LinkButton is a quiet text button: accent-colored text on no
// background, lit softly under the pointer. It sizes to its label.
func LinkButton(label string, opts ...Option) *El {
	return Row(With([]Item{
		PadX(core.LooseGap), HoverBg(core.Hover),
		Text(label, TextCenter(), Color(core.AccentLine)),
	}, opts)...)
}

// PrimaryButton is the accent-colored call-to-action variant of Button.
func PrimaryButton(label string, opts ...Option) *El {
	return Row(With([]Item{
		Bg(core.Accent), Control("primary"), Lighten(), PadX(buttonPadX),
		Text(label, TextCenter(), Ellipsis(), Grow(1), Color(core.AccentText)),
	}, fitted(opts))...)
}

// Choice is one chip of a segmented choice: a PrimaryButton when it is the
// selected one, a Button otherwise. Options (W, H, OnClick, KeepInView)
// apply to both.
func Choice(label string, selected bool, opts ...Option) *El {
	if selected {
		return PrimaryButton(label, opts...)
	}
	return Button(label, opts...)
}

// IconButton is Button with an icon in place of the label.
func IconButton(glyph *El, opts ...Option) *El {
	return Row(With([]Item{Bg(core.Sunken), Control("icon-button"), Lighten(), glyph}, opts)...)
}

// fitted is opts followed by FitW, so a button's W is its minimum.
func fitted(opts []Option) []Option {
	return append(append([]Option(nil), opts...), FitW())
}
