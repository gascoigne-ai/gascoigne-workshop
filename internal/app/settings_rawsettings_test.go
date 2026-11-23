package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/hajimehoshi/ebiten/v2"
	"os"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

func setRaw(app *App, text string) {
	app.prefs.Shell.RawText.Text = text
	app.prefs.Shell.RawText.Cursor = len([]rune(text))
	app.prefs.Shell.RawEdited(app.prefs)
}

func TestAValidRawEditAppliesAtOnceAndIsWrittenAsTyped(t *testing.T) {
	a, path := openSettingsPage(t, "appearance")
	a.prefs.OpenRaw()
	text := "{\n  \"layout\": \"side\"\n}\n"
	setRaw(a, text)
	if a.settings.Layout != "side" || len(a.prefs.Shell.Problems) != 0 {
		t.Fatalf("Layout %q problems %+v", a.settings.Layout, a.prefs.Shell.Problems)
	}
	if data, _ := os.ReadFile(path); string(data) != text {
		t.Fatalf("file = %q, want the text as typed", data)
	}
}

func TestARawEditWithProblemsLeavesTheLastGoodSettingsAndTheFile(t *testing.T) {
	a, path := openSettingsPage(t, "appearance")
	a.prefs.OpenRaw()
	setRaw(a, `{"layout": "side"}`)
	good, _ := os.ReadFile(path)
	for _, bad := range []string{`{"layout": `, `{"layout": "diagonal"}`, `{"colour": 1}`, `[1]`} {
		setRaw(a, bad)
		if a.settings.Layout != "side" || len(a.prefs.Shell.Problems) == 0 {
			t.Errorf("%q: Layout %q problems %v", bad, a.settings.Layout, a.prefs.Shell.Problems)
		}
		if data, _ := os.ReadFile(path); string(data) != string(good) {
			t.Errorf("%q: the file changed to %q", bad, data)
		}
	}
	if dump := view.Dump(a.prefs.View(a.ctx)); !strings.Contains(dump, "JSON") {
		t.Fatalf("raw view:\n%s", dump)
	}
}

func TestTheRawViewMarksKeysThatHaveProblems(t *testing.T) {
	a, _ := openSettingsPage(t, "appearance")
	a.prefs.OpenRaw()
	setRaw(a, "{\n  \"colour\": 1,\n  \"layout\": \"stacked\"\n}")
	marked := func(line string) []string {
		var out []string
		for _, span := range a.prefs.Shell.RawText.Styler(line) {
			if span.Style == w.Error {
				out = append(out, span.Text)
			}
		}
		return out
	}
	if got := marked(`  "colour": 1,`); len(got) != 1 || got[0] != `"colour"` {
		t.Fatalf("marked on the bad line: %q", got)
	}
	if got := marked(`  "layout": "stacked"`); len(got) != 0 {
		t.Fatalf("marked on a good line: %q", got)
	}
}

func TestOpeningTheRawViewShowsWhatTheFormChanged(t *testing.T) {
	a, _ := openSettingsPage(t, "appearance")
	a.prefs.Shell.Search = widget.NewTextField("request and response")
	clickText(t, a, "Side by side", 0)
	a.prefs.OpenRaw()
	if !strings.Contains(a.prefs.Shell.RawText.Text, `"layout": "side"`) {
		t.Fatalf("raw text = %q", a.prefs.Shell.RawText.Text)
	}
	a.prefs.Shell.CloseRaw()
	if a.prefs.Shell.Raw {
		t.Fatal("still in raw mode")
	}
}

