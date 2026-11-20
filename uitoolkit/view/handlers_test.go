package view

import "testing"

func TestSelectDismissChangedRunTheirHandlers(t *testing.T) {
	var picked []int
	dismissed, changed := 0, 0
	element := Row(
		OnSelect(func(index int) { picked = append(picked, index) }),
		OnDismiss(func() { dismissed++ }),
		OnChange(func() { changed++ }),
	)
	element.Select(3)
	element.Dismiss()
	element.Changed()
	if len(picked) != 1 || picked[0] != 3 || dismissed != 1 || changed != 1 {
		t.Errorf("picked %v, dismissed %d, changed %d; want [3], 1, 1", picked, dismissed, changed)
	}
}

func TestSelectDismissChangedWithoutHandlerDoNothing(t *testing.T) {
	element := Row()
	element.Select(1)
	element.Dismiss()
	element.Changed()
}
