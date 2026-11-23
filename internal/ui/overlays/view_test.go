package overlays

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// With nothing open every slot is nil: a typed nil in a Node would make the
// window's template draw an empty layer.
func TestNoOverlayOpenNoViews(t *testing.T) {
	world := newWorld(t)
	views := world.over.Views(world.ctx)
	slots := map[string]Node{
		"settings": views.Settings, "picker": views.Picker, "browser": views.Browser, "popup": views.Popup,
		"dialog": views.Dialog, "callLog": views.CallLog,
	}
	for name, slot := range slots {
		if slot != nil {
			t.Errorf("the %s slot has a node while closed", name)
		}
	}
}

func TestEachOpenOverlayFillsItsOwnSlot(t *testing.T) {
	tests := map[string]struct {
		open func(world *world)
		slot func(Views) Node
		want string
	}{
		"dialog": {func(world *world) { world.over.Confirm("Delete it?", "No way back", "Delete", noop) },
			func(views Views) Node { return views.Dialog }, `text="Delete it?"`},
		"call log": {func(world *world) { world.over.Calls().Open() },
			func(views Views) Node { return views.CallLog }, "Log"},
		"browser": {func(world *world) { world.over.Browser.Pick("Choose a file", filedialog.OpenFile, "", func(string) {}) },
			func(views Views) Node { return views.Browser }, "Choose a file"},
		"picker": {func(world *world) { world.openPicker() },
			func(views Views) Node { return views.Picker }, "Source"},
		"popup menu": {openMenu,
			func(views Views) Node { return views.Popup }, "One"},
		"toast": {func(world *world) { world.over.Toasts.Show("Saved", severity.Info) },
			func(views Views) Node { return views.Toasts }, "Saved"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			world := newWorld(t)
			test.open(world)
			if dump := world.dump(test.slot); !strings.Contains(dump, test.want) {
				t.Fatalf("the slot lacks %q:\n%s", test.want, dump)
			}
		})
	}
}

// A dialog's own content sits in the dialog, built with the frame's context
// and in the width the opener asked for.
func TestADialogDrawsItsOwnContentInItsWidth(t *testing.T) {
	world := newWorld(t)
	var built *w.Ctx
	world.over.Open(DialogState{Title: "Form"}, func(ctx *w.Ctx) Node {
		built = ctx
		return Text("typed form")
	}, 560)
	dump := world.dump(func(views Views) Node { return views.Dialog })
	if built != world.ctx || !strings.Contains(dump, "typed form") || !strings.Contains(dump, "w=560") {
		t.Fatalf("built with %v:\n%s", built == world.ctx, dump)
	}
}

func TestADialogWithoutContentHasTheMessageWidth(t *testing.T) {
	world := newWorld(t)
	world.over.Confirm("Sure?", "", "OK", noop)
	if dump := world.dump(func(views Views) Node { return views.Dialog }); strings.Contains(dump, "w=560") {
		t.Fatalf("a message dialog got the wide panel:\n%s", dump)
	}
}
