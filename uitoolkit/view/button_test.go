package view

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
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
			text := button.kids[0]
			if text.rect.X < button.rect.X+buttonPadX || text.rect.X+text.rect.W > button.rect.X+button.rect.W-buttonPadX {
				t.Errorf("%s %q: label %+v spills into the inset of the button %+v", test.name, label, text.rect, button.rect)
			}
			if test.fixed == 0 && button.rect.W < c.Width(c.F.UI, label)+2*buttonPadX {
				t.Errorf("%s %q: width %d leaves no inset around the label", test.name, label, button.rect.W)
			}
			if test.fixed != 0 && button.rect.W != test.fixed {
				t.Errorf("%s %q: width %d, want the given %d", test.name, label, button.rect.W, test.fixed)
			}
		}
	}
}

// Choice shows the selected option as a primary button and the others as
// plain ones, and keeps the options it was given (width, click).
func TestChoiceLightsOnlyTheSelectedOption(t *testing.T) {
	clicked := 0
	tests := []struct {
		name     string
		selected bool
		control  string
	}{
		{"selected", true, "primary"},
		{"not selected", false, "button"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chip := Choice("JSON", test.selected, W(70), OnClick(func() { clicked++ }))
			if chip.control != test.control {
				t.Errorf("control = %q, want %q", chip.control, test.control)
			}
			if chip.w != 70 {
				t.Errorf("width = %d, want the given 70", chip.w)
			}
			if chip.onClick == nil {
				t.Error("the click handler was dropped")
			}
		})
	}
}

// A list row shows the selection with a fixed background and any other row
// with a hover tint; the options of the caller (height, click) still apply.
func TestListRowLooksSelectedOrHoverable(t *testing.T) {
	selected := ListRow(true, H(28), OnClick(func() {}))
	if selected.control != "row" || selected.bg != core.Selected || selected.hoverBg != "" {
		t.Errorf("selected row: control %q bg %v hoverBg %v", selected.control, selected.bg, selected.hoverBg)
	}
	plain := ListRow(false, H(28))
	if plain.control != "row" || plain.hoverBg != core.Hover || plain.bg == core.Selected {
		t.Errorf("plain row: control %q bg %v hoverBg %v", plain.control, plain.bg, plain.hoverBg)
	}
	if selected.h != 28 || selected.onClick == nil {
		t.Errorf("selected row lost its options: h %d onClick %v", selected.h, selected.onClick != nil)
	}
}

// A note is one line of small text in the given color, NoteH high, that
// shortens with an ellipsis instead of wrapping.
func TestNoteIsOneColoredLineOfSmallText(t *testing.T) {
	c := testCtx(t)
	note := Note("Port must be a number", core.Error)
	Layout(c, Row(W(400), H(60), note), 400, 60)
	if note.h != core.NoteH {
		t.Errorf("height = %d, want NoteH %d", note.h, core.NoteH)
	}
	if len(note.kids) != 1 || note.kids[0].fg != core.Error {
		t.Fatalf("want one text child in the Error color, got %d kids", len(note.kids))
	}
	if muted := Note("careful", core.Muted); muted.kids[0].fg != core.Muted {
		t.Errorf("muted note color = %q", muted.kids[0].fg)
	}
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
		if row.h != core.RowH {
			t.Errorf("row height = %d, want RowH %d", row.h, core.RowH)
		}
		if row.kids[0].rect.W != 160 {
			t.Errorf("label width = %d, want 160", row.kids[0].rect.W)
		}
	}
	if short.kids[1].rect.X != long.kids[1].rect.X {
		t.Errorf("controls start at %d and %d, want one column", short.kids[1].rect.X, long.kids[1].rect.X)
	}
}

// A check row is a checkbox followed by its label, and a click anywhere on
// the row toggles it.
func TestCheckRowClicksAnywhere(t *testing.T) {
	c := testCtx(t)
	toggled := 0
	row := CheckRow(true, OnClick(func() { toggled++ }), Text("Follow redirects"))
	Layout(c, Row(row), 400, 40)
	if row.h != core.RowH || len(row.kids) != 2 {
		t.Fatalf("row height %d with %d kids, want RowH and checkbox + label", row.h, len(row.kids))
	}
	click(c, row.rect.X+row.rect.W-2, row.rect.Y+row.rect.H/2)
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
	if bar.h != core.ToolbarH {
		t.Errorf("height = %d, want ToolbarH %d", bar.h, core.ToolbarH)
	}
	button := bar.kids[0]
	if want := bar.rect.Y + (bar.rect.H-button.rect.H)/2; button.rect.Y != want {
		t.Errorf("button Y = %d, want %d (centered)", button.rect.Y, want)
	}
	if bar.borderB == "" {
		t.Error("a toolbar has a line under it")
	}
}

// A tab bar leaves room above the tabs, so their outline never touches the
// bar above, and draws a line under the inactive ones.
func TestTabBarLeavesRoomAboveTheTabs(t *testing.T) {
	c := testCtx(t)
	tabs := Tabs([]TabItem{{Label: "Params"}}, 0)
	bar := TabBar(tabs)
	Layout(c, Col(W(400), H(100), bar), 400, 100)
	if bar.h != core.TabBarH {
		t.Errorf("height = %d, want TabBarH %d", bar.h, core.TabBarH)
	}
	if tabs.rect.Y != bar.rect.Y+core.TabGap {
		t.Errorf("tabs start at %d, want %d below the top", tabs.rect.Y-bar.rect.Y, core.TabGap)
	}
	if bar.borderB == "" {
		t.Error("a tab bar has a line under it")
	}
}
