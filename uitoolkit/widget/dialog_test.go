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

// The bar is a distinct strip: its own color, a line under it, and the room
// between the title and the close button drags the dialog.
func TestDialogTitleIsADragStripOfItsOwn(t *testing.T) {
	c := testCtx(t)
	move := &MoveState{}
	bar := DialogTitle("Log", OnDismiss(func() {}))
	panel := Col(W(400), H(200), FloatCenter(), Moved(move), Modal(), bar)
	root := Stack(Row(), panel)
	Layout(c, root, 800, 600)
	hasProps(t, bar, "bg=frame", "borderB=border", "hoverBg=hover")
	r := bar.Rect()
	x, y := r.X+r.W/2, r.Y+r.H/2 // between the title and the close button
	c.In = core.Input{X: x, Y: y, Click: true, Held: true}
	Update(c, root)
	c.In = core.Input{X: x + 30, Y: y + 20, Held: true}
	Update(c, root)
	if move.X != 30 || move.Y != 20 {
		t.Fatalf("moved by %d,%d, want 30,20", move.X, move.Y)
	}
}

func TestDialogToolbarIsARowOfControlsUnderTheTitle(t *testing.T) {
	c := testCtx(t)
	bar := DialogToolbar(Spacer(), Row(W(30)))
	Layout(c, Col(W(400), H(100), bar), 400, 100)
	if bar.Rect().H != core.DialogToolbarH {
		t.Fatalf("height = %d, want DialogToolbarH %d", bar.Rect().H, core.DialogToolbarH)
	}
	kids := bar.Children()
	if kids[0].Rect().X != 16 || kids[1].Rect().X+kids[1].Rect().W != 400-16 {
		t.Fatalf("controls at %+v and %+v, want them between the 16 px paddings", kids[0].Rect(), kids[1].Rect())
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

func TestAMessageDialogMovesWhenItsTitleIsDragged(t *testing.T) {
	state, move := &DialogState{Title: "Delete?", Body: "Sure?"}, &MoveState{}
	frag := MessageDialog(state, Moved(move))
	root := Stack(Row(), frag)
	c := testCtx(t)
	Layout(c, root, 800, 600)
	panel := frag.Children()[1].Rect()
	x, y := panel.X+panel.W/2, panel.Y+20 // on the title row, below the 16 px padding
	c.In = core.Input{X: x, Y: y, Click: true, Held: true}
	Update(c, root)
	c.In = core.Input{X: x + 50, Y: y + 25, Held: true}
	Update(c, root)
	if move.X != 50 || move.Y != 25 {
		t.Fatalf("moved by %d,%d, want 50,25", move.X, move.Y)
	}
}

func TestAMessageDialogHasATitleBarWhoseCloseButtonDismissesIt(t *testing.T) {
	state, closed := &DialogState{Title: "Delete?", Body: "Sure?"}, 0
	frag := MessageDialog(state, OnDismiss(func() { closed++ }))
	root := Stack(Row(), frag)
	c := testCtx(t)
	Layout(c, root, 800, 600)
	panel := frag.Children()[1]
	bar := panel.Children()[0]
	if bar.Rect().H != core.DialogTitleH || bar.Rect().Y != panel.Rect().Y {
		t.Fatalf("title bar %+v in panel %+v, want it flush at the top, DialogTitleH high", bar.Rect(), panel.Rect())
	}
	cb := bar.Children()[len(bar.Children())-1].Rect()
	click(c, cb.X+cb.W/2, cb.Y+cb.H/2)
	Update(c, root)
	if closed != 1 {
		t.Fatalf("the close button dismissed %d times, want 1", closed)
	}
}

func TestDialogSubtitleSitsAfterTheTitleAndGivesWayToTheCloseButton(t *testing.T) {
	c := testCtx(t)
	bar := DialogTitle("Settings", DialogSubtitle("A very long page name that cannot fit in a narrow dialog title bar"), OnDismiss(func() {}))
	Layout(c, Col(W(260), H(100), bar), 260, 100)
	kids := bar.Children()
	title, sub, closeButton := kids[0].Rect(), kids[1].Rect(), kids[len(kids)-1].Rect()
	if sub.X < title.X+title.W {
		t.Errorf("subtitle at %d overlaps the title ending at %d", sub.X, title.X+title.W)
	}
	if sub.X+sub.W > closeButton.X {
		t.Errorf("subtitle ends at %d, past the close button at %d", sub.X+sub.W, closeButton.X)
	}
}
