package ui

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
)

func TestRecordingATakenKeyOffersReplaceAndSavesNothingYet(t *testing.T) {
	a, _ := newTestApp(t)
	a.startRecording("close-tab")
	a.step(combo(ebiten.KeyT, true, false, false)) // Ctrl+T is "New request"
	if a.recording != "" {
		t.Fatal("recording should end: the user decides now")
	}
	pending, ok := a.pendingKeys["close-tab"]
	if !ok || pending.otherName != "New request" {
		t.Fatalf("pending = %+v", a.pendingKeys)
	}
	if len(a.settings.Shortcuts) != 0 {
		t.Fatalf("shortcuts = %v, want nothing saved before the user replaces", a.settings.Shortcuts)
	}
	if dump := vml.Dump(a.shortcutsView()); !strings.Contains(dump, "Used by New request") || !strings.Contains(dump, "Replace") {
		t.Fatalf("the row shows no conflict:\n%s", dump)
	}
}

func TestReplaceGivesTheKeyToThisActionAndUnbindsTheOther(t *testing.T) {
	a, path := newTestApp(t)
	a.startRecording("close-tab")
	a.step(combo(ebiten.KeyT, true, false, false))
	a.replaceBinding("close-tab")
	if a.shortcutText("close-tab") != "Ctrl+T" || a.shortcutText("new-request") != "Unbound" {
		t.Fatalf("close-tab %q, new-request %q", a.shortcutText("close-tab"), a.shortcutText("new-request"))
	}
	if len(a.pendingKeys) != 0 {
		t.Fatal("the pending choice stayed")
	}
	if saved := mustLoad(t, path); saved.Shortcuts["close-tab"] != "Ctrl+T" || saved.Shortcuts["new-request"] != "" {
		t.Fatalf("saved %v", saved.Shortcuts)
	}
}

func TestRecordingAgainDropsAPendingReplace(t *testing.T) {
	a, _ := newTestApp(t)
	a.startRecording("close-tab")
	a.step(combo(ebiten.KeyT, true, false, false))
	a.startRecording("close-tab")
	if _, ok := a.pendingKeys["close-tab"]; ok {
		t.Fatal("a new recording keeps the old pending choice")
	}
}

func TestClearUnbindsAnActionAndResetBringsTheDefaultBack(t *testing.T) {
	a, _ := newTestApp(t)
	a.clearBinding("close-tab")
	if a.shortcutText("close-tab") != "Unbound" {
		t.Fatalf("text = %q", a.shortcutText("close-tab"))
	}
	a.resetBinding("close-tab")
	if a.shortcutText("close-tab") != "Ctrl+W" {
		t.Fatalf("text = %q", a.shortcutText("close-tab"))
	}
}

func TestShortcutSearchMatchesTheActionNameOrTheKey(t *testing.T) {
	tests := []struct {
		query   string
		want    string
		notWant string
	}{
		{"close", "Close tab", "Send request"},
		{"CLOSE", "Close tab", "Send request"},
		{"ctrl+w", "Close tab", "Send request"},
		{"enter", "Send request", "Close tab"},
	}
	for _, test := range tests {
		t.Run(test.query, func(t *testing.T) {
			a, _ := newTestApp(t)
			a.shortcutSearch = vml.NewTextField(test.query)
			dump := vml.Dump(a.shortcutsView())
			if !strings.Contains(dump, test.want) || strings.Contains(dump, test.notWant) {
				t.Fatalf("search %q:\n%s", test.query, dump)
			}
		})
	}
	a, _ := newTestApp(t)
	a.shortcutSearch = vml.NewTextField("zzzz")
	if !strings.Contains(vml.Dump(a.shortcutsView()), "No shortcut matches") {
		t.Fatal("no empty-result note")
	}
}

func TestAKeyThatTypesTextIsAWarningNotARefusal(t *testing.T) {
	a, _ := newTestApp(t)
	a.startRecording("close-tab")
	a.step(combo(ebiten.KeyJ, false, false, false)) // a plain letter types text
	if a.settings.Shortcuts["close-tab"] != "J" {
		t.Fatalf("shortcuts = %v", a.settings.Shortcuts)
	}
	if !strings.Contains(vml.Dump(a.shortcutsView()), "text") {
		t.Fatal("no warning on the row")
	}
}

func mustLoad(t *testing.T, path string) settings.Settings {
	t.Helper()
	saved, err := settings.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return saved
}