func TestAChangeOnDiskShowsUpInTheSettingsAndInTheRawView(t *testing.T) {
	a, path := openSettingsPage(t, "appearance")
	a.prefs.OpenRaw()
	if err := os.WriteFile(path, []byte(`{"corners": "square"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	a.prefs.Poll()
	if a.settings.Corners != "square" {
		t.Fatalf("Corners = %q after the file changed", a.settings.Corners)
	}
	if a.prefs.Shell.RawText.Text != `{"corners": "square"}` {
		t.Fatalf("raw text = %q", a.prefs.Shell.RawText.Text)
	}
}

func TestABadFileOnDiskKeepsTheSettingsAndSaysSo(t *testing.T) {
	a, path := openSettingsPage(t, "appearance")
	a.saveSettings(func(prefs *settings.Settings) { prefs.Corners = "round" })
	if err := os.WriteFile(path, []byte(`{"corners": `), 0o600); err != nil {
		t.Fatal(err)
	}
	a.prefs.Poll()
	if a.settings.Corners != "round" {
		t.Fatalf("Corners = %q, want the last good value", a.settings.Corners)
	}
	found := false
	for _, toast := range a.toasts {
		found = found || (toast.Kind == widget.ToastError && strings.Contains(toast.Text, "settings.json"))
	}
	if !found {
		t.Fatalf("no toast about the file; toasts %+v", a.toasts)
	}
}

func TestOurOwnWritesAreNotReloadedAsOutsideChanges(t *testing.T) {
	a, _ := openSettingsPage(t, "appearance")
	a.saveSettings(func(prefs *settings.Settings) { prefs.Corners = "round" })
	a.settings.Corners = "square" // a change not yet saved must survive a poll
	a.prefs.Poll()
	if a.settings.Corners != "square" {
		t.Fatal("the poll reloaded the file we had just written")
	}
}

func TestCtrlSpaceInTheRawViewOffersKeysAndValues(t *testing.T) {
	a, _ := openSettingsPage(t, "appearance")
	a.prefs.OpenRaw()
	a.prefs.Shell.RawText = widget.NewTextArea(`{"appearance": {"la`)
	a.prefs.Shell.RawText.Cursor = len([]rune(a.prefs.Shell.RawText.Text))
	a.prefs.Shell.RawText.Focused = true
	a.completeVariable()
	if widget.PopupMenu(a.ctx) == nil || len(widget.PopupMenu(a.ctx).Items) != 1 || widget.PopupMenu(a.ctx).Items[0].Label != "layout" {
		t.Fatalf("menu = %+v", widget.PopupMenu(a.ctx))
	}
	widget.PopupMenu(a.ctx).Items[0].Run()
	if a.prefs.Shell.RawText.Text != `{"appearance": {"layout"` {
		t.Fatalf("text after completing = %q", a.prefs.Shell.RawText.Text)
	}

	a.closeMenu()
	a.prefs.Shell.RawText = widget.NewTextArea(`{"appearance": {"layout": "st`)
	a.prefs.Shell.RawText.Cursor = len([]rune(a.prefs.Shell.RawText.Text))
	a.prefs.Shell.RawText.Focused = true
	a.completeVariable()
	if widget.PopupMenu(a.ctx) == nil || widget.PopupMenu(a.ctx).Items[0].Label != "stacked" {
		t.Fatalf("menu = %+v", widget.PopupMenu(a.ctx))
	}
}

// Enter that accepts a completion must not also type a line break into the
// editor that still has the focus under the menu, and Up and Down that move
// in the menu must not move the caret.
func TestEnterAcceptingACompletionDoesNotStartANewLine(t *testing.T) {
	a, _ := openSettingsPage(t, "appearance")
	a.prefs.OpenRaw()
	a.prefs.Shell.RawText = widget.NewTextArea("{\n  \"")
	a.prefs.Shell.RawText.Cursor = len([]rune(a.prefs.Shell.RawText.Text))
	a.prefs.Shell.RawText.Focused = true
	a.completeVariable() // the menu of sections
	if widget.PopupMenu(a.ctx) == nil {
		t.Fatal("no menu")
	}
	a.Step(core.Input{X: 5000, Y: 5000, Enter: true, Keys: []ebiten.Key{ebiten.KeyEnter}})
	if strings.Count(a.prefs.Shell.RawText.Text, "\n") != 1 {
		t.Fatalf("Enter added a line break: %q", a.prefs.Shell.RawText.Text)
	}
	if !strings.Contains(a.prefs.Shell.RawText.Text, `"appearance"`) {
		t.Fatalf("the completion was not applied: %q", a.prefs.Shell.RawText.Text)
	}
}

func TestTheMenuKeysDoNotMoveTheCaretOfTheEditorUnderIt(t *testing.T) {
	a, _ := openSettingsPage(t, "appearance")
	a.prefs.OpenRaw()
	a.prefs.Shell.RawText = widget.NewTextArea("{\n  \"")
	a.prefs.Shell.RawText.Cursor = len([]rune(a.prefs.Shell.RawText.Text))
	a.prefs.Shell.RawText.Focused = true
	a.completeVariable()
	caret := a.prefs.Shell.RawText.Cursor
	a.Step(core.Input{X: 5000, Y: 5000, Edits: []core.Edit{core.EditUp}, Keys: []ebiten.Key{ebiten.KeyArrowUp}})
	a.Step(core.Input{X: 5000, Y: 5000, Edits: []core.Edit{core.EditDown}, Keys: []ebiten.Key{ebiten.KeyArrowDown}})
	if a.prefs.Shell.RawText.Cursor != caret {
		t.Fatalf("the caret moved from %d to %d while the menu used the arrows", caret, a.prefs.Shell.RawText.Cursor)
	}
}

func TestEscapeClosingTheMenuKeepsTheEditorFocused(t *testing.T) {
	a, _ := openSettingsPage(t, "appearance")
	a.prefs.OpenRaw()
	a.prefs.Shell.RawText = widget.NewTextArea("{\n  \"")
	a.prefs.Shell.RawText.Cursor = len([]rune(a.prefs.Shell.RawText.Text))
	a.prefs.Shell.RawText.Focused = true
	a.completeVariable()
	a.Step(core.Input{X: 5000, Y: 5000, Escape: true})
	if widget.PopupMenu(a.ctx) != nil || !a.prefs.Shell.RawText.Focused {
		t.Fatalf("menu open %v, editor focused %v: Escape should close only the menu", widget.PopupMenu(a.ctx) != nil, a.prefs.Shell.RawText.Focused)
	}
}
