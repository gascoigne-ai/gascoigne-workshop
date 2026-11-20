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
		{"OnClickAt", view.OnClickAt(func(int, int) {}), "onClick=true"},
		{"OnRightClick", view.OnRightClick(func(int, int) {}), "onRightClick=true"},
		{"OnRightClickIn", view.OnRightClickIn(func(*core.Ctx, int, int) {}), "onRightClick=true"},
		{"OnMiddleClick", view.OnMiddleClick(func() {}), "onMiddleClick=true"},
		{"OnDoubleClick", view.OnDoubleClick(func() {}), "onDoubleClick=true"},
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

// Dump promises that equal dumps mean equal looks, so an attribute that
// shapes the look must change the dump: it once omitted the size bounds,
// the alignment, the clip, the corners and more, and two different views
// compared equal.
func TestDumpShowsEveryAttributeThatShapesTheLook(t *testing.T) {
	tests := []struct {
		name       string
		one, other view.Option // other nil: compared with a bare Row
	}{
		{"MinW", view.MinW(10), nil},
		{"MinW value", view.MinW(10), view.MinW(20)},
		{"MaxW", view.MaxW(10), nil},
		{"MaxW value", view.MaxW(10), view.MaxW(20)},
		{"MinH", view.MinH(10), nil},
		{"MinH value", view.MinH(10), view.MinH(20)},
		{"MaxH", view.MaxH(10), nil},
		{"MaxH value", view.MaxH(10), view.MaxH(20)},
		{"AlignItems", view.AlignItems(view.AlignStart), nil},
		{"AlignItems value", view.AlignItems(view.AlignStart), view.AlignItems(view.AlignEnd)},
		{"AlignSelf", view.AlignSelf(view.AlignCenter), nil},
		{"AlignSelf value", view.AlignSelf(view.AlignCenter), view.AlignSelf(view.AlignEnd)},
		{"JustifyContent", view.JustifyContent(view.JustifyEnd), nil},
		{"JustifyContent value", view.JustifyContent(view.JustifyEnd), view.JustifyContent(view.JustifyBetween)},
		{"Clip", view.Clip(), nil},
		{"DismissMargin", view.DismissMargin(8), nil},
		{"DismissMargin value", view.DismissMargin(8), view.DismissMargin(9)},
		{"DragHandle", view.DragHandle(), nil},
		{"EdgeB", view.EdgeB("accent-line"), nil},
		{"EdgeB token", view.EdgeB("accent-line"), view.EdgeB("border")},
		{"EdgeB against EdgeT", view.EdgeB("accent-line"), view.EdgeT("accent-line")},
		{"RoundT", view.RoundT("button"), nil},
		{"RoundT control", view.RoundT("button"), view.RoundT("input")},
		{"RoundT against RoundB", view.RoundT("button"), view.RoundB("button")},
		{"KeepInView", view.KeepInView(), nil},
		{"Moved", view.Moved(&view.MoveState{}), nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			other := view.Row()
			if test.other != nil {
				other = view.Row(test.other)
			}
			one := view.Row(test.one)
			if view.Dump(one) == view.Dump(other) {
				t.Errorf("Dump of %s equals the dump of %s: %q", test.name, "its counterpart", view.Dump(one))
			}
		})
	}
}
