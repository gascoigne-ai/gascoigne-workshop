package widget

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// A button without a width is as wide as its label plus the inset, and a
// button with a width too small for its label shortens the label instead
// of letting it spill over the edges.
func TestButtonLabelNeverOverflowsTheButton(t *testing.T) {
	c := testCtx(t)
	labels := []string{"OK", "Fill composer", "Auto-reconnect (12)"}
	tests := []struct {
		name  string
		build func(label string) *El
		fixed int // the width given, 0 for none
	}{
		{"button", func(label string) *El { return Button(label) }, 0},
		{"primary", func(label string) *El { return PrimaryButton(label) }, 0},
		{"button too narrow", func(label string) *El { return Button(label, W(50)) }, 50},
		{"primary too narrow", func(label string) *El { return PrimaryButton(label, W(50)) }, 50},
	}
	for _, test := range tests {
		for _, label := range labels {
			button := test.build(label)
			Layout(c, Row(H(40), button), 600, 40)
			text, box := button.Children()[0].Rect(), button.Rect()
			if text.X < box.X+buttonPadX || text.X+text.W > box.X+box.W-buttonPadX {
				t.Errorf("%s %q: label %+v spills into the inset of the button %+v", test.name, label, text, box)
			}
			if test.fixed == 0 && box.W < c.Width(c.F.UI, label)+2*buttonPadX {
				t.Errorf("%s %q: width %d leaves no inset around the label", test.name, label, box.W)
			}
			if test.fixed != 0 && box.W != test.fixed {
				t.Errorf("%s %q: width %d, want the given %d", test.name, label, box.W, test.fixed)
			}
		}
	}
}

func TestButtonsClickAndOverrideDefaults(t *testing.T) {
	var plain, primary int
	b1 := Button("Cancel", W(80), OnClick(func() { plain++ }))
	b2 := PrimaryButton("Send", W(80), Bg(core.Error), OnClick(func() { primary++ }))
	root := Row(b1, b2)
	c := testCtx(t)
	Layout(c, root, 200, 30)

	click(c, 10, 10)
	Update(c, root)
	click(c, 100, 10)
	Update(c, root)
	if plain != 1 || primary != 1 {
		t.Fatalf("plain %d primary %d, want 1 and 1", plain, primary)
	}
	hasProps(t, b2, "bg=error")
	hasProps(t, b1, "bg=sunken")
}

func TestIconButtonIsAnOutlinedChipAroundTheGlyph(t *testing.T) {
	glyph := TrashIcon(Grow(1))
	button := IconButton(glyph, W(40), H(28))
	hasProps(t, button, "bg=sunken", "control=icon-button")
	Layout(testCtx(t), Col(button), 100, 100)
	if glyph.Rect().W != 40 {
		t.Fatalf("the glyph must fill the button so hover and clicks land on it, W=%d", glyph.Rect().W)
	}
}

// Choice shows the selected option as a primary button and the others as
// plain ones, and keeps the options it was given (width, click).
func TestChoiceLightsOnlyTheSelectedOption(t *testing.T) {
	tests := []struct {
		name     string
		selected bool
		control  string
	}{
		{"selected", true, "control=primary"},
		{"not selected", false, "control=button"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chip := Choice("JSON", test.selected, W(70), OnClick(func() {}))
			hasProps(t, chip, test.control, "w=70", "onClick=true")
		})
	}
}

// A list row shows the selection with a fixed background and any other row
// with a hover tint; the options of the caller (height, click) still apply.
func TestListRowLooksSelectedOrHoverable(t *testing.T) {
	selected := ListRow(true, H(28), OnClick(func() {}))
	hasProps(t, selected, "control=row", "bg=selected", "h=28", "onClick=true")
	if line := props(selected); contains(line, "hoverBg") {
		t.Errorf("a selected row has no hover tint: %q", line)
	}
	hasProps(t, ListRow(false, H(28)), "control=row", "hoverBg=hover", "h=28")
}

// A note is one line of small text in the given color, NoteH high, that
// shortens with an ellipsis instead of wrapping.
func TestNoteIsOneColoredLineOfSmallText(t *testing.T) {
	note := Note("Port must be a number", core.Error)
	hasProps(t, note, "h=18")
	kids := note.Children()
	if len(kids) != 1 {
		t.Fatalf("want one text child, got %d", len(kids))
	}
	hasProps(t, kids[0], "fg=error", "small=true", "ellipsis=true")
	hasProps(t, Note("careful", core.Muted).Children()[0], "fg=muted")
}

