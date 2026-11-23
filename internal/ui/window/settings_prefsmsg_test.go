package window

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/prefs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
)

func TestSendAppliesSettingsMessages(t *testing.T) {
	a := testApp(t)
	a.over.Settings.Send(prefs.Replace{Settings: settings.Settings{ResponseView: "raw", UIScalePct: 100}})
	if a.settings.ResponseView != "raw" || !a.resp.Look(a.cur().Tab).Raw() {
		t.Fatalf("Replace: setting %q raw %v", a.settings.ResponseView, a.resp.Look(a.cur().Tab).Raw())
	}
	a.over.Settings.Send(prefs.Change{Edit: func(current *settings.Settings) { current.ResponseView = "compact" }})
	if !a.resp.Look(a.cur().Tab).Compact() {
		t.Fatal("Change edits the settings in effect")
	}
	a.over.Settings.Send(prefs.Edit{Edit: func(current *settings.Settings) { current.ResponseView = "pretty" }})
	if a.settings.ResponseView != "pretty" {
		t.Fatalf("Edit: %q", a.settings.ResponseView)
	}
	a.over.Settings.Send(prefs.ScaleLive{Percent: 130})
	if a.settings.UIScalePct != 130 {
		t.Fatalf("ScaleLive: %d", a.settings.UIScalePct)
	}
}

// FileMarked is one message for both file states: Saved false marks the
// file as behind the edits, Saved true as written.
func TestSendFileMarkedRecordsFileState(t *testing.T) {
	a := testApp(t)
	a.over.Settings.Send(prefs.FileMarked{Saved: false})
	if !a.over.Settings.Unsaved() {
		t.Fatal("FileMarked{Saved: false} must mark the file unsaved")
	}
	a.over.Settings.Send(prefs.FileMarked{Saved: true})
	if a.over.Settings.Unsaved() {
		t.Fatal("FileMarked{Saved: true} must mark the file saved")
	}
}

// Change rebuilds the Colors page's fields from the colors in effect; Edit
// leaves them, because the page edits through it while a field is typed.
func TestSendChangeRebuildsColorsPageAndEditDoesNot(t *testing.T) {
	tests := map[string]struct {
		msg     func(prefs.EditFunc) prefs.Msg
		rebuilt bool
	}{
		"Change": {func(edit prefs.EditFunc) prefs.Msg { return prefs.Change{Edit: edit} }, true},
		"Edit":   {func(edit prefs.EditFunc) prefs.Msg { return prefs.Edit{Edit: edit} }, false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			a := testApp(t)
			token := theme.CustomColors[0].Token
			before := a.over.Settings.Colors.Input(a.over.Settings, token)
			a.over.Settings.Send(test.msg(func(current *settings.Settings) { current.ResponseView = "compact" }))
			if rebuilt := a.over.Settings.Colors.Input(a.over.Settings, token) != before; rebuilt != test.rebuilt {
				t.Fatalf("colors page rebuilt = %v, want %v", rebuilt, test.rebuilt)
			}
		})
	}
}

func TestSendRecordsFileStateAndStatus(t *testing.T) {
	a := testApp(t)
	a.over.Settings.Send(prefs.SetStatus{Text: "hello"})
	if a.notices.Status() != "hello" {
		t.Fatalf("status %q", a.notices.Status())
	}
	a.over.Settings.Send(prefs.Toast{Text: "oops", Level: severity.Error})
	if len(a.over.Toasts.Stack()) != 1 || a.over.Toasts.Stack()[0].Text != "oops" {
		t.Fatal("Toast")
	}
}
