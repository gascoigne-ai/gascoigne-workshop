package window

import (
	"bytes"
	"errors"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/adapter/folders"
)

// The checklist of failures a user can meet. Each row triggers one and
// says what must be shown; none may pass silently. Wording rule: a toast
// (an error stays until clicked) states what failed and why, in the form
// "<what> failed: <cause>" or "Not a <thing>: <cause>"; the status line
// says "Request failed" for a failed send, whose reason is in the pane.
func TestEveryFailurePathIsShownToTheUser(t *testing.T) {
	blockedDir := func(t *testing.T) string {
		t.Helper()
		file := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		return filepath.Join(file, "inside") // a folder below a file cannot be made
	}
	tests := []struct {
		name      string
		trigger   func(t *testing.T, app *Component)
		wantToast string                    // text of an error toast; "" when only the status is checked
		wantState string                    // text the status line must contain
		settings  func(t *testing.T) string // where the settings are kept; nil is a file of their own
		state     func(t *testing.T) string // where the state is kept; nil keeps none
	}{
		{"network failure", func(t *testing.T, app *Component) {
			tab := app.cur()
			app.pointTabAt(tab, "http://127.0.0.1:1/")
			app.send(tab)
			waitForResponse(t, app, tab)
		}, "", "Request failed", nil, nil},
		{"clipboard cannot be read for a cURL import", func(t *testing.T, app *Component) {
			app.ctx.Clipboard = &fakeClip{err: errors.New("no clipboard tool")}
			app.importCurl(&app.col().Folder)
		}, "Cannot read the clipboard", "", nil, nil},
		{"clipboard is not a cURL command", func(t *testing.T, app *Component) {
			app.ctx.Clipboard = &fakeClip{text: "hello"}
			app.importCurl(&app.col().Folder)
		}, "Not a cURL command", "", nil, nil},
		{"clipboard is not an OpenAPI document", func(t *testing.T, app *Component) {
			app.ctx.Clipboard = &fakeClip{text: "{}"}
			app.importOpenAPI(&app.col().Folder)
		}, "Not an OpenAPI document", "", nil, nil},
		{"clipboard is not a Postman collection", func(t *testing.T, app *Component) {
			app.ctx.Clipboard = &fakeClip{text: "[]"}
			app.importPostman(&app.col().Folder)
		}, "Not a Postman collection", "", nil, nil},
		{"the collection cannot be saved", func(t *testing.T, app *Component) {
			app.setWorkspace(blockedDir(t))
			app.saveWorkspace()
		}, "Saving failed", "", nil, nil},
		{"there is no folder to save to", func(t *testing.T, app *Component) {
			app.setWorkspace("")
			app.defaultWorkspace = ""
			app.saveWorkspace()
		}, "No folder to save to", "", nil, nil},
		{"the settings cannot be saved", func(t *testing.T, app *Component) {
			app.saveSettings(func(*settings.Settings) {})
		}, "Could not save settings", "", blockedDir, nil},
		{"a response body cannot be written", func(t *testing.T, app *Component) {
			app.folders = folders.Disk{} // the real disk, so the blocked folder is what fails
			tab := app.cur()
			tab.Resp = &request.Response{Status: 200, Body: []byte("x")}
			saveBodyAs(t, app, tab, blockedDir(t))
		}, "Saving failed", "", nil, nil},
		{"a history was written by a newer version", func(t *testing.T, app *Component) {
			req := app.cur().Req
			if err := app.historyStore().Save(app.cols.Set().HistoryKey(req), []history.Entry{{Status: 200}}); err != nil {
				t.Fatal(err)
			}
			files, _ := filepath.Glob(filepath.Join(app.stateFile.HistoryDir(), "*.json"))
			if len(files) != 1 {
				t.Fatalf("history files %q, want one", files)
			}
			data, _ := os.ReadFile(files[0])
			newer := bytes.Replace(data, []byte(`"version":1`), []byte(`"version":2`), 1) // no newer build exists to write one
			if err := os.WriteFile(files[0], newer, 0o600); err != nil {
				t.Fatal(err)
			}
			app.loadHistory(req)
		}, "written by a newer version", "", nil, func(t *testing.T) string { return filepath.Join(t.TempDir(), "state.json") }}, // history lives beside it
		{"the state cannot be saved", func(t *testing.T, app *Component) {
			app.stateFile.Tick(app.stateFile.Key(), app.now()) // notices the change
			app.sidebarSplit.Size++
			app.stateFile.Tick(app.keys.State(), app.now())
			app.stateFile.Tick(app.keys.State(), app.now().Add(10*time.Second))
		}, "not a directory", "", nil, blockedDir},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var options []appOption
			if test.settings != nil {
				options = append(options, settingsAt(test.settings(t)))
			}
			if test.state != nil {
				options = append(options, withOptions(startOptions{StatePath: test.state(t)}))
			}
			a := testApp(t, options...)
			before := a.notices.Status()
			test.trigger(t, a)
			a.WaitWrites() // saves run in the background
			if test.wantToast != "" {
				found := false
				for _, toast := range a.over.Toasts.Stack() {
					found = found || (toast.Kind == widget.ToastError && strings.Contains(toast.Text, test.wantToast))
				}
				if !found {
					t.Fatalf("no error toast containing %q; toasts %+v, status %q", test.wantToast, a.over.Toasts.Stack(), a.notices.Status())
				}
			}
			if test.wantState != "" && !strings.Contains(a.notices.Status(), test.wantState) {
				t.Fatalf("status %q (was %q), want it to contain %q", a.notices.Status(), before, test.wantState)
			}
		})
	}
}

// Regression: a failed write of the response history or the sent log put
// its message in the status line, which the very next line of apply
// replaced with the result of the send, so the failure was never seen.
func TestAHistoryThatCannotBeWrittenIsReportedByAToast(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	a := testApp(t, withOptions(startOptions{StatePath: filepath.Join(file, "state.json")})) // history lives beside it, below a file
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {                           // made a file once the app has started
		t.Fatal(err)
	}
	tab := a.cur()
	a.pointTabAt(tab, jsonServer(t).URL)
	a.send(tab)
	waitForResponse(t, a, tab)
	a.WaitWrites() // the history is written in the background

	found := false
	for _, toast := range a.over.Toasts.Stack() {
		found = found || (toast.Kind == widget.ToastError && strings.Contains(toast.Text, "history"))
	}
	if !found {
		t.Fatalf("no toast about the history; toasts %+v, status %q", a.over.Toasts.Stack(), a.notices.Status())
	}
	if len(a.over.Toasts.Stack()) > 2 {
		t.Fatalf("one failure made %d toasts: %+v", len(a.over.Toasts.Stack()), a.over.Toasts.Stack())
	}
}
