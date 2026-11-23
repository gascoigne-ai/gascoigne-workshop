package app

import (
	"os"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

func setLayout(app *App, layout string) {
	app.changeSettings(func(prefs *settings.Settings) { prefs.Layout = layout })
}

func TestSettingsAreSavedAutomaticallyByDefault(t *testing.T) {
	a := testApp(t, withSettings(defaultLayout))
	path := a.SettingsPath()
	if !a.settings.AutoSave {
		t.Fatal("auto-save is off by default")
	}
	setLayout(a, "side")
	if !strings.Contains(fileText(t, path), `"layout": "side"`) || a.prefs.Unsaved() {
		t.Fatalf("file %q unsaved %v", fileText(t, path), a.prefs.Unsaved())
	}
}

func TestWithAutoSaveOffChangesApplyAtOnceButAreWrittenOnlyOnSave(t *testing.T) {
	a := testApp(t, withSettings(defaultLayout))
	path := a.SettingsPath()
	a.changeSettings(func(prefs *settings.Settings) { prefs.AutoSave = false })
	if !strings.Contains(fileText(t, path), `"autoSave": false`) {
		t.Fatalf("turning auto-save off must itself be stored, file %q", fileText(t, path))
	}
	setLayout(a, "side")
	if a.settings.Layout != "side" {
		t.Fatal("the change did not apply")
	}
	if strings.Contains(fileText(t, path), "layout") || !a.prefs.Unsaved() {
		t.Fatalf("the change was written (file %q) or not marked unsaved (%v)", fileText(t, path), a.prefs.Unsaved())
	}
	a.prefs.Shell.Save(a.prefs)
	if !strings.Contains(fileText(t, path), `"layout": "side"`) || a.prefs.Unsaved() {
		t.Fatalf("after Save: file %q unsaved %v", fileText(t, path), a.prefs.Unsaved())
	}
}

func TestTurningAutoSaveBackOnWritesWhatWasPending(t *testing.T) {
	a := testApp(t, withSettings(defaultLayout))
	path := a.SettingsPath()
	a.changeSettings(func(prefs *settings.Settings) { prefs.AutoSave = false })
	setLayout(a, "side")
	a.changeSettings(func(prefs *settings.Settings) { prefs.AutoSave = true })
	if !strings.Contains(fileText(t, path), `"layout": "side"`) || a.prefs.Unsaved() {
		t.Fatalf("file %q unsaved %v", fileText(t, path), a.prefs.Unsaved())
	}
}

func TestClosingWithUnsavedChangesAsksWhatToDo(t *testing.T) {
	setup := func(t *testing.T) (*App, string) {
		a, path := openSettingsMenu(t)
		a.changeSettings(func(prefs *settings.Settings) { prefs.AutoSave = false })
		setLayout(a, "side")
		a.prefs.Close()
		if a.dialog == nil || !a.prefs.Open() {
			t.Fatal("closing with unsaved changes must ask first and keep the settings open")
		}
		return a, path
	}
	press := func(app *App, label string) {
		for _, b := range app.dialog.Buttons {
			if b.Label == label {
				b.Run()
				return
			}
		}
		t.Fatalf("no %q button in %+v", label, app.dialog.Buttons)
	}
	t.Run("cancel keeps the dialog and the changes", func(t *testing.T) {
		a, _ := setup(t)
		press(a, "Cancel")
		if a.dialog != nil || !a.prefs.Open() || a.settings.Layout != "side" {
			t.Fatalf("dialog %v menu %v layout %q", a.dialog != nil, a.prefs.Open(), a.settings.Layout)
		}
	})
	t.Run("save writes and closes", func(t *testing.T) {
		a, path := setup(t)
		press(a, "Save")
		if a.prefs.Open() || !strings.Contains(fileText(t, path), `"layout": "side"`) {
			t.Fatalf("menu open %v, file %q", a.prefs.Open(), fileText(t, path))
		}
	})
	t.Run("discard reverts to what was stored and closes", func(t *testing.T) {
		a, path := setup(t)
		press(a, "Discard")
		if a.prefs.Open() || a.settings.Layout != "stacked" || strings.Contains(fileText(t, path), "side") || a.prefs.Unsaved() {
			t.Fatalf("menu %v layout %q file %q unsaved %v", a.prefs.Open(), a.settings.Layout, fileText(t, path), a.prefs.Unsaved())
		}
	})
}

func TestACleanDialogClosesWithoutAsking(t *testing.T) {
	a, _ := openSettingsMenu(t)
	a.prefs.Close()
	if a.prefs.Open() || a.dialog != nil {
		t.Fatalf("menu %v dialog %v", a.prefs.Open(), a.dialog != nil)
	}
}

func TestAnOutsideChangeOfTheFileDoesNotOverwriteUnsavedChanges(t *testing.T) {
	a := testApp(t, withSettings(defaultLayout))
	path := a.SettingsPath()
	a.changeSettings(func(prefs *settings.Settings) { prefs.AutoSave = false })
	setLayout(a, "side")
	if err := os.WriteFile(path, []byte(`{"autoSave": false, "corners": "square"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	a.prefs.Poll()
	if a.settings.Layout != "side" || a.settings.Corners == "square" {
		t.Fatalf("layout %q corners %q: the reload clobbered the unsaved changes", a.settings.Layout, a.settings.Corners)
	}
}

func TestTheDialogHasASaveButtonAndTheAutoSaveSetting(t *testing.T) {
	a, _ := openSettingsMenu(t)
	if dump := view.Dump(a.prefs.View(a.ctx)); strings.Contains(dump, `text="Save"`) {
		t.Fatalf("a Save button while auto-save is on and nothing is pending:\n%.800s", dump)
	}
	a.changeSettings(func(prefs *settings.Settings) { prefs.AutoSave = false })
	if dump := view.Dump(a.prefs.View(a.ctx)); !strings.Contains(dump, `text="Save"`) {
		t.Fatalf("no Save button in the title bar with auto-save off:\n%.800s", dump)
	}
	a.prefs.Shell.Cat = "general"
	a.Step(core.Input{})
	page := view.Dump(a.prefs.View(a.ctx))
	for _, want := range []string{"Save changes automatically", "written to settings.json at once"} {
		if !strings.Contains(page, want) {
			t.Errorf("the General page lacks %q", want)
		}
	}
}
