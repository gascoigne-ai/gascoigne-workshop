package widget

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// A dialog dims the window and floats a modal panel in the middle; a click
// outside the panel dismisses it.
func TestDialogDimsTheWindowAndDismissesOnAnOutsideClick(t *testing.T) {
	c := testCtx(t)
	dismissed := 0
	panel := Dialog(W(200), H(100), OnDismiss(func() { dismissed++ }), Text("hello"))
	root := Stack(Row(Grow(1)), panel)
	Layout(c, root, 600, 400)

	scrim, body := panel.Children()[0], panel.Children()[1]
	hasProps(t, scrim, "bg=shadow")
	hasProps(t, body, "modal=true", "focusScope=true", "control=menu", "shadow=true", "onDismiss=true")
	if rect := body.Rect(); rect.X != (600-200)/2 || rect.Y != (400-100)/2 {
		t.Errorf("panel at %d,%d, want it centered", rect.X, rect.Y)
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
	panel := ResizableDialog(&size, 200, 100, 500, 300, W(300), H(200), Text("hi"))
	Layout(c, Stack(panel), 600, 400)
	hasProps(t, panel.Children()[0], "bg=shadow")
	hasProps(t, panel.Children()[1], "modal=true", "control=menu")
}

// The title bar is DialogTitleH high: the title, the caller's controls, and
// a close button that dismisses the dialog.
func TestDialogTitleHasTheCloseButton(t *testing.T) {
	c := testCtx(t)
	closed := 0
	bar := DialogTitle("Log", Spacer(Grow(1)), OnDismiss(func() { closed++ }))
	Layout(c, Row(W(600), H(100), bar), 600, 100)
	if bar.Rect().H != core.DialogTitleH {
		t.Errorf("height = %d, want DialogTitleH %d", bar.Rect().H, core.DialogTitleH)
	}
	closeButton := bar.Children()[len(bar.Children())-1].Rect()
	click(c, closeButton.X+closeButton.W/2, closeButton.Y+closeButton.H/2)
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
