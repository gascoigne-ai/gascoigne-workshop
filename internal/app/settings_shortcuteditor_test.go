package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/ui/shortcutpage"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

func TestRecordingATakenKeyOffersReplaceAndSavesNothingYet(t *testing.T) {
	a := testApp(t)
	a.shortcutKeys.Start("close-tab")
	a.step(combo(ebiten.KeyT, true, false, false)) // Ctrl+T is "New request"
	if a.shortcutKeys.Recording() != "" {
		t.Fatal("recording should end: the user decides now")
	}
	pending, ok := a.shortcutKeys.Pending("close-tab")
	if !ok || pending.OtherName != "New request" {
		t.Fatalf("pending = %+v", pending)
	}
	if len(a.settings.Shortcuts) != 0 {
		t.Fatalf("shortcuts = %v, want nothing saved before the user replaces", a.settings.Shortcuts)
	}
	if dump := view.Dump(a.shortcutKeys.View(a.ctx, a)); !strings.Contains(dump, "Used by New request") || !strings.Contains(dump, "Replace") {
		t.Fatalf("the row shows no conflict:\n%s", dump)
	}
}

func TestReplaceGivesTheKeyToThisActionAndUnbindsTheOther(t *testing.T) {
	a := testApp(t)
	path := a.SettingsPath()
	a.shortcutKeys.Start("close-tab")
	a.step(combo(ebiten.KeyT, true, false, false))
	a.shortcutKeys.Replace(a, "close-tab")
	if shortcutpage.KeyText(a, "close-tab") != "Ctrl+T" || shortcutpage.KeyText(a, "new-request") != "Unbound" {
		t.Fatalf("close-tab %q, new-request %q", shortcutpage.KeyText(a, "close-tab"), shortcutpage.KeyText(a, "new-request"))
	}
	if a.shortcutKeys.PendingCount() != 0 {
		t.Fatal("the pending choice stayed")
	}
	if saved := mustLoad(t, path); saved.Shortcuts["close-tab"] != "Ctrl+T" || saved.Shortcuts["new-request"] != "" {
		t.Fatalf("saved %v", saved.Shortcuts)
	}
}

func TestRecordingAgainDropsAPendingReplace(t *testing.T) {
	a := testApp(t)
	a.shortcutKeys.Start("close-tab")
	a.step(combo(ebiten.KeyT, true, false, false))
	a.shortcutKeys.Start("close-tab")
	if _, ok := a.shortcutKeys.Pending("close-tab"); ok {
		t.Fatal("a new recording keeps the old pending choice")
	}
}

func TestClearUnbindsAnActionAndResetBringsTheDefaultBack(t *testing.T) {
	a := testApp(t)
	a.shortcutKeys.Clear(a, "close-tab")
	if shortcutpage.KeyText(a, "close-tab") != "Unbound" {
		t.Fatalf("text = %q", shortcutpage.KeyText(a, "close-tab"))
	}
	a.shortcutKeys.Reset(a, "close-tab")
	if shortcutpage.KeyText(a, "close-tab") != "Ctrl+W" {
		t.Fatalf("text = %q", shortcutpage.KeyText(a, "close-tab"))
	}
}

func TestAKeyThatTypesTextIsAWarningNotARefusal(t *testing.T) {
	a := testApp(t)
	a.shortcutKeys.Start("close-tab")
	a.step(combo(ebiten.KeyJ, false, false, false)) // a plain letter types text
	if a.settings.Shortcuts["close-tab"] != "J" {
		t.Fatalf("shortcuts = %v", a.settings.Shortcuts)
	}
	if !strings.Contains(view.Dump(a.shortcutKeys.View(a.ctx, a)), "text") {
		t.Fatal("no warning on the row")
	}
}

func mustLoad(t *testing.T, path string) settings.Settings {
	t.Helper()
	saved, err := settingsDisk.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return saved
}

// shortcutsPageOpen shows the Shortcuts page of the settings dialog.
func shortcutsPageOpen(a *App) {
	a.menu = true
	a.prefs.Cat = "shortcuts"
}

func TestArrowKeysMoveTheShortcutSelectionOnTheShownPage(t *testing.T) {
	a := testApp(t)
	shortcutsPageOpen(a)
	first, second := a.Actions()[0].ID, a.Actions()[1].ID
	a.step(core.Input{Keys: []ebiten.Key{ebiten.KeyArrowDown}})
	if got := a.shortcutKeys.Selected(); got != first {
		t.Fatalf("selected %q after Down, want %q", got, first)
	}
	a.step(core.Input{Keys: []ebiten.Key{ebiten.KeyArrowDown}})
	if got := a.shortcutKeys.Selected(); got != second {
		t.Fatalf("selected %q after Down again, want %q", got, second)
	}
}

// Enter on the selected row records it, and the recorded key reaches the
// settings through Send, as a click on the row does.
func TestEnterRecordsTheSelectedShortcutAndTheKeyIsSaved(t *testing.T) {
	a := testApp(t)
	shortcutsPageOpen(a)
	first := a.Actions()[0].ID
	a.step(core.Input{Keys: []ebiten.Key{ebiten.KeyArrowDown}})
	a.step(core.Input{Enter: true})
	if got := a.shortcutKeys.Recording(); got != first {
		t.Fatalf("recording %q after Enter, want %q", got, first)
	}
	a.step(combo(ebiten.KeyF9, false, false, false))
	if got := a.settings.Shortcuts[first]; got != "F9" {
		t.Fatalf("saved shortcuts %v, want %s=F9", a.settings.Shortcuts, first)
	}
}

// Other pages, the raw settings text and a settings search have no
// shortcut rows on screen, so their keys leave the selection alone.
func TestArrowKeysLeaveTheShortcutSelectionWhenThePageIsNotShown(t *testing.T) {
	cases := map[string]func(a *App){
		"dialog closed": func(a *App) { a.prefs.Cat = "shortcuts" },
		"other page":    func(a *App) { a.menu, a.prefs.Cat = true, "general" },
		"raw text":      func(a *App) { shortcutsPageOpen(a); a.prefs.Raw = true },
		"searching":     func(a *App) { shortcutsPageOpen(a); a.prefs.Search = widget.NewTextField("font") },
	}
	for name, prepare := range cases {
		t.Run(name, func(t *testing.T) {
			a := testApp(t)
			prepare(a)
			a.step(core.Input{Keys: []ebiten.Key{ebiten.KeyArrowDown}})
			if got := a.shortcutKeys.Selected(); got != "" {
				t.Fatalf("selected %q, want none", got)
			}
		})
	}
}
