package ui

import (
	"os"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

func fileText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func setLayout(a *App, layout string) {
	a.changeSettings(func(s *settings.Settings) { s.Layout = layout })
}

func TestSettingsAreSavedAutomaticallyByDefault(t *testing.T) {
	a, path := newSettingsTestApp(t)
	if !a.settings.AutoSave {
		t.Fatal("auto-save is off by default")
	}
	setLayout(a, "side")
	if !strings.Contains(fileText(t, path), `"layout": "side"`) || a.settingsUnsaved {
		t.Fatalf("file %q unsaved %v", fileText(t, path), a.settingsUnsaved)
	}
}

func TestWithAutoSaveOffChangesApplyAtOnceButAreWrittenOnlyOnSave(t *testing.T) {
	a, path := newSettingsTestApp(t)
	a.changeSettings(func(s *settings.Settings) { s.AutoSave = false })
	if !strings.Contains(fileText(t, path), `"autoSave": false`) {
		t.Fatalf("turning auto-save off must itself be stored, file %q", fileText(t, path))
	}
	setLayout(a, "side")
	if a.settings.Layout != "side" {
		t.Fatal("the change did not apply")
	}
	if strings.Contains(fileText(t, path), "layout") || !a.settingsUnsaved {
		t.Fatalf("the change was written (file %q) or not marked unsaved (%v)", fileText(t, path), a.settingsUnsaved)
	}
	a.prefs.Save(a)
	if !strings.Contains(fileText(t, path), `"layout": "side"`) || a.settingsUnsaved {
		t.Fatalf("after Save: file %q unsaved %v", fileText(t, path), a.settingsUnsaved)
	}
}

func TestTurningAutoSaveBackOnWritesWhatWasPending(t *testing.T) {
	a, path := newSettingsTestApp(t)
	a.changeSettings(func(s *settings.Settings) { s.AutoSave = false })
	setLayout(a, "side")
	a.changeSettings(func(s *settings.Settings) { s.AutoSave = true })
	if !strings.Contains(fileText(t, path), `"layout": "side"`) || a.settingsUnsaved {
		t.Fatalf("file %q unsaved %v", fileText(t, path), a.settingsUnsaved)
	}
}

func TestClosingWithUnsavedChangesAsksWhatToDo(t *testing.T) {
	setup := func(t *testing.T) (*App, string) {
		a, path := openSettingsMenu(t)
		a.changeSettings(func(s *settings.Settings) { s.AutoSave = false })
		setLayout(a, "side")
		a.closeSettings()
		if a.dialog == nil || !a.menu {
			t.Fatal("closing with unsaved changes must ask first and keep the settings open")
		}
		return a, path
	}
	press := func(a *App, label string) {
		for _, b := range a.dialog.Buttons {
			if b.Label == label {
				b.Run()
				return
			}
		}
		t.Fatalf("no %q button in %+v", label, a.dialog.Buttons)
	}
	t.Run("cancel keeps the dialog and the changes", func(t *testing.T) {
		a, _ := setup(t)
		press(a, "Cancel")
		if a.dialog != nil || !a.menu || a.settings.Layout != "side" {
			t.Fatalf("dialog %v menu %v layout %q", a.dialog != nil, a.menu, a.settings.Layout)
		}
	})
	t.Run("save writes and closes", func(t *testing.T) {
		a, path := setup(t)
		press(a, "Save")
		if a.menu || !strings.Contains(fileText(t, path), `"layout": "side"`) {
			t.Fatalf("menu open %v, file %q", a.menu, fileText(t, path))
		}
	})
	t.Run("discard reverts to what was stored and closes", func(t *testing.T) {
		a, path := setup(t)
		press(a, "Discard")
		if a.menu || a.settings.Layout != "stacked" || strings.Contains(fileText(t, path), "side") || a.settingsUnsaved {
			t.Fatalf("menu %v layout %q file %q unsaved %v", a.menu, a.settings.Layout, fileText(t, path), a.settingsUnsaved)
		}
	})
}

func TestACleanDialogClosesWithoutAsking(t *testing.T) {
	a, _ := openSettingsMenu(t)
	a.closeSettings()
	if a.menu || a.dialog != nil {
		t.Fatalf("menu %v dialog %v", a.menu, a.dialog != nil)
	}
}

func TestAnOutsideChangeOfTheFileDoesNotOverwriteUnsavedChanges(t *testing.T) {
	a, path := newSettingsTestApp(t)
	a.changeSettings(func(s *settings.Settings) { s.AutoSave = false })
	setLayout(a, "side")
	if err := os.WriteFile(path, []byte(`{"autoSave": false, "corners": "square"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	a.prefs.Poll(a)
	if a.settings.Layout != "side" || a.settings.Corners == "square" {
		t.Fatalf("layout %q corners %q: the reload clobbered the unsaved changes", a.settings.Layout, a.settings.Corners)
	}
}

func TestTheDialogHasASaveButtonAndTheAutoSaveSetting(t *testing.T) {
	a, _ := openSettingsMenu(t)
	if dump := view.Dump(a.menuView()); strings.Contains(dump, `text="Save"`) {
		t.Fatalf("a Save button while auto-save is on and nothing is pending:\n%.800s", dump)
	}
	a.changeSettings(func(s *settings.Settings) { s.AutoSave = false })
	if dump := view.Dump(a.menuView()); !strings.Contains(dump, `text="Save"`) {
		t.Fatalf("no Save button in the title bar with auto-save off:\n%.800s", dump)
	}
	a.prefs.Cat = "general"
	a.step(core.Input{})
	page := view.Dump(a.menuView())
	for _, want := range []string{"Save changes automatically", "written to settings.json at once"} {
		if !strings.Contains(page, want) {
			t.Errorf("the General page lacks %q", want)
		}
	}
}
