package view

import "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"

// dialogLook is what makes a panel a dialog: a floating, modal menu-style
// panel in the middle of the window that owns the keyboard focus.
var dialogLook = []Item{Bg(core.Bg), Control("menu"), Shadow(), FloatCenter(), FocusScope(), Modal()}

// DialogBox is a modal panel centered in the window over a dimmed scrim. Give
// it its size (W, H), layout options and content, and OnDismiss for what
// an outside click does. Mount it in the window's Stack while it is open.
func DialogBox(items ...Item) *El {
	return Frag(Row(Bg(core.Shadow)), Col(append(append([]Item{}, dialogLook...), items...)...))
}

// ResizableDialogBox is a DialogBox the user can resize between the given
// bounds; state keeps the size across frames.
func ResizableDialogBox(state *ResizeState, minW, minH, maxW, maxH int, items ...Item) *El {
	return Frag(Row(Bg(core.Shadow)), Resizable(state, minW, minH, maxW, maxH, append(append([]Item{}, dialogLook...), items...)...))
}

// DialogTitle is the header of a dialog: the title, the items you give
// (a search field, buttons) and a close button at the end. OnDismiss is
// what the close button does.
func DialogTitle(title string, items ...Item) *El {
	var bar *El
	closeButton := Row(W(32), H(core.CompactH), Bg(core.Sunken), Control("icon-button"), Lighten(), Tooltip("Close (Esc)"),
		OnClick(func() {
			if bar.onDismiss != nil {
				bar.onDismiss()
			}
		}),
		CrossIcon(Grow(1)),
	)
	bar = Row(append(append([]Item{
		H(core.DialogTitleH), PadX(16), Gap(8), AlignItems(AlignCenter), BorderB(core.BorderSoft), Text(title, Bold()),
	}, items...), closeButton)...)
	return bar
}
