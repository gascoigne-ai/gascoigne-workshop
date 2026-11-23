package app

import (
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// wantActionIDs is the order the Shortcuts page lists the actions in.
var wantActionIDs = []string{
	"send", "focus-url", "new-request", "close-tab", "next-tab", "prev-tab",
	"duplicate-row", "move-row-up", "move-row-down", "delete-row", "complete-variable",
	"save", "open-workspace", "close-collection", "find", "settings",
}

func TestShortcutsPageListsTheActionsInDisplayOrder(t *testing.T) {
	a := testApp(t)
	if got := a.over.Settings.ActionIDs(); !slices.Equal(got, wantActionIDs) {
		t.Fatalf("ActionIDs = %v, want %v", got, wantActionIDs)
	}
	dump := view.Dump(a.over.Settings.Shortcuts.View(a.ctx, a.over.Settings))
	last := -1
	for _, name := range []string{"Send request", "Focus URL", "New request", "Close tab", "Next tab", "Previous tab", "Duplicate row", "Move row up", "Move row down", "Delete row", "Complete variable", "Save collection", "Open collection folder", "Close collection", "Find in response", "Settings"} {
		at := strings.Index(dump, name)
		if at <= last {
			t.Fatalf("%q is missing or out of order in:\n%s", name, dump)
		}
		last = at
	}
}

// Two actions on one chord both run: a hand-edited settings file can give
// them the same key, and it has always worked that way.
func TestEveryActionOnAChordRuns(t *testing.T) {
	a := testApp(t)
	a.settings.Shortcuts = map[string]string{"settings": "Ctrl+F"}
	a.Step(combo(ebiten.KeyF, true, false, false))
	if !a.over.Settings.Open() {
		t.Fatal("the settings dialog did not open on its rebound chord")
	}
	if !a.resp.Look(a.cur().Tab).FindOpen {
		t.Fatal("find did not open on its default chord, which settings shares now")
	}
}

func TestShortcutsAreOffWhileAnOverlayIsOpen(t *testing.T) {
	a := testApp(t)
	a.confirm("Delete request?", "This cannot be undone.", "Delete", func() {})
	a.Step(core.Input{})
	tabs := len(a.tabs.Tabs())
	a.Step(combo(ebiten.KeyT, true, false, false))
	if len(a.tabs.Tabs()) != tabs {
		t.Fatal("Ctrl+T must not open a tab behind a dialog")
	}
}

// A claim for Undo in text keeps the table's undo from running, but a user
// binding on the same keys still fires.
func TestAnUndoClaimInTextStillRunsAUserBindingOnIt(t *testing.T) {
	a := testApp(t)
	a.settings.Shortcuts = map[string]string{"new-request": "Ctrl+Z"}
	tabs := len(a.tabs.Tabs())
	a.editor.URLBar.Field(a.cur().Tab).Focused = true
	a.Step(core.Input{Keys: []ebiten.Key{ebiten.KeyZ}, Ctrl: true, Undo: true})
	if len(a.tabs.Tabs()) != tabs+1 {
		t.Fatalf("tabs %d, want %d: the binding on Ctrl+Z must run", len(a.tabs.Tabs()), tabs+1)
	}
}

func TestTableUndoRunsAfterAUserBindingOnCtrlZ(t *testing.T) {
	a := testApp(t)
	a.settings.Shortcuts = map[string]string{"settings": "Ctrl+Z"}
	req := a.cur().Req
	click(a, 739, paramsRow0Y)
	if len(req.Params) != 2 {
		t.Fatal("test setup")
	}
	a.Step(core.Input{Keys: []ebiten.Key{ebiten.KeyZ}, Ctrl: true, Undo: true})
	if !a.over.Settings.Open() || len(req.Params) != 3 {
		t.Fatalf("menu %v params %d: both the binding and the table undo must run", a.over.Settings.Open(), len(req.Params))
	}
}

func TestHintNamesTheEffectiveKeyAndNothingWhenUnbound(t *testing.T) {
	a := testApp(t)
	if got := a.hint("Close tab", "close-tab"); got != "Close tab (Ctrl+W)" {
		t.Fatalf("default hint %q", got)
	}
	a.settings.Shortcuts = map[string]string{"close-tab": "Ctrl+Q"}
	if got := a.hint("Close tab", "close-tab"); got != "Close tab (Ctrl+Q)" {
		t.Fatalf("rebound hint %q", got)
	}
	a.settings.Shortcuts = map[string]string{"close-tab": ""}
	if got := a.hint("Close tab", "close-tab"); got != "Close tab" {
		t.Fatalf("unbound hint %q", got)
	}
}

func TestSettingsCompletionOffersEveryActionID(t *testing.T) {
	a := testApp(t)
	got := settings.Complete(`{"shortcuts": {"`, len(`{"shortcuts": {"`), a.over.Settings.ActionIDs())
	for _, id := range wantActionIDs {
		if !slices.ContainsFunc(got, func(suggestion settings.Completion) bool { return strings.Contains(suggestion.Insert, id) }) {
			t.Errorf("completion lacks %q: %+v", id, got)
		}
	}
}
