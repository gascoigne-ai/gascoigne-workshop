package widget

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"

	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// dialogLook is what makes a panel a dialog: a floating, modal menu-style
// panel in the middle of the window that owns the keyboard focus.
var dialogLook = []Item{Bg(core.Bg), Control("menu"), Shadow(), FloatCenter(), FocusScope(), Modal()}

// Dialog is a modal panel centered in the window over a dimmed scrim. Give
// it its size (W, H), layout options and content, and OnDismiss for what
// an outside click does. Mount it in the window's Stack while it is open.
func Dialog(items ...Item) *El {
	return Frag(Row(Bg(core.Shadow)), Col(append(append([]Item{}, dialogLook...), items...)...))
}

// ResizableDialog is a Dialog the user can resize between the given
// bounds; state keeps the size across frames.
func ResizableDialog(state *ResizeState, minW, minH, maxW, maxH int, items ...Item) *El {
	return Frag(Row(Bg(core.Shadow)), Resizable(state, minW, minH, maxW, maxH, append(append([]Item{}, dialogLook...), items...)...))
}

// DialogTitle is the title bar of a dialog, a bar of its own on the sunken
// color: the title, free room, any items you give and a close button at the
// end. The user drags the dialog by the bar (the room between the title and
// the buttons), so put controls in a DialogToolbar below it, not in the
// bar. OnDismiss is what the close button does.
func DialogTitle(title string, items ...Item) *El {
	var bar *El
	closeButton := Row(W(32), H(core.CompactH), Bg(core.Sunken), Control("icon-button"), Lighten(), Tooltip("Close (Esc)"),
		OnClick(func() {
			if bar.DismissHandler() != nil {
				bar.DismissHandler()()
			}
		}),
		CrossIcon(Grow(1)),
	)
	bar = Row(append(append([]Item{
		H(core.DialogTitleH), DragHandle(), PadX(16), Gap(8), AlignItems(AlignCenter), Bg(core.Sunken), BorderB(core.BorderSoft), Text(title, Bold()), Spacer(),
	}, items...), closeButton)...)
	return bar
}

// DialogToolbar is the row of controls under a dialog's title bar: a search
// field, Save, a switch. Give it the controls; fields that should take the
// free room get Grow(1).
func DialogToolbar(items ...Item) *El {
	return Row(append([]Item{H(core.DialogToolbarH), PadX(16), Gap(8), AlignItems(AlignCenter), BorderB(core.BorderSoft)}, items...)...)
}

// DialogButton is a button of a message dialog; the primary one is lit and
// is what Enter runs.
type DialogButton struct {
	Label   string
	Primary bool
	Run     func()
}

// DialogState is a modal message: a title, a text, an optional input line
// (a prompt) and buttons. Buttons close the dialog themselves in their
// Run, so the state needs no knowledge of its owner.
type DialogState struct {
	Title, Body string
	Input       *TextField
	Buttons     []DialogButton
}

// Default runs the primary button, if the dialog has one: what Enter does.
func (state *DialogState) Default() {
	for _, button := range state.Buttons {
		if button.Primary {
			button.Run()
			return
		}
	}
}

// MessageDialogW is the width of a message dialog.
const MessageDialogW = 380

// MessageDialog shows state as a Dialog: title, body, any items you add
// (a form, between the body and the input), the input line and the buttons
// on the right. Enter in the input runs the primary button. Give it
// OnDismiss for what an outside click does.
func MessageDialog(state *DialogState, items ...Item) *El {
	return Dialog(append([]Item{
		W(MessageDialogW), Pad(16), Gap(12),
		Text(state.Title, H(core.LabelH), DragHandle()),
		If(state.Body != "", Lines(state.Body, core.Muted)),
	}, append(items,
		If(state.Input != nil, Field(state.Input, H(core.RowH), Bg(core.Sunken), OnEnter(state.Default), TextMenu(state.Input))),
		Row(H(core.RowH), Gap(8),
			Spacer(),
			For(state.Buttons, func(_ int, button DialogButton) Node {
				return Choice(button.Label, button.Primary, MinW(80), OnClick(button.Run))
			}),
		),
	)...)...)
}
