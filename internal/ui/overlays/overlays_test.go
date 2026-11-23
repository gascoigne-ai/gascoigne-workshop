package overlays

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/overlay"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/menus"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

func openMenu(world *world) {
	menus.Open(world.ctx, 10, 10, []MenuItem{{Label: "One", Run: noop}})
}

// The overlays fill their own rows of the layers and leave the window's
// (the table cell, the find bar, the request) to the window.
func TestLayersMarkOnlyTheOverlaysRows(t *testing.T) {
	world := newWorld(t)
	if world.over.Layers() != (overlay.Open{}) {
		t.Fatalf("layers of closed overlays %v", world.over.Layers())
	}
	openers := map[overlay.Layer]func(){
		overlay.ContextMenu: func() { openMenu(world) },
		overlay.Dialog:      func() { world.over.Confirm("Sure?", "", "OK", noop) },
		overlay.DestPicker:  func() { world.openPicker() },
		overlay.FileBrowser: func() { world.over.Browser.Pick("Pick", filedialog.OpenFile, "", func(string) {}) },
		overlay.CallLog:     func() { world.over.Calls().Open() },
		overlay.FontList: func() {
			world.over.Settings.Show()
			world.over.Settings.Form.OpenFontList(world.ctx, world.over.Settings, 0)
		},
		overlay.Settings: func() { world.over.Settings.Show() },
	}
	for layer, open := range openers {
		fresh := newWorld(t)
		world = fresh
		open()
		got := world.over.Layers()
		if !got[layer] {
			t.Errorf("layer %d is not marked after it opened", layer)
		}
		for other := overlay.Layer(0); other <= overlay.Request; other++ {
			if got[other] && other != layer && !(layer == overlay.FontList && other == overlay.Settings) {
				t.Errorf("opening layer %d marked layer %d too", layer, other)
			}
		}
		for _, windows := range []overlay.Layer{overlay.CellEdit, overlay.FindBar, overlay.Request} {
			if got[windows] {
				t.Errorf("the overlays marked the window's layer %d", windows)
			}
		}
	}
}

func TestCloseClosesItsOwnLayerAndRefusesOthers(t *testing.T) {
	world := newWorld(t)
	openMenu(world)
	world.over.Confirm("Sure?", "", "OK", noop)
	world.openPicker()
	world.over.Browser.Pick("Pick", filedialog.OpenFile, "", func(string) {})
	world.over.Calls().Open()
	world.over.Settings.Show()
	for _, layer := range []overlay.Layer{overlay.ContextMenu, overlay.Dialog, overlay.DestPicker, overlay.FileBrowser, overlay.CallLog, overlay.Settings} {
		if !world.over.Close(layer) {
			t.Errorf("layer %d is the overlays' and Close refused it", layer)
		}
		if world.over.Layers()[layer] {
			t.Errorf("layer %d is still open after Close", layer)
		}
	}
	for _, layer := range []overlay.Layer{overlay.CellEdit, overlay.FindBar, overlay.Request} {
		if world.over.Close(layer) {
			t.Errorf("layer %d is the window's and Close took it", layer)
		}
	}
}

// A closer outlives its dialog: the dialog it opened may be gone, and the
// one that replaced it is not its to close.
func TestOpenReturnsACloserThatClosesOnlyItsDialog(t *testing.T) {
	world := newWorld(t)
	closeFirst := world.over.Open(DialogState{Title: "first"}, nil, 0)
	closeSecond := world.over.Open(DialogState{Title: "second"}, nil, 0)
	closeFirst()
	if dialog := world.over.Dialog(); dialog == nil || dialog.Title != "second" {
		t.Fatalf("the first closer closed the second dialog: %+v", world.over.Dialog())
	}
	closeSecond()
	if world.over.Busy() {
		t.Fatal("the closer of the open dialog did not close it")
	}
	closeSecond()
	if world.over.Busy() {
		t.Fatal("a closer called twice opened something")
	}
}

func TestEnterRunsTheDefaultOfADialogWithoutAField(t *testing.T) {
	tests := map[string]struct {
		open  func(world *world, answer *confirmed)
		ran   int
		open_ bool
	}{
		"a confirm runs its primary button": {func(world *world, answer *confirmed) { world.over.Confirm("Sure?", "", "OK", answer.run) }, 1, false},
		"a prompt keeps Enter for its field": {func(world *world, answer *confirmed) {
			world.over.Prompt("Name", "x", "OK", func(string) { answer.run() })
		}, 0, true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			world := newWorld(t)
			var answer confirmed
			test.open(world, &answer)
			world.frame(core.Input{Enter: true})
			if answer.ran != test.ran || world.over.Busy() != test.open_ {
				t.Fatalf("ran %d times, dialog open %v; want %d and %v", answer.ran, world.over.Busy(), test.ran, test.open_)
			}
		})
	}
}

func TestPickerAndBrowserKeysWaitWhileADialogIsOpen(t *testing.T) {
	world := newWorld(t)
	host := world.openPicker()
	world.over.Confirm("Sure?", "", "OK", noop)
	world.frame(core.Input{Edits: []core.Edit{core.EditDown}})
	world.over.CloseDialog()
	world.frame(core.Input{Enter: true})
	if host.picked != 0 {
		t.Fatal("the picker moved its selection under a dialog")
	}
	world.frame(core.Input{Edits: []core.Edit{core.EditDown}})
	world.frame(core.Input{Enter: true})
	if host.picked != 1 {
		t.Fatal("the picker took no key once the dialog was gone")
	}
}

func TestTheBrowserFieldYieldsTheKeyboardToADialog(t *testing.T) {
	world := newWorld(t)
	world.over.Browser.Pick("Pick", filedialog.OpenFile, "", func(string) {})
	world.over.Confirm("Sure?", "", "OK", noop)
	if !world.over.Busy() || !world.over.Layers()[overlay.FileBrowser] {
		t.Fatal("the browser and a question over it are not both open")
	}
	world.frame(core.Input{Edits: []core.Edit{core.EditDown}})
	if !world.over.Browser.Shown() {
		t.Fatal("the browser closed under a dialog")
	}
	if path := world.over.Browser.Path(); strings.Contains(path, "a.json") {
		t.Fatalf("the browser moved its selection under a dialog: %q", path)
	}
}

func TestHoldsKeyboardCoversMenuDialogPickerBrowserCallLog(t *testing.T) {
	openers := map[string]func(world *world){
		"menu":    openMenu,
		"dialog":  func(world *world) { world.over.Confirm("Sure?", "", "OK", noop) },
		"picker":  func(world *world) { world.openPicker() },
		"browser": func(world *world) { world.over.Browser.Pick("Pick", filedialog.OpenFile, "", func(string) {}) },
		"calllog": func(world *world) { world.over.Calls().Open() },
	}
	if newWorld(t).over.HoldsKeyboard() {
		t.Error("closed overlays hold the keyboard")
	}
	for name, open := range openers {
		world := newWorld(t)
		open(world)
		if !world.over.HoldsKeyboard() {
			t.Errorf("an open %s does not hold the keyboard", name)
		}
	}
	world := newWorld(t)
	world.over.Settings.Show()
	if world.over.HoldsKeyboard() {
		t.Error("the settings dialog holds the keyboard like a dialog")
	}
}

func TestAMenuHasTheKeysBeforeADialog(t *testing.T) {
	world := newWorld(t)
	var answer confirmed
	world.over.Confirm("Sure?", "", "OK", answer.run)
	openMenu(world)
	world.frame(core.Input{Enter: true})
	if answer.ran != 0 {
		t.Fatal("Enter reached the dialog under an open menu")
	}
}
