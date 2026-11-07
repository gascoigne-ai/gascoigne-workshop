package ui

import (
	"os"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
)

func setRaw(a *App, text string) {
	a.rawText.Text = text
	a.rawText.Cursor = len([]rune(text))
	a.rawEdited()
}

func TestAValidRawEditAppliesAtOnceAndIsWrittenAsTyped(t *testing.T) {
	a, path := openSettingsPage(t, "appearance")
	a.openRawSettings()
	text := "{\n  \"layout\": \"stacked\"\n}\n"
	setRaw(a, text)
	if a.settings.Layout != "stacked" || len(a.rawProblems) != 0 {
		t.Fatalf("Layout %q problems %+v", a.settings.Layout, a.rawProblems)
	}
	if data, _ := os.ReadFile(path); string(data) != text {
		t.Fatalf("file = %q, want the text as typed", data)
	}
}

func TestARawEditWithProblemsLeavesTheLastGoodSettingsAndTheFile(t *testing.T) {
	a, path := openSettingsPage(t, "appearance")
	a.openRawSettings()
	setRaw(a, `{"layout": "stacked"}`)
	good, _ := os.ReadFile(path)
	for _, bad := range []string{`{"layout": `, `{"layout": "diagonal"}`, `{"colour": 1}`, `[1]`} {
		setRaw(a, bad)
		if a.settings.Layout != "stacked" || len(a.rawProblems) == 0 {
			t.Errorf("%q: Layout %q problems %v", bad, a.settings.Layout, a.rawProblems)
		}
		if data, _ := os.ReadFile(path); string(data) != string(good) {
			t.Errorf("%q: the file changed to %q", bad, data)
		}
	}
	if dump := vml.Dump(a.menuView()); !strings.Contains(dump, "JSON") {
		t.Fatalf("raw view:\n%s", dump)
	}
}

func TestTheRawViewMarksKeysThatHaveProblems(t *testing.T) {
	a, _ := openSettingsPage(t, "appearance")
	a.openRawSettings()
	setRaw(a, "{\n  \"colour\": 1,\n  \"layout\": \"side\"\n}")
	marked := func(line string) []string {
		var out []string
		for _, span := range a.rawText.Styler(line) {
			if span.Style == w.Error {
				out = append(out, span.Text)
			}
		}
		return out
	}
	if got := marked(`  "colour": 1,`); len(got) != 1 || got[0] != `"colour"` {
		t.Fatalf("marked on the bad line: %q", got)
	}
	if got := marked(`  "layout": "side"`); len(got) != 0 {
		t.Fatalf("marked on a good line: %q", got)
	}
}

func TestOpeningTheRawViewShowsWhatTheFormChanged(t *testing.T) {
	a, _ := openSettingsPage(t, "appearance")
	a.settingsSearch = vml.NewTextField("request and response")
	clickText(t, a, "Stacked", 0)
	a.openRawSettings()
	if !strings.Contains(a.rawText.Text, `"layout": "stacked"`) {
		t.Fatalf("raw text = %q", a.rawText.Text)
	}
	a.closeRawSettings()
	if a.rawMode {
		t.Fatal("still in raw mode")
	}
}

func TestAChangeOnDiskShowsUpInTheSettingsAndInTheRawView(t *testing.T) {
	a, path := openSettingsPage(t, "appearance")
	a.openRawSettings()
	if err := os.WriteFile(path, []byte(`{"corners": "square"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	a.pollSettingsFile()
	if a.settings.Corners != "square" {
		t.Fatalf("Corners = %q after the file changed", a.settings.Corners)
	}
	if a.rawText.Text != `{"corners": "square"}` {
		t.Fatalf("raw text = %q", a.rawText.Text)
	}
}

func TestABadFileOnDiskKeepsTheSettingsAndSaysSo(t *testing.T) {
	a, path := openSettingsPage(t, "appearance")
	a.saveSettings(func(s *settings.Settings) { s.Corners = "round" })
	if err := os.WriteFile(path, []byte(`{"corners": `), 0o600); err != nil {
		t.Fatal(err)
	}
	a.pollSettingsFile()
	if a.settings.Corners != "round" {
		t.Fatalf("Corners = %q, want the last good value", a.settings.Corners)
	}
	found := false
	for _, toast := range a.toasts {
		found = found || (toast.isErr && strings.Contains(toast.text, "settings.json"))
	}
	if !found {
		t.Fatalf("no toast about the file; toasts %+v", a.toasts)
	}
}

func TestOurOwnWritesAreNotReloadedAsOutsideChanges(t *testing.T) {
	a, _ := openSettingsPage(t, "appearance")
	a.saveSettings(func(s *settings.Settings) { s.Corners = "round" })
	a.settings.Corners = "square" // a change not yet saved must survive a poll
	a.pollSettingsFile()
	if a.settings.Corners != "square" {
		t.Fatal("the poll reloaded the file we had just written")
	}
}

func TestCtrlSpaceInTheRawViewOffersKeysAndValues(t *testing.T) {
	a, _ := openSettingsPage(t, "appearance")
	a.openRawSettings()
	a.rawText = vml.NewTextArea(`{"la`)
	a.rawText.Cursor = 4
	a.rawText.Focused = true
	a.completeVariable()
	if a.ctxMenu == nil || len(a.ctxMenu.items) != 1 || a.ctxMenu.items[0].label != "layout" {
		t.Fatalf("menu = %+v", a.ctxMenu)
	}
	a.ctxMenu.items[0].run()
	if a.rawText.Text != `{"layout"` {
		t.Fatalf("text after completing = %q", a.rawText.Text)
	}

	a.closeMenu()
	a.rawText = vml.NewTextArea(`{"layout": "st`)
	a.rawText.Cursor = len([]rune(a.rawText.Text))
	a.rawText.Focused = true
	a.completeVariable()
	if a.ctxMenu == nil || a.ctxMenu.items[0].label != "stacked" {
		t.Fatalf("menu = %+v", a.ctxMenu)
	}
}
