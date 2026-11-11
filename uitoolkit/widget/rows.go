package widget

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// ListRow is a row of a list: the selected one has the selection
// background, the others a tint under the pointer. Its content and options
// (H, PadX, OnClick) are the caller's.
func ListRow(selected bool, items ...Item) *El {
	look := HoverBg(core.Hover)
	if selected {
		look = Bg(core.Selected)
	}
	return Row(append([]Item{Control("row"), look}, items...)...)
}

// Note is a one-line message under a control, in small text of the given
// color (core.Error for a problem, core.Muted for a hint). It is NoteH high
// and cuts a long text with an ellipsis.
func Note(text string, tone core.Token, opts ...Option) *El {
	return Row(with([]Item{
		H(core.NoteH),
		Text(text, Small(), Color(tone), Ellipsis(), Grow(1)),
	}, opts)...)
}

// formLabelGap is the space between a form row's label and its controls.
const formLabelGap = 12

// FormRow is a settings-style row: a label in a column of labelWidth, then
// the controls. Rows with the same labelWidth line their controls up.
func FormRow(label string, labelWidth int, items ...Item) *El {
	return Row(append([]Item{H(core.RowH), Gap(formLabelGap), Text(label, W(labelWidth))}, items...)...)
}

// CheckRow is a checkbox followed by its items (usually a Text label and a
// muted hint). Give it OnClick: the whole row is the click target.
func CheckRow(on bool, items ...Item) *El {
	return Row(append([]Item{H(core.RowH), Gap(8), Checkbox(on)}, items...)...)
}

// Toolbar is a bar of controls over a pane: ToolbarH high, inset by the
// gutter, centered vertically, with a line under it.
func Toolbar(items ...Item) *El {
	return Row(append([]Item{
		H(core.ToolbarH), PadX(core.Gutter), Gap(6), AlignItems(AlignCenter), BorderB(core.Border),
	}, items...)...)
}

// TabBar is the strip that holds Tabs or ScrollTabs (and anything beside
// them): TabBarH high, TabGap of room above the tabs, a line under the
// inactive ones.
func TabBar(items ...Item) *El {
	return Row(append([]Item{H(core.TabBarH), PadT(core.TabGap), BorderB(core.Border)}, items...)...)
}
