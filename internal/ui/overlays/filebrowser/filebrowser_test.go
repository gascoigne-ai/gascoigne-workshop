package filebrowser

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace"
	browsing "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/filebrowser"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
)

func homeWith(names ...string) fakeFolders {
	var entries []workspace.FolderEntry
	for _, name := range names {
		entries = append(entries, workspace.FolderEntry{Name: name})
	}
	return fakeFolders{home: "/home/me", entries: map[string][]workspace.FolderEntry{"/home/me": entries}}
}

func TestPickUsesTheDesktopDialogWhenAvailable(t *testing.T) {
	world := newWorld(t, homeWith("a.json"))
	asked := make(chan string, 1)
	world.desktop.choose = func(_ context.Context, title string, mode filedialog.Mode, name string) (string, error) {
		asked <- title + "|" + name
		return "/home/me/a.json", nil
	}
	got := ""
	world.c.Pick("Pick one", filedialog.SaveFile, "x.json", func(path string) { got = path })
	if world.c.Shown() {
		t.Fatal("the in-app browser opened although the desktop has a dialog")
	}
	select {
	case got := <-asked:
		if want := "Pick one|x.json"; got != want {
			t.Fatalf("the desktop was asked %q, want %q", got, want)
		}
	default:
		t.Fatal("the desktop was not asked")
	}
	world.bus.drain()
	if got != "/home/me/a.json" {
		t.Fatalf("the path the desktop gave was %q", got)
	}
}

func TestPickFallsBackToTheBrowser(t *testing.T) {
	world := newWorld(t, homeWith("a.json", "b.json"))
	got := ""
	world.c.Pick("Pick one", filedialog.OpenFile, "", func(path string) { got = path })
	if !world.c.Shown() {
		t.Fatal("no dialog program, and the browser did not open")
	}
	if want := "/home/me"; world.c.session.model.Dir != want {
		t.Fatalf("browser in %q, want the home folder %q", world.c.session.model.Dir, want)
	}
	if got != "" {
		t.Fatalf("answered before the user chose: %q", got)
	}
	world.c.Close()
	if world.c.Shown() {
		t.Fatal("Close left the browser open")
	}
}

func TestDesktopFailureIsAnErrorToast(t *testing.T) {
	world := newWorld(t, homeWith())
	world.desktop.choose = func(context.Context, string, filedialog.Mode, string) (string, error) {
		return "", errors.New("no display")
	}
	called := false
	world.c.Pick("Pick one", filedialog.OpenFile, "", func(string) { called = true })
	world.bus.drain()
	if len(world.window.toasts) != 1 || world.window.toasts[0].level != severity.Error ||
		!strings.Contains(world.window.toasts[0].text, "File dialog failed: no display") {
		t.Fatalf("toasts %+v, want one error naming the failure", world.window.toasts)
	}
	if called {
		t.Fatal("a failed dialog answered")
	}
}

func TestACancelledDesktopDialogSaysNothing(t *testing.T) {
	world := newWorld(t, homeWith())
	world.desktop.choose = func(context.Context, string, filedialog.Mode, string) (string, error) {
		return "", filedialog.ErrCancelled
	}
	world.c.Pick("Pick one", filedialog.OpenFile, "", func(string) { t.Error("answered") })
	world.bus.drain()
	if len(world.window.toasts) != 0 {
		t.Fatalf("toasts %+v after the user cancelled", world.window.toasts)
	}
}

func TestTheReplaceQuestionGoesThroughDialogs(t *testing.T) {
	world := newWorld(t, homeWith("api.json"))
	got := ""
	world.c.Pick("Save", filedialog.SaveFile, "api.json", func(path string) { got = path })
	world.step(core.Input{Enter: true, Keys: []ebiten.Key{ebiten.KeyEnter}})
	if len(world.dialogs.asked) != 1 {
		t.Fatalf("questions %+v, want one about replacing", world.dialogs.asked)
	}
	ask := world.dialogs.asked[0]
	if ask.title != "Replace file" || ask.okLabel != "Replace" || !strings.Contains(ask.body, "api.json already exists") {
		t.Fatalf("question %+v", ask)
	}
	if got != "" || !world.c.Shown() {
		t.Fatalf("answered %q or closed before the user said yes", got)
	}
	ask.run()
	if got != "/home/me/api.json" || world.c.Shown() {
		t.Fatalf("after yes: got %q, browser shown %v", got, world.c.Shown())
	}
}

// While a question is asked over the browser, its keys are the question's
// and its path field does not take the keyboard.
func TestTheBrowserKeysWaitForAQuestion(t *testing.T) {
	world := newWorld(t, homeWith("a.json", "b.json"))
	world.c.Pick("Pick", filedialog.OpenFile, "", func(string) {})
	world.step(core.Input{})
	before := world.c.session.model.Selected
	world.dialogs.asked = append(world.dialogs.asked, question{}) // a dialog is open over the browser
	world.step(core.Input{Edits: []core.Edit{core.EditDown}, Keys: []ebiten.Key{ebiten.KeyArrowDown}})
	if world.c.session.model.Selected != before {
		t.Fatalf("Down moved the selection under a question: %d -> %d", before, world.c.session.model.Selected)
	}
}

func TestThePathFieldHasTheKeyboardOnlyWhileNoQuestionIsAsked(t *testing.T) {
	world := newWorld(t, homeWith("a.json"))
	world.c.Pick("Pick", filedialog.OpenFile, "", func(string) {})
	world.dialogs.asked = append(world.dialogs.asked, question{})
	world.step(core.Input{})
	if world.c.session.path.Focused {
		t.Fatal("the path field took the keyboard under a question")
	}
	world.dialogs.asked = nil
	world.step(core.Input{})
	if !world.c.session.path.Focused {
		t.Fatal("the path field has the keyboard while no question is asked")
	}
}

func TestTakingBackAQuestionLeavesALaterBrowserOpen(t *testing.T) {
	world := newWorld(t, homeWith("a.json"))
	cancel := world.c.Pick("First", filedialog.OpenFile, "", func(string) {})
	world.c.Close()
	world.c.Pick("Second", filedialog.OpenFile, "", func(string) {})
	cancel()
	if !world.c.Shown() || world.c.session.title != "Second" {
		t.Fatal("taking back the first question closed the second browser")
	}
}

var _ = browsing.Open
