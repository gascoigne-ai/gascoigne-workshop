package widget

import (
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Pins taken before the styler and table-state renames: what the field's,
// the area's and the table's stylers receive, and the chips' dump.

func TestFieldStylerIsAskedForTheFieldTextWhenDrawn(t *testing.T) {
	field := NewTextField("https://h/v1")
	var asked []string
	root := Row(Field(&field, Styled(func(text string) []Span {
		asked = append(asked, text)
		return []Span{{Text: text, Style: core.OK}}
	})))
	drawn(t, testCtx(t), root, 300, 30)
	if len(asked) == 0 || asked[0] != "https://h/v1" {
		t.Fatalf("styler asked for %q, want the field text", asked)
	}
}

func TestAreaStylerIsCalledOncePerLine(t *testing.T) {
	area := NewTextArea("one\ntwo\nthree")
	var asked []string
	area.Styler = func(line string) []Span {
		asked = append(asked, line)
		return []Span{{Text: line, Style: core.OK}}
	}
	drawn(t, testCtx(t), Col(Area(&area, H(200))), 300, 200)
	if want := []string{"one", "two", "three"}; !reflect.DeepEqual(asked, want) {
		t.Fatalf("styler lines = %q, want %q", asked, want)
	}
}

func TestAMessageDialogRendersEachButtonWithItsControl(t *testing.T) {
	state := &DialogState{Title: "T", Buttons: []DialogButton{
		{Label: "Cancel", Run: func() {}},
		{Label: "Go", Primary: true, Run: func() {}},
	}}
	dump := Dump(MessageDialog(state, OnDismiss(func() {})))
	for _, want := range []string{`text="Cancel"`, `text="Go"`, "control=primary", "control=button"} {
		if !contains(dump, want) {
			t.Errorf("dialog lacks %q:\n%s", want, dump)
		}
	}
}

func TestADumpOfASegmentedRowShowsTheSelectedChipAsPrimary(t *testing.T) {
	dump := Dump(Row(SegmentChip("A", true, W(40)), SegmentChip("B", false, W(40))))
	for _, want := range []string{"control=primary", "control=button", `text="A"`, `text="B"`} {
		if !contains(dump, want) {
			t.Errorf("segmented row lacks %q:\n%s", want, dump)
		}
	}
}

func TestTableWeightsFallBackToTheSpecUnlessTheStateMatches(t *testing.T) {
	tests := []struct {
		name  string
		state *TableState
		want  float64
	}{
		{"no state", nil, 1},
		{"empty state", &TableState{}, 1},
		{"wrong length", &TableState{Weights: []float64{9}}, 1},
		{"matching state", &TableState{Weights: []float64{9, 5}}, 9},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec := TableSpec{Columns: []string{"a", "b"}, Weights: []float64{1, 2}, State: test.state}
			if got := spec.weight(0); got != test.want {
				t.Fatalf("weight = %v, want %v", got, test.want)
			}
		})
	}
}
