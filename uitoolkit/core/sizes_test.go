package core

import "testing"

// Each size keeps its value from before the rules (the golden scenes depend
// on it) and equals its rule: a height is a base line plus padding on both
// sides, a spacing is a whole number of Units, a bar is a class plus a step.
func TestSizeConstantsFollowTheRules(t *testing.T) {
	tests := []struct {
		name      string
		got, rule int
		want      int
		units     int // the step in Units, for spacings; 0 for the rest
	}{
		{"RowH", RowH, BaseLine + 2*(3*Unit), 28, 0},
		{"CompactH", CompactH, BaseLine + 2*(2*Unit), 24, 0},
		{"SlimH", SlimH, BaseLine + 2*(1*Unit), 20, 0},
		{"LabelH", LabelH, BaseLine + 3*Unit, 22, 0},
		{"NoteH", NoteH, BaseSmallLine + 2*(1*Unit), 18, 0},
		{"TabLead", TabLead, StepHair, 2, 1},
		{"EdgeW", EdgeW, StepHair, 2, 1},
		{"NearGap", NearGap, StepNear, 4, 2},
		{"ShadowOffset", ShadowOffset, StepNear, 4, 2},
		{"TightGap", TightGap, StepTight, 6, 3},
		{"TabGap", TabGap, StepTight, 6, 3},
		{"RuleInset", RuleInset, StepTight, 6, 3},
		{"EdgeGrab", EdgeGrab, StepTight, 6, 3},
		{"LooseGap", LooseGap, StepLoose, 8, 4},
		{"CellPad", CellPad, StepLoose, 8, 4},
		{"LabelPad", LabelPad, StepLabel, 10, 5},
		{"TabSepInset", TabSepInset, StepLabel, 10, 5},
		{"Gutter", Gutter, StepGutter, 12, 6},
		{"DialogPad", DialogPad, StepDialog, 16, 8},
		{"ToolbarH", ToolbarH, RowH + 2*NearGap, 36, 0},
		{"DialogToolbarH", DialogToolbarH, RowH + 2*TightGap, 40, 0},
		{"TabBarH", TabBarH, RowH + 2*TightGap, 40, 0},
		{"DialogTitleH", DialogTitleH, RowH + 2*LooseGap, 44, 0},
		{"IconBtnW", IconBtnW, CompactH + 2*NearGap, 32, 0},
		{"WordBtnW", WordBtnW, IconBtnW + Gutter, 44, 0},
		{"CheckboxSize", CheckboxSize, BaseSmallLine, 14, 0},
		{"DialogMargin", DialogMargin, 40, 40, 0},
		{"DialogMaxH", DialogMaxH, 560, 560, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Errorf("%s = %d, want %d as before the rules", test.name, test.got, test.want)
			}
			if test.got != test.rule {
				t.Errorf("%s = %d, its rule gives %d", test.name, test.got, test.rule)
			}
			if test.units > 0 && test.got != test.units*Unit {
				t.Errorf("%s = %d, not step %d of Unit %d", test.name, test.got, test.units, Unit)
			}
		})
	}
}
