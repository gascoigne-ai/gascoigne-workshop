package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	reqport "github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Window sizes the overlays are pinned at: the test window and a small
// one, where the dialog is still centered and the toasts move with the
// corner.
var overlaySizes = []struct {
	name          string
	width, height int
}{
	{"wide", winW, winH},
	{"narrow", 480, 360},
}

// assertDumpPinned lays root out in a window of width by height and
// compares its dump with testdata/name.dump.
func assertDumpPinned(t *testing.T, a *App, root Node, name string, width, height int) {
	t.Helper()
	Layout(a.ctx, root, width, height)
	want, err := os.ReadFile(filepath.Join("testdata", name+".dump"))
	if err != nil {
		t.Fatal(err)
	}
	if got := Dump(root); got != string(want) {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// dialogKinds open each kind of dialog the app shows: a confirm (body and
// buttons), a prompt (an input), an MCP tool's typed form and an MCP
// sampling request (a long body and an input).
var dialogKinds = []struct {
	name string
	open func(a *App)
}{
	{"confirm", func(a *App) {
		a.confirm("Delete Search Repos?", "This cannot be undone.", "Delete", func() {})
	}},
	{"prompt", func(a *App) {
		a.prompt("Rename", "Search Repos", "Rename", func(string) {})
	}},
	{"toolform", func(a *App) { a.openToolForm(a.cur(), searchTool()) }},
	{"sampling", func(a *App) {
		a.openSamplingDialog(samplingAsked, make(chan reqport.SamplingResult, 1), make(chan struct{}, 1))
	}},
}

// Each kind of dialog is pinned at both window sizes. The dialog floats,
// so it is laid out inside a Stack the size of the window.
func TestTheDialogLaysOutAsPinned(t *testing.T) {
	for _, kind := range dialogKinds {
		for _, size := range overlaySizes {
			name := "dialog-" + kind.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				a, _ := newTestApp(t)
				t.Cleanup(a.waitWrites)
				kind.open(a)
				assertDumpPinned(t, a, Stack(a.dialogView()), name, size.width, size.height)
			})
		}
	}
}

// The prompt's field edits the dialog's own input, which the dump does not
// print.
func TestThePromptFieldEditsTheDialogInput(t *testing.T) {
	a, _ := newTestApp(t)
	t.Cleanup(a.waitWrites)
	a.prompt("Rename", "Search Repos", "Rename", func(string) {})
	root := a.dialogView()
	Layout(a.ctx, Stack(root), winW, winH)
	field := findWidget(root, "*widget.fieldB")
	if field == nil {
		t.Fatalf("no field in the prompt:\n%s", Dump(root))
	}
	if got, want := fmt.Sprintf("%+v", field.Behavior()), fmt.Sprintf("%p", a.dialog.Input); !strings.Contains(got, want) {
		t.Errorf("the field edits %s, want the dialog's input %s", got, want)
	}
}

// No dialog, no view.
func TestNoDialogNoDialogView(t *testing.T) {
	a, _ := newTestApp(t)
	t.Cleanup(a.waitWrites)
	if root := a.dialogView(); root != nil {
		t.Fatalf("dialog view %v with no dialog open", root)
	}
}

// The toasts are pinned at both window sizes, an info toast above an error
// whose text runs out of room.
func TestTheToastsLayOutAsPinned(t *testing.T) {
	for _, size := range overlaySizes {
		name := "toasts-" + size.name
		t.Run(name, func(t *testing.T) {
			a, _ := newTestApp(t)
			t.Cleanup(a.waitWrites)
			a.w, a.h = size.width, size.height
			a.toast("Cookies cleared", false)
			a.toast("Enter the gRPC server first, for example grpc://localhost:50051", true)
			assertDumpPinned(t, a, Stack(a.toastsView()), name, size.width, size.height)
		})
	}
}

// With nothing to show the toasts add nothing to the window.
func TestNoToastsNoToastsView(t *testing.T) {
	a, _ := newTestApp(t)
	t.Cleanup(a.waitWrites)
	root := Stack(a.toastsView())
	Layout(a.ctx, root, winW, winH)
	if got := len(root.Children()); got != 0 {
		t.Fatalf("the window gets %d toast views with no toasts:\n%s", got, Dump(root))
	}
}

// Clicking a toast's text dismisses it, and Enter in an open confirm runs
// its primary button, through the whole window.
func TestToastsAndDialogAnswerInput(t *testing.T) {
	a, _ := newTestApp(t)
	t.Cleanup(a.waitWrites)
	a.toast("There is no response to save", true)
	a.step(core.Input{})
	clickText(t, a, "There is no response to save", 0)
	if len(a.toasts) != 0 {
		t.Fatalf("toasts %+v after the click", a.toasts)
	}
	ran := 0
	a.confirm("Delete Search Repos?", "This cannot be undone.", "Delete", func() { ran++ })
	a.step(core.Input{})
	clickText(t, a, "Delete", 0)
	if ran != 1 || a.dialog != nil {
		t.Fatalf("Delete ran %d times, dialog open %v", ran, a.dialog != nil)
	}
}
