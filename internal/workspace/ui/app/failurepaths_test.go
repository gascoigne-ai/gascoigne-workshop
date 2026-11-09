package ui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
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
		trigger   func(t *testing.T, a *App)
		wantToast string // text of an error toast; "" when only the status is checked
		wantState string // text the status line must contain
	}{
		{"network failure", func(t *testing.T, a *App) {
			tab := a.cur()
			pointTabAt(tab, "http://127.0.0.1:1/")
			a.send(tab)
			waitForResponse(t, a, tab)
		}, "", "Request failed"},
		{"clipboard cannot be read for a cURL import", func(t *testing.T, a *App) {
			a.ctx.Clipboard = &fakeClip{err: errors.New("no clipboard tool")}
			a.importCurl(&a.col.Folder)
		}, "Cannot read the clipboard", ""},
		{"clipboard is not a cURL command", func(t *testing.T, a *App) {
			a.ctx.Clipboard = &fakeClip{text: "hello"}
			a.importCurl(&a.col.Folder)
		}, "Not a cURL command", ""},
		{"clipboard is not an OpenAPI document", func(t *testing.T, a *App) {
			a.ctx.Clipboard = &fakeClip{text: "{}"}
			a.importOpenAPI(&a.col.Folder)
		}, "Not an OpenAPI document", ""},
		{"clipboard is not a Postman collection", func(t *testing.T, a *App) {
			a.ctx.Clipboard = &fakeClip{text: "[]"}
			a.importPostman(&a.col.Folder)
		}, "Not a Postman collection", ""},
		{"the collection cannot be saved", func(t *testing.T, a *App) {
			a.workspace = blockedDir(t)
			a.saveWorkspace()
		}, "Saving failed", ""},
		{"there is no folder to save to", func(t *testing.T, a *App) {
			a.workspace, a.defaultWorkspace = "", ""
			a.saveWorkspace()
		}, "No folder to save to", ""},
		{"the settings cannot be saved", func(t *testing.T, a *App) {
			a.settingsPath = blockedDir(t)
			a.saveSettings(func(*settings.Settings) {})
		}, "Could not save settings", ""},
		{"a response body cannot be written", func(t *testing.T, a *App) {
			tab := a.cur()
			tab.Resp = &model.Response{Status: 200, Body: []byte("x")}
			a.writeBody(tab, blockedDir(t))
		}, "Saving failed", ""},
		{"there is no response to save", func(t *testing.T, a *App) {
			a.cur().Resp = nil
			a.saveBody(a.cur())
		}, "There is no response to save", ""},
		{"there is no response to open in the browser", func(t *testing.T, a *App) {
			a.cur().Resp = nil
			a.openInBrowser(a.cur())
		}, "There is no response to open", ""},
		{"the state cannot be saved", func(t *testing.T, a *App) {
			a.statePath = blockedDir(t)
			a.stateTick() // notices the change
			a.sidebarSplit.Size++
			a.stateTick()
			clock := a.now()
			a.now = func() time.Time { return clock.Add(10 * time.Second) }
			a.stateTick()
		}, "not a directory", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a, _ := newTestApp(t)
			before := a.status
			test.trigger(t, a)
			if test.wantToast != "" {
				found := false
				for _, toast := range a.toasts {
					found = found || (toast.isErr && strings.Contains(toast.text, test.wantToast))
				}
				if !found {
					t.Fatalf("no error toast containing %q; toasts %+v, status %q", test.wantToast, a.toasts, a.status)
				}
			}
			if test.wantState != "" && !strings.Contains(a.status, test.wantState) {
				t.Fatalf("status %q (was %q), want it to contain %q", a.status, before, test.wantState)
			}
		})
	}
}

// Regression: a failed write of the response history or the sent log put
// its message in the status line, which the very next line of apply
// replaced with the result of the send, so the failure was never seen.
func TestAHistoryThatCannotBeWrittenIsReportedByAToast(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	a, _ := newTestApp(t)
	a.statePath = filepath.Join(file, "state.json") // history lives beside it, below a file
	tab := a.cur()
	pointTabAt(tab, jsonServer(t).URL)
	a.send(tab)
	waitForResponse(t, a, tab)

	found := false
	for _, toast := range a.toasts {
		found = found || (toast.isErr && strings.Contains(toast.text, "history"))
	}
	if !found {
		t.Fatalf("no toast about the history; toasts %+v, status %q", a.toasts, a.status)
	}
	if len(a.toasts) > 2 {
		t.Fatalf("one failure made %d toasts: %+v", len(a.toasts), a.toasts)
	}
}
