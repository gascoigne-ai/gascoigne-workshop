package view

import "testing"

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
