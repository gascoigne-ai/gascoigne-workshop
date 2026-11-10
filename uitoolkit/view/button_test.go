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
