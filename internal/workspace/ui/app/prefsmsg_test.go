package ui

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/controller/prefs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
)

func TestSendAppliesSettingsMessages(t *testing.T) {
	a, _ := newTestApp(t)
	a.Send(prefs.Replace{Settings: settings.Settings{ResponseView: "raw", UIScalePct: 100}})
	if a.settings.ResponseView != "raw" || !a.cur().look.Raw() {
		t.Fatalf("Replace: setting %q raw %v", a.settings.ResponseView, a.cur().look.Raw())
	}
	a.Send(prefs.Change{Edit: func(s *settings.Settings) { s.ResponseView = "compact" }})
	if !a.cur().look.Compact() {
		t.Fatal("Change edits the settings in effect")
	}
	a.Send(prefs.Edit{Edit: func(s *settings.Settings) { s.ResponseView = "pretty" }})
	if a.settings.ResponseView != "pretty" {
		t.Fatalf("Edit: %q", a.settings.ResponseView)
	}
	a.Send(prefs.ScaleLive{Percent: 130})
	if a.settings.UIScalePct != 130 {
		t.Fatalf("ScaleLive: %d", a.settings.UIScalePct)
	}
}

func TestSendRecordsFileStateAndStatus(t *testing.T) {
	a, _ := newTestApp(t)
	a.Send(prefs.MarkUnsaved{})
	if !a.Unsaved() {
		t.Fatal("MarkUnsaved")
	}
	a.Send(prefs.MarkSaved{})
	if a.Unsaved() {
		t.Fatal("MarkSaved")
	}
	a.Send(prefs.SetStatus{Text: "hello"})
	if a.status != "hello" {
		t.Fatalf("status %q", a.status)
	}
	a.Send(prefs.Toast{Text: "oops", IsErr: true})
	if len(a.toasts) != 1 || a.toasts[0].Text != "oops" {
		t.Fatal("Toast")
	}
}
