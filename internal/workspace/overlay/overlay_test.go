package overlay

import "testing"

func TestEscapeClosesTheTopmostOpenLayerFirst(t *testing.T) {
	order := []Layer{ContextMenu, Dialog, DestPicker, FileBrowser, CallLog, FontList, Settings, CellEdit, FindBar, Request}
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

func TestOnlyMenusDialogsPickersAndTheCallLogHoldTheKeyboard(t *testing.T) {
	holders := map[Layer]bool{ContextMenu: true, Dialog: true, DestPicker: true, FileBrowser: true, CallLog: true}
	for layer := Layer(0); layer < layers; layer++ {
		var open Open
		open[layer] = true
		if got := open.HoldsKeyboard(); got != holders[layer] {
			t.Errorf("layer %d: HoldsKeyboard = %v, want %v", layer, got, holders[layer])
		}
	}
	if (Open{}).HoldsKeyboard() {
		t.Error("nothing open holds the keyboard")
	}
}
