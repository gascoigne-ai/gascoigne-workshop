package settingsdialog

import (
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/keymap"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/prefs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/storage"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog/shortcutpage"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/fonts"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// fakeApp answers every Deps interface of the component and records what the
// component asked of the application.
type fakeApp struct {
	files    *memoryStore // the settings files the component reads and writes
	current  settings.Settings
	themeErr error
	scale    int
	applied  []settings.Settings
	toasts   []string
	status   []string
	stale    int
	redraws  int
	dialogs  []widget.DialogState
	closed   int // dialogs the component closed
	picks    []*pick
	colors   map[w.Token]color.RGBA
	menus    [][]widget.MenuItem
}

type pick struct {
	title     string
	done      func(path string)
	cancelled bool
}

func (f *fakeApp) Settings() settings.Settings { return f.current }
func (f *fakeApp) Edit(edit func(*settings.Settings)) error {
	edit(&f.current)
	return f.themeErr
}
func (f *fakeApp) Apply(current settings.Settings) {
	f.current = current
	f.applied = append(f.applied, current)
}
func (f *fakeApp) SetScale(percent int)                       { f.scale = percent }
func (f *fakeApp) Size() (int, int)                           { return 1280, 800 }
func (f *fakeApp) Pointer() (int, int)                        { return 400, 300 }
func (f *fakeApp) SetStatus(text string)                      { f.status = append(f.status, text) }
func (f *fakeApp) Stale()                                     { f.stale++ }
func (f *fakeApp) Redraw()                                    { f.redraws++ }
func (f *fakeApp) Dispatch(run func())                        { run() }
func (f *fakeApp) HostOS() string                             { return "linux" }
func (f *fakeApp) CopyText(string)                            {}
func (f *fakeApp) SetFonts(_, _ w.FontData) error             { return nil }
func (f *fakeApp) Toast(text string, _ severity.Level)        { f.toasts = append(f.toasts, text) }
func (f *fakeApp) OpenMenu(_, _ int, items []widget.MenuItem) { f.menus = append(f.menus, items) }
func (f *fakeApp) Color(token w.Token) color.RGBA             { return f.colors[token] }
func (f *fakeApp) Open(state widget.DialogState) func() {
	f.dialogs = append(f.dialogs, state)
	return func() { f.closed++ }
}
func (f *fakeApp) Pick(title string, done func(path string)) func() {
	request := &pick{title: title, done: done}
	f.picks = append(f.picks, request)
	return func() { request.cancelled = true }
}
func (f *fakeApp) Keymap() keymap.Keymap {
	commands := []keymap.Command{
		{ID: "send", Name: "Send request", Default: keymap.Chord{Key: int(ebiten.KeyEnter), Ctrl: true}},
		{ID: "focus-url", Name: "Focus URL", Default: keymap.Chord{Key: int(ebiten.KeyL), Ctrl: true}},
	}
	return keymap.Resolve(f.current.Shortcuts, commands, shortcutpage.ParseChord)
}

// newComponent is a component over an in-memory settings file at a path in a
// temp dir, with the application in its default settings.
func newComponent(t *testing.T, access storage.Access) (*Component, *fakeApp, string) {
	t.Helper()
	app := &fakeApp{current: settings.Default(), colors: map[w.Token]color.RGBA{}, files: newMemoryStore()}
	path := filepath.Join(t.TempDir(), "settings.json")
	component := New(Deps{
		Settings: app, Window: app, Files: app, Dialogs: app, Bus: app, Looks: app, Keys: app,
		Path: path, Store: app.files, Access: access, Fonts: fonts.System{},
	})
	return component, app, path
}

// press runs the button of the open question that has label.
func press(t *testing.T, app *fakeApp, label string) {
	t.Helper()
	if len(app.dialogs) == 0 {
		t.Fatalf("no question was asked")
	}
	for _, button := range app.dialogs[len(app.dialogs)-1].Buttons {
		if button.Label == label {
			button.Run()
			return
		}
	}
	t.Fatalf("no %q button in %+v", label, app.dialogs[len(app.dialogs)-1].Buttons)
}

// unsavedComponent is a component with the dialog open and a Layout change
// applied but not written, auto-save being off.
func unsavedComponent(t *testing.T) (*Component, *fakeApp, string) {
	t.Helper()
	component, app, path := newComponent(t, storage.Writable)
	component.Show()
	component.Change(func(current *settings.Settings) { current.AutoSave = false })
	component.Edit(func(current *settings.Settings) { current.Layout = "side" })
	if !component.Unsaved() {
		t.Fatal("the change was not marked unsaved")
	}
	return component, app, path
}

func TestClosingWithUnsavedChangesAsksWhatToDo(t *testing.T) {
	t.Run("cancel keeps the dialog and the changes", func(t *testing.T) {
		component, app, _ := unsavedComponent(t)
		component.Close()
		if len(app.dialogs) != 1 || !component.Open() {
			t.Fatalf("questions %d, open %v: closing must ask first and keep the dialog", len(app.dialogs), component.Open())
		}
		press(t, app, "Cancel")
		if app.closed != 1 || !component.Open() || app.current.Layout != "side" || !component.Unsaved() {
			t.Fatalf("closed %d open %v layout %q unsaved %v", app.closed, component.Open(), app.current.Layout, component.Unsaved())
		}
	})
	t.Run("save writes and closes", func(t *testing.T) {
		component, app, path := unsavedComponent(t)
		component.Close()
		press(t, app, "Save")
		if component.Open() || component.Unsaved() || !strings.Contains(app.files.text(path), `"layout": "side"`) {
			t.Fatalf("open %v unsaved %v file %q", component.Open(), component.Unsaved(), app.files.text(path))
		}
	})
	t.Run("discard reverts to what was stored and closes", func(t *testing.T) {
		component, app, path := unsavedComponent(t)
		component.Close()
		press(t, app, "Discard")
		if component.Open() || component.Unsaved() || app.current.Layout != settings.Default().Layout || strings.Contains(app.files.text(path), "side") {
			t.Fatalf("open %v unsaved %v layout %q file %q", component.Open(), component.Unsaved(), app.current.Layout, app.files.text(path))
		}
	})
}

func TestACleanDialogClosesWithoutAsking(t *testing.T) {
	component, app, _ := newComponent(t, storage.Writable)
	component.Show()
	component.Close()
	if component.Open() || len(app.dialogs) != 0 {
		t.Fatalf("open %v, %d questions", component.Open(), len(app.dialogs))
	}
}

func TestToggleOpensAndClosesTheDialog(t *testing.T) {
	component, _, _ := newComponent(t, storage.Writable)
	component.Toggle()
	if !component.Open() {
		t.Fatal("Toggle did not open the dialog")
	}
	component.Toggle()
	if component.Open() {
		t.Fatal("Toggle did not close the dialog")
	}
}

func TestPersistWritesTheFileAndAnOwnWriteIsNotReloaded(t *testing.T) {
	component, app, path := unsavedComponent(t)
	component.Send(prefs.Persist{})
	if component.Unsaved() || !strings.Contains(app.files.text(path), `"layout": "side"`) {
		t.Fatalf("unsaved %v file %q", component.Unsaved(), app.files.text(path))
	}
	component.Poll()
	if len(app.applied) != 0 {
		t.Fatalf("the file just written was applied again: %+v", app.applied)
	}
	app.files.write(path, `{"autoSave": false, "corners": "square"}`)
	component.Poll()
	if len(app.applied) != 1 || app.current.Corners != "square" {
		t.Fatalf("applied %d, corners %q: an outside change must be picked up", len(app.applied), app.current.Corners)
	}
}

func TestReplaceAppliesTheSettingsAndReloadsThePages(t *testing.T) {
	component, app, _ := newComponent(t, storage.Writable)
	token := w.Token("surface")
	app.colors[token] = color.RGBA{R: 0x11, G: 0x11, B: 0x11, A: 0xff}
	if got := component.Colors.Input(component, token).Text; got != "#111111" {
		t.Fatalf("color field %q before", got)
	}
	app.colors[token] = color.RGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xff}
	replaced := settings.Default()
	replaced.Layout = "side"
	component.Send(prefs.Replace{Settings: replaced})
	if len(app.applied) != 1 || app.current.Layout != "side" {
		t.Fatalf("applied %d, layout %q", len(app.applied), app.current.Layout)
	}
	if got := component.Colors.Input(component, token).Text; got != "#222222" {
		t.Fatalf("color field %q after, want it rebuilt from the colors in effect", got)
	}
}

