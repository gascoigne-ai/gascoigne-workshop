package ui

import (
	"context"
	"testing"
	"time"
)

// A file chooser opened from the settings belongs to the dialog: closing the
// dialog closes the chooser and nothing is applied afterwards.
func TestClosingTheSettingsClosesAFilePromptOpenedFromThem(t *testing.T) {
	a, _ := openSettingsMenu(t)
	a.files = fakeDialog{} // unavailable: the path prompt is used
	picked := ""
	a.pickSettingsFile("Choose CA certificates", func(path string) { picked = path })
	if a.dialog == nil {
		t.Fatal("no prompt")
	}
	a.closeSettingsNow()
	if a.dialog != nil {
		t.Fatal("the prompt stayed open after the settings closed")
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
	if a.dialog == nil {
		t.Fatal("closing the settings closed a prompt that is not theirs")
	}
}

// fakeDialog is a file dialog that exists when choose is set.
type fakeDialog struct {
	choose func(ctx context.Context, title string, save bool, name string) (string, error)
}

func (dialog fakeDialog) Available() bool { return dialog.choose != nil }
func (dialog fakeDialog) Choose(ctx context.Context, title string, save bool, name string) (string, error) {
	return dialog.choose(ctx, title, save, name)
}
