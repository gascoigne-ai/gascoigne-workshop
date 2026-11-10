package overlay

import "testing"

func TestEscapeClosesTheTopmostOpenLayerFirst(t *testing.T) {
	order := []Layer{ContextMenu, Dialog, CallLog, FontList, Settings, CellEdit, FindBar, Request}
	var open Open
	for _, layer := range order {
		open[layer] = true
	}
	for _, want := range order {
		got, ok := open.Top()
		if !ok || got != want {
			t.Fatalf("Top = %v %v, want %v", got, ok, want)
		}
		open[got] = false
	}
	if _, ok := open.Top(); ok {
		t.Fatal("nothing is open")
	}
}
