package app

import (
	"context"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/prefs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
	"testing"
	"time"
)

// A file chooser opened from the settings belongs to the dialog: closing the
// dialog closes the chooser and nothing is applied afterwards.
func TestClosingTheSettingsClosesAFileBrowserOpenedFromThem(t *testing.T) {
	a, _ := openSettingsMenu(t)
	a.files = fakeDialog{} // unavailable: the in-app file browser is used
	picked := ""
	a.prefs.Send(prefs.PickFile{Title: "Choose CA certificates", Done: func(path string) { picked = path }})
	if a.browser == nil {
		t.Fatal("no file browser")
	}
	a.prefs.Close()
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
	a.files = fakeDialog{choose: func(ctx context.Context, _ string, _ filedialog.Mode, _ string) (string, error) {
		<-ctx.Done() // the dialog stays open until it is cancelled
		close(finished)
		return "", ctx.Err()
	}}
	called := false
	a.prefs.Send(prefs.PickFile{Title: "Choose", Done: func(string) { called = true }})
	a.prefs.Close()
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
	a.pickFile("Choose the file to send", filedialog.OpenFile, "", func(string) {}) // the request body's, not the settings'
	a.prefs.Close()
	if a.browser == nil {
		t.Fatal("closing the settings closed a file browser that is not theirs")
	}
}

// A settings file question taken back after the user opened another
// browser must not close that one.
func TestTakingBackAQuestionLeavesALaterBrowserOpen(t *testing.T) {
	a, _ := openSettingsMenu(t)
	a.files = fakeDialog{}
	cancel := a.pickFile("First", filedialog.OpenFile, "", func(string) {})
	a.closeBrowser()
	a.pickFile("Second", filedialog.OpenFile, "", func(string) {})
	cancel()
	if a.browser == nil || a.browser.title != "Second" {
		t.Fatal("taking back the first question closed the second browser")
	}
}

// fakeDialog is a file dialog that exists when choose is set; suggest is
// the program it names for native dialogs.
type fakeDialog struct {
	choose  func(ctx context.Context, title string, mode filedialog.Mode, name string) (string, error)
	suggest string
}

func (dialog fakeDialog) Available() bool   { return dialog.choose != nil }
func (dialog fakeDialog) Suggested() string { return dialog.suggest }
func (dialog fakeDialog) Choose(ctx context.Context, title string, mode filedialog.Mode, name string) (string, error) {
	return dialog.choose(ctx, title, mode, name)
}

// The desktop dialog is asked in the mode the caller picked.
func TestPickFileAsksTheDesktopDialogInTheCallersMode(t *testing.T) {
	for _, mode := range []filedialog.Mode{filedialog.OpenFile, filedialog.SaveFile} {
		a := testApp(t)
		asked := make(chan filedialog.Mode, 1)
		a.files = fakeDialog{choose: func(_ context.Context, _ string, got filedialog.Mode, _ string) (string, error) {
			asked <- got
			return "", filedialog.ErrCancelled
		}}
		a.pickFile("Pick", mode, "x.json", func(string) {})
		select {
		case got := <-asked:
			if got != mode {
				t.Fatalf("asked in mode %v, want %v", got, mode)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("the desktop dialog was not asked")
		}
	}
}
