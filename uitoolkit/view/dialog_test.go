package view

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// A dialog dims the window and floats a modal panel in the middle; a click
// outside the panel dismisses it.
func TestDialogBoxDimsTheWindowAndDismissesOnAnOutsideClick(t *testing.T) {
	c := testCtx(t)
	dismissed := 0
	panel := DialogBox(W(200), H(100), OnDismiss(func() { dismissed++ }), Text("hello"))
	root := Stack(Row(Grow(1)), panel)
	Layout(c, root, 600, 400)

	scrim, body := panel.kids[0], panel.kids[1]
	if scrim.bg != core.Shadow {
		t.Errorf("first child bg = %q, want the Shadow scrim", scrim.bg)
	}
	if !body.modal || !body.focusScope || body.control != "menu" || !body.shadow {
		t.Errorf("panel modal %v focusScope %v control %q shadow %v", body.modal, body.focusScope, body.control, body.shadow)
	}
	if body.rect.X != (600-200)/2 || body.rect.Y != (400-100)/2 {
		t.Errorf("panel at %d,%d, want it centered", body.rect.X, body.rect.Y)
	}
	click(c, 5, 5)
	Update(c, root)
	if dismissed != 1 {
		t.Fatalf("an outside click dismissed %d times, want 1", dismissed)
	}
}

func TestResizableDialogKeepsTheDialogLook(t *testing.T) {
	c := testCtx(t)
	var size ResizeState
	panel := ResizableDialogBox(&size, 200, 100, 500, 300, W(300), H(200), Text("hi"))
	Layout(c, Stack(panel), 600, 400)
	body := panel.kids[1]
	if panel.kids[0].bg != core.Shadow || !body.modal || body.control != "menu" {
		t.Fatalf("resizable dialog lost the scrim or the dialog look: scrim %q modal %v control %q", panel.kids[0].bg, body.modal, body.control)
	}
}

// The title bar is DialogTitleH high: the title, the caller's controls, and
// a close button that dismisses the dialog.
func TestDialogTitleHasTheCloseButton(t *testing.T) {
	c := testCtx(t)
	closed := 0
	bar := DialogTitle("Log", Spacer(Grow(1)), OnDismiss(func() { closed++ }))
	Layout(c, Row(W(600), H(100), bar), 600, 100)
	if bar.h != core.DialogTitleH {
		t.Errorf("height = %d, want DialogTitleH %d", bar.h, core.DialogTitleH)
	}
	closeButton := bar.kids[len(bar.kids)-1]
	click(c, closeButton.rect.X+closeButton.rect.W/2, closeButton.rect.Y+closeButton.rect.H/2)
	Update(c, Row(bar))
	if closed != 1 {
		t.Fatalf("the close button dismissed %d times, want 1", closed)
	}
}

// ---- MessageDialog --------------------------------------------------------

func TestDialogStateDefaultRunsThePrimaryButton(t *testing.T) {
	ran := ""
	state := &DialogState{Buttons: []DialogButton{
		{Label: "Cancel", Run: func() { ran = "cancel" }},
		{Label: "OK", Primary: true, Run: func() { ran = "ok" }},
	}}
	state.Default()
	if ran != "ok" {
		t.Fatalf("Default ran %q, want the primary button", ran)
	}
	(&DialogState{Buttons: []DialogButton{{Label: "Close", Run: func() { ran = "close" }}}}).Default()
	if ran != "ok" {
		t.Fatalf("a dialog without a primary button ran %q on Default", ran)
	}
}

func TestMessageDialogShowsTitleBodyInputAndButtons(t *testing.T) {
	c := testCtx(t)
	input := NewTextField("name")
	input.Focused = true
	ran := ""
	state := &DialogState{
		Title: "Rename", Body: "Pick a new name.", Input: &input,
		Buttons: []DialogButton{
			{Label: "Cancel", Run: func() { ran = "cancel" }},
			{Label: "Rename", Primary: true, Run: func() { ran = "rename" }},
		},
	}
	panel := MessageDialog(state, OnDismiss(func() {}))
	root := Stack(panel)
	Layout(c, root, 800, 600)
	dump := Dump(panel)
	for _, want := range []string{"Rename", "Pick a new name.", "fieldB", "Cancel"} {
		if !contains(dump, want) {
			t.Errorf("dialog lacks %q:\n%s", want, dump)
		}
	}
	// Enter in the input runs the primary button.
	c.In = core.Input{Enter: true}
	Update(c, root)
	if ran != "rename" {
		t.Fatalf("Enter in the input ran %q, want the primary button", ran)
	}
}

func contains(text, part string) bool { return strings.Contains(text, part) }
