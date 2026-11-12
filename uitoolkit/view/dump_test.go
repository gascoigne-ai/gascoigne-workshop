package view_test

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// A view and its template are compared by Dump, so every handler has to show
// in it: two elements that differ only in what a click does must not dump
// alike.
func TestDumpNamesEveryHandlerAnElementHas(t *testing.T) {
	tests := []struct {
		name   string
		option view.Option
		want   string
	}{
		{"OnClick", view.OnClick(func() {}), "onClick=true"},
		{"OnClickAt", view.OnClickAt(func(int, int) {}), "onClickAt=true"},
		{"OnRightClick", view.OnRightClick(func(int, int) {}), "onRightClick=true"},
		{"OnRightClickIn", view.OnRightClickIn(func(*core.Ctx, int, int) {}), "onRightClickIn=true"},
		{"OnMiddleClick", view.OnMiddleClick(func() {}), "onMiddleClick=true"},
		{"OnDragOver", view.OnDragOver(func() {}), "onDragOver=true"},
		{"OnChange", view.OnChange(func() {}), "onChange=true"},
		{"OnSelect", view.OnSelect(func(int) {}), "onSelect=true"},
		{"OnDismiss", view.OnDismiss(func() {}), "onDismiss=true"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dump := view.Dump(view.Row(test.option))
			if !strings.Contains(dump, test.want) {
				t.Errorf("Dump of a Row with %s = %q, want it to contain %q", test.name, dump, test.want)
			}
		})
	}
}

func TestDumpOfElementsWithoutHandlersNamesNone(t *testing.T) {
	if dump := view.Dump(view.Row()); dump != "flow\n" {
		t.Errorf("Dump of a bare Row = %q, want just its kind", dump)
	}
}
