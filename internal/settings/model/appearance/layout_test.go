package appearance

import "testing"

// Stacked is the layout most people want, so it is the first choice.
func TestStackedIsTheFirstLayoutChoice(t *testing.T) {
	if LayoutChoices[0].ID != "stacked" {
		t.Fatalf("first layout choice is %q", LayoutChoices[0].ID)
	}
}
