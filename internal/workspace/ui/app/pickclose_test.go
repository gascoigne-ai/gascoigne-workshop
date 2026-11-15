package app

import (
	"context"
	"testing"
	"time"
)

// A file chooser opened from the settings belongs to the dialog: closing the
// dialog closes the chooser and nothing is applied afterwards.
func TestClosingTheSettingsClosesAFileBrowserOpenedFromThem(t *testing.T) {
	a, _ := openSettingsMenu(t)
	a.files = fakeDialog{} // unavailable: the in-app file browser is used
	picked := ""
	a.pickSettingsFile("Choose CA certificates", func(path string) { picked = path })
	if a.browser == nil {
		t.Fatal("no file browser")
	}
	a.closeSettingsNow()
	if a.browser != nil {
		t.Fatal("the file browser stayed open after the settings closed")
	}
	if picked != "" {
		t.Fatalf("a path %q was applied", picked)
	}
}

func TestClosingTheSettingsCancelsTheNativeFileDialogAndIgnoresItsAnswer(t *testing.T) {
	a, _ := openSettingsMenu(t)
	finished := make(chan struct{})
	a.files = fakeDialog{choose: func(ctx context.Context, _ string, _ bool, _ string) (string, error) {
		<-ctx.Done() // the dialog stays open until it is cancelled
		close(finished)
		return "", ctx.Err()
	}}
	called := false
	a.pickSettingsFile("Choose", func(string) { called = true })
	a.closeSettingsNow()
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("the file dialog was not cancelled when the settings closed")
	}
	time.Sleep(20 * time.Millisecond)
	a.drain()
	if called || len(a.toasts) != 0 {
		t.Fatalf("a cancelled dialog applied a path (%v) or complained: %+v", called, a.toasts)
	}
}

func TestAFileDialogOpenedElsewhereSurvivesTheSettingsClosing(t *testing.T) {
	a, _ := openSettingsMenu(t)
	a.files = fakeDialog{}
	a.pickFile("Choose the file to send", false, "", func(string) {}) // the request body's, not the settings'
	a.closeSettingsNow()
	if a.browser == nil {
		t.Fatal("closing the settings closed a file browser that is not theirs")
	}
}

// A settings file question taken back after the user opened another
// browser must not close that one.
func TestTakingBackAQuestionLeavesALaterBrowserOpen(t *testing.T) {
	a, _ := openSettingsMenu(t)
	a.files = fakeDialog{}
	cancel := a.pickFile("First", false, "", func(string) {})
	a.closeBrowser()
	a.pickFile("Second", false, "", func(string) {})
	cancel()
	if a.browser == nil || a.browser.title != "Second" {
		t.Fatal("taking back the first question closed the second browser")
	}
}

// fakeDialog is a file dialog that exists when choose is set; suggest is
// the program it names for native dialogs.
type fakeDialog struct {
	choose  func(ctx context.Context, title string, save bool, name string) (string, error)
	suggest string
}

func (dialog fakeDialog) Available() bool   { return dialog.choose != nil }
func (dialog fakeDialog) Suggested() string { return dialog.suggest }
func (dialog fakeDialog) Choose(ctx context.Context, title string, save bool, name string) (string, error) {
	return dialog.choose(ctx, title, save, name)
}