// A form row puts a fixed-width label before its controls, which share the
// rest of the row.
func TestFormRowKeepsTheLabelColumnAligned(t *testing.T) {
	c := testCtx(t)
	shortField, longField := NewTextField(""), NewTextField("")
	short := FormRow("Timeout", 160, Field(&shortField, W(80)))
	long := FormRow("A much longer label", 160, Field(&longField, W(80)))
	Layout(c, Col(short, long), 600, 100)
	for _, row := range []*El{short, long} {
		if row.Rect().H != core.RowH {
			t.Errorf("row height = %d, want RowH %d", row.Rect().H, core.RowH)
		}
		if row.Children()[0].Rect().W != 160 {
			t.Errorf("label width = %d, want 160", row.Children()[0].Rect().W)
		}
	}
	if short.Children()[1].Rect().X != long.Children()[1].Rect().X {
		t.Errorf("controls start at %d and %d, want one column", short.Children()[1].Rect().X, long.Children()[1].Rect().X)
	}
}

// A check row is a checkbox followed by its label, and a click anywhere on
// the row toggles it.
func TestCheckRowClicksAnywhere(t *testing.T) {
	c := testCtx(t)
	toggled := 0
	row := CheckRow(true, OnClick(func() { toggled++ }), Text("Follow redirects"))
	Layout(c, Row(row), 400, 40)
	if row.Rect().H != core.RowH || len(row.Children()) != 2 {
		t.Fatalf("row height %d with %d kids, want RowH and checkbox + label", row.Rect().H, len(row.Children()))
	}
	rect := row.Rect()
	click(c, rect.X+rect.W-2, rect.Y+rect.H/2)
	Update(c, row)
	if toggled != 1 {
		t.Fatalf("clicks on the row toggled %d times, want 1", toggled)
	}
}

// A toolbar is a row of controls, vertically centered, with a line under it.
func TestToolbarCentersItsControlsAndLinesTheBottom(t *testing.T) {
	c := testCtx(t)
	bar := Toolbar(Button("Save", W(56), H(core.CompactH)))
	Layout(c, Col(W(400), H(100), bar), 400, 100)
	hasProps(t, bar, "borderB=border")
	if bar.Rect().H != core.ToolbarH {
		t.Errorf("height = %d, want ToolbarH %d", bar.Rect().H, core.ToolbarH)
	}
	button, box := bar.Children()[0].Rect(), bar.Rect()
	if want := box.Y + (box.H-button.H)/2; button.Y != want {
		t.Errorf("button Y = %d, want %d (centered)", button.Y, want)
	}
}

// A tab bar leaves room above the tabs, so their outline never touches the
// bar above, and draws a line under the inactive ones.
func TestTabBarLeavesRoomAboveTheTabs(t *testing.T) {
	c := testCtx(t)
	tabs := Tabs([]TabItem{{Label: "Params"}}, 0)
	bar := TabBar(tabs)
	Layout(c, Col(W(400), H(100), bar), 400, 100)
	hasProps(t, bar, "borderB=border")
	if bar.Rect().H != core.TabBarH {
		t.Errorf("height = %d, want TabBarH %d", bar.Rect().H, core.TabBarH)
	}
	if tabs.Rect().Y != bar.Rect().Y+core.TabGap {
		t.Errorf("tabs start at %d, want %d below the top", tabs.Rect().Y-bar.Rect().Y, core.TabGap)
	}
}

// The open tab joins the pane below it: the pane background, outlined in
// the accent on top, left and right, and kept in view when the strip scrolls.
func TestActiveTabLookIsOneOption(t *testing.T) {
	hasProps(t, Row(ActiveTab()), "bg=bg", "edge=accent-line", "edgeTop=true", "edgeLeft=true", "edgeRight=true")
	if line := props(Row()); contains(line, "edge") || contains(line, "bg=") {
		t.Fatalf("an inactive tab has no look: %q", line)
	}
}

func TestClickingATabSelectsIt(t *testing.T) {
	sel := -1
	tabs := Tabs([]TabItem{{Label: "Params", Badge: 3}, {Label: "Body"}}, 0, OnSelect(func(i int) { sel = i }), H(40))
	root := Row(tabs)
	c := testCtx(t)
	Layout(c, root, 400, 40)

	second := tabs.Children()[1].Rect()
	click(c, second.X+second.W/2, 20)
	Update(c, root)
	if sel != 1 {
		t.Fatalf("selected %d, want 1", sel)
	}
}