func TestEveryMessageMarksTheViewStale(t *testing.T) {
	component, app, _ := newComponent(t, storage.Writable)
	component.Send(prefs.MarkDirty{})
	component.Send(prefs.SetStatus{Text: "x"})
	if app.stale != 2 || app.redraws != 1 || len(app.status) != 1 {
		t.Fatalf("stale %d redraws %d status %q", app.stale, app.redraws, app.status)
	}
}

func TestDraggingTheScaleSliderSetsTheScaleWithoutWriting(t *testing.T) {
	component, app, path := newComponent(t, storage.Writable)
	component.Send(prefs.ScaleLive{Percent: 150})
	if app.scale != 150 || app.files.text(path) != "" {
		t.Fatalf("scale %d, file %q", app.scale, app.files.text(path))
	}
}

func TestACloseMessageClosesLikeTheCloseButton(t *testing.T) {
	component, app, _ := unsavedComponent(t)
	component.Send(prefs.Close{})
	if len(app.dialogs) != 1 || !component.Open() {
		t.Fatalf("questions %d open %v: the close button asks about unsaved changes too", len(app.dialogs), component.Open())
	}
}

func TestANewerFileIsNeverWritten(t *testing.T) {
	component, app, path := newComponent(t, storage.ReadOnly)
	component.Edit(func(current *settings.Settings) { current.Layout = "side" })
	component.Send(prefs.Persist{})
	if len(app.toasts) == 0 || !strings.Contains(app.toasts[0], "newer version") {
		t.Fatalf("toasts %q, want the refusal named", app.toasts)
	}
	if app.files.has(path) {
		t.Fatalf("the file was written: %q", app.files.text(path))
	}
}

