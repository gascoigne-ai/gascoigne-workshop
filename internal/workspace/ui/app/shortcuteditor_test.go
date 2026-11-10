package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/adapter/settingsfile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/ui/shortcutpage"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

func TestRecordingATakenKeyOffersReplaceAndSavesNothingYet(t *testing.T) {
	a, _ := newTestApp(t)
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
	if dump := view.Dump(a.shortcutKeys.View(a)); !strings.Contains(dump, "Used by New request") || !strings.Contains(dump, "Replace") {
		t.Fatalf("the row shows no conflict:\n%s", dump)
	}
}

func TestReplaceGivesTheKeyToThisActionAndUnbindsTheOther(t *testing.T) {
	a, path := newTestApp(t)
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
	a, _ := newTestApp(t)
	a.shortcutKeys.Start("close-tab")
	a.step(combo(ebiten.KeyT, true, false, false))
	a.shortcutKeys.Start("close-tab")
	if _, ok := a.shortcutKeys.Pending("close-tab"); ok {
		t.Fatal("a new recording keeps the old pending choice")
	}
}

func TestClearUnbindsAnActionAndResetBringsTheDefaultBack(t *testing.T) {
	a, _ := newTestApp(t)
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
	a, _ := newTestApp(t)
	a.shortcutKeys.Start("close-tab")
	a.step(combo(ebiten.KeyJ, false, false, false)) // a plain letter types text
	if a.settings.Shortcuts["close-tab"] != "J" {
		t.Fatalf("shortcuts = %v", a.settings.Shortcuts)
	}
	if !strings.Contains(view.Dump(a.shortcutKeys.View(a)), "text") {
		t.Fatal("no warning on the row")
	}
}

func mustLoad(t *testing.T, path string) settings.Settings {
	t.Helper()
	saved, err := settingsfile.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return saved
}