func TestAFailedThemeStopsTheWrite(t *testing.T) {
	component, app, path := newComponent(t, storage.Writable)
	app.themeErr = os.ErrInvalid
	component.Edit(func(current *settings.Settings) { current.Layout = "side" })
	if len(app.status) != 1 || app.files.text(path) != "" {
		t.Fatalf("status %q file %q, want the error shown and nothing written", app.status, app.files.text(path))
	}
}

func TestAskingForAFileTakesBackThePreviousQuestion(t *testing.T) {
	component, app, _ := newComponent(t, storage.Writable)
	component.Show()
	component.Send(prefs.PickFile{Title: "first", Done: func(string) {}})
	component.Send(prefs.PickFile{Title: "second", Done: func(string) {}})
	if len(app.picks) != 2 || !app.picks[0].cancelled || app.picks[1].cancelled {
		t.Fatalf("picks %+v", app.picks)
	}
	component.Close()
	if !app.picks[1].cancelled {
		t.Fatal("closing the dialog left its file question open")
	}
}

func TestMenuItemsOpenAsAMenuOfTheWindow(t *testing.T) {
	component, app, _ := newComponent(t, storage.Writable)
	ran := false
	component.Send(prefs.OpenMenu{X: 5, Y: 6, Items: []prefs.MenuItem{{Label: "a", Hint: "h", Run: func() { ran = true }}}})
	if len(app.menus) != 1 || app.menus[0][0].Label != "a" || app.menus[0][0].Hint != "h" {
		t.Fatalf("menus %+v", app.menus)
	}
	app.menus[0][0].Run()
	if !ran {
		t.Fatal("the item did not run the page's callback")
	}
}

func TestTheSettingsCommandOpensTheDialog(t *testing.T) {
	component, _, _ := newComponent(t, storage.Writable)
	commands := component.Commands()
	if len(commands) != 1 || commands[0].ID != "settings" || commands[0].Name != "Settings" {
		t.Fatalf("commands %+v", commands)
	}
	if want := (keymap.Chord{Key: int(ebiten.KeyComma), Ctrl: true}); commands[0].Default != want {
		t.Fatalf("default %+v, want %+v", commands[0].Default, want)
	}
	commands[0].Run()
	if !component.Open() {
		t.Fatal("the command did not open the dialog")
	}
}

func TestKeysBelongToTheRecordingWhileAKeyIsRecorded(t *testing.T) {
	component, app, _ := newComponent(t, storage.Writable)
	component.Shortcuts.Start("send")
	if !component.Recording() {
		t.Fatal("Start did not begin recording")
	}
	if !component.Keys(w.Input{Keys: []ebiten.Key{ebiten.KeyQ}, Ctrl: true}) {
		t.Fatal("the recording did not swallow the keys")
	}
	if component.Recording() || app.current.Shortcuts["send"] != "Ctrl+Q" {
		t.Fatalf("recording %v shortcuts %v, want Ctrl+Q recorded for send", component.Recording(), app.current.Shortcuts)
	}
}

func TestKeysMoveThroughTheShortcutsPageOnlyWhileItIsShown(t *testing.T) {
	component, _, _ := newComponent(t, storage.Writable)
	down := w.Input{Keys: []ebiten.Key{ebiten.KeyArrowDown}}
	if component.Keys(down) {
		t.Fatal("a closed dialog took the keys")
	}
	component.Show()
	component.Shell.Cat = "appearance"
	if component.Keys(down) {
		t.Fatal("another page took the keys")
	}
	component.Shell.Cat = "shortcuts"
	if !component.Keys(down) || component.Shortcuts.Selected() != "send" {
		t.Fatalf("the shortcuts page did not take the arrow key (selected %q)", component.Shortcuts.Selected())
	}
}
