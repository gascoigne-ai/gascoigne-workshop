package app

import (
	"context"
	"errors"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
	"github.com/hajimehoshi/ebiten/v2"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Regression: the send goroutine read a.reg and a.settings while clearing
// the cookies replaced a.reg on the UI goroutine, a data race. The
// goroutine must work from what dispatch captured before it started.
func TestClearingCookiesWhileARequestRunsIsRaceFree(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		<-release
		io.WriteString(writer, `{"ok":true}`)
	}))
	t.Cleanup(srv.Close)
	a := testApp(t)
	tab := a.cur()
	a.pointTabAt(tab, srv.URL)

	a.dispatch(tab, tab.Req)
	authHost{a}.Clear()
	close(release)

	waitForResponse(t, a, tab)
}

// Regression: textFocused listed the focused fields by hand and missed
// the assertions editor, the GraphQL variables, the file path and the
// message composer, so plain-key shortcuts and table undo fired while
// they had the keyboard.
func TestFocusInAnyEditorOfTheTabCountsAsTyping(t *testing.T) {
	cases := []struct {
		name  string
		focus func(*App, *reqTab)
	}{
		{"url", func(a *App, tab *reqTab) { a.urlbar.Field(tab.Tab).Focused = true }},
		{"find", func(a *App, tab *reqTab) { a.resp.Field(tab.Tab).Focused = true }},
		{"body", func(a *App, tab *reqTab) { a.body.Area(tab.Tab).Focused = true }},
		{"file path", func(a *App, tab *reqTab) { a.body.File(tab.Tab).Focused = true }},
		{"message composer", func(a *App, tab *reqTab) { a.ws.Composer(tab.Tab).Focused = true }},
		{"auth token", func(a *App, tab *reqTab) { a.auth.Field(tab.Tab, "token").Focused = true }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := testApp(t)
			if a.textFocused() {
				t.Fatal("nothing is focused at the start")
			}
			c.focus(a, a.cur())
			if !a.textFocused() {
				t.Fatalf("%s has the keyboard but textFocused is false", c.name)
			}
		})
	}
}

// Regression: only changeSettings refreshed the open tabs when the
// response view setting changed; an edit through saveSettings (the colour
// and shortcut pages, the environment chip) left them on the old view.
func TestAnySettingsEditThatChangesTheResponseViewRefreshesOpenTabs(t *testing.T) {
	a := testApp(t) // starts on "pretty"
	tab := a.cur()
	if a.resp.Look(tab.Tab).Raw() {
		t.Fatal("start: the tab should not be raw")
	}

	a.saveSettings(func(prefs *settings.Settings) { prefs.ResponseView = "raw" })

	if !a.resp.Look(tab.Tab).Raw() {
		t.Fatal("the open tab kept its old response view")
	}
}

// Closing a tab dropped its widgets but left its request running and its
// socket open; the result, or the events, then arrived for a tab nobody
// could see.
func TestClosingATabStopsItsRequest(t *testing.T) {
	srv, _ := pathServer(t)
	a := testApp(t)
	tab := a.cur()
	a.pointTabAt(tab, srv.URL+"/slow")
	click(a, sendX, urlBarY)
	if !a.sends.Sending(tab.Req) {
		t.Fatal("test setup: the request must be running")
	}
	a.closeTab(a.tabs.Active())
	if a.sends.Sending(tab.Req) {
		t.Error("the request of the closed tab is still running")
	}
}

func TestClosingATabClosesItsSession(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	session := &fakeSession{}
	openSession(tab, session)
	a.closeTab(a.tabs.Active())
	if !session.closed {
		t.Error("the session of the closed tab is still open")
	}
}

func TestAFailedCommandIsShownInTheStatusLine(t *testing.T) {
	a := testApp(t)
	a.handle(bus.Failed{Err: errors.New("a command panicked: boom")})
	if !strings.Contains(a.notices.Status(), "boom") {
		t.Fatalf("status %q does not show the failure", a.notices.Status())
	}
}

type panickingStreamer struct{}

func (panickingStreamer) Send(context.Context, request.Request) (request.Response, error) {
	return request.Response{}, nil
}

func (panickingStreamer) Connect(context.Context, request.Request) (request.Session, error) {
	panic("connect boom")
}

// The connect and file dialog goroutines were started with a bare go
// statement, so a panic in a protocol or a dialog program ended the app.
func TestAPanicWhileConnectingIsShownNotFatal(t *testing.T) {
	a := testApp(t)
	a.reg = request.Registry{"websocket": panickingStreamer{}}
	tab := a.cur()
	tab.Req.Protocol = "websocket"
	tab.Req.URL = "ws://x.test"
	a.connect(tab)
	statusSoon(t, a, "connect boom")
}

func TestAPanicInTheDesktopDialogIsShownNotFatal(t *testing.T) {
	a := testApp(t)
	a.files = fakeDialog{choose: func(context.Context, string, filedialog.Mode, string) (string, error) { panic("dialog boom") }}
	a.pickFile("Open", filedialog.OpenFile, "", func(string) {}) // the desktop dialog is asked: it exists
	statusSoon(t, a, "dialog boom")
}

// A huge timeout (from a request or the settings file) overflowed
// time.Duration to a negative value, so the request failed at once.
func TestTimeoutIsCappedInsteadOfOverflowing(t *testing.T) {
	a := testApp(t)
	for _, sec := range []int{1 << 40, math.MaxInt} {
		got := a.timeoutFor(request.Request{TimeoutSec: sec})
		if got <= 0 || got > settings.MaxTimeout {
			t.Errorf("timeoutFor(%d s) = %v, want 1..%v", sec, got, settings.MaxTimeout)
		}
	}
}

type countingStreamer struct {
	panickingStreamer
	connects *atomic.Int32
}

func (streamer countingStreamer) Connect(context.Context, request.Request) (request.Session, error) {
	streamer.connects.Add(1)
	return nil, errors.New("test server: no session")
}

// An imported collection could name any command as an MCP stdio server,
// and connecting started it at once. A command now needs the user's yes,
// once, and is remembered.
func TestAStdioCommandAsksOnceBeforeItRuns(t *testing.T) {
	a := testApp(t)
	var connects atomic.Int32
	a.reg = request.Registry{"mcp": countingStreamer{connects: &connects}}
	tab := a.cur()
	tab.Req.Protocol, tab.Req.URL = "mcp", "stdio:/bin/true --flag"

	a.connect(tab)
	if a.over.Dialog() == nil || !strings.Contains(a.over.Dialog().Body, "/bin/true --flag") {
		t.Fatalf("no dialog naming the command: %+v", a.over.Dialog())
	}
	time.Sleep(50 * time.Millisecond)
	if connects.Load() != 0 {
		t.Fatal("the command started before the user agreed")
	}
	a.dialogDefault() // Allow
	for i := 0; i < 200 && connects.Load() == 0; i++ {
		a.Step(core.Input{})
		time.Sleep(5 * time.Millisecond)
	}
	if connects.Load() != 1 {
		t.Fatalf("connects after Allow = %d, want 1", connects.Load())
	}
	a.connect(tab)
	if a.over.Dialog() != nil {
		t.Fatal("an allowed command asked again")
	}
	if !slices.Contains(a.stateFile.Snapshot().Trusted, "command:/bin/true --flag") {
		t.Fatalf("trusted %v: the answer is not remembered in the state", a.stateFile.Snapshot().Trusted)
	}
	tab.Req.URL = "stdio:/bin/true --other"
	a.connect(tab)
	if a.over.Dialog() == nil {
		t.Fatal("a changed command must ask again")
	}
}

func TestAFileOutsideTheCollectionAsksBeforeItIsSent(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	tab.Req.Method, tab.Req.URL = "POST", "http://127.0.0.1:1/x"
	tab.Req.BodyType, tab.Req.Body = request.BodyFile, "/etc/hostname"
	a.send(tab)
	if a.over.Dialog() == nil || !strings.Contains(a.over.Dialog().Body, "/etc/hostname") || a.sends.Sending(tab.Req) {
		t.Fatalf("dialog %+v sending %v: want a question and no request", a.over.Dialog(), a.sends.Sending(tab.Req))
	}
}

// The find field of the response is a text field: a plain-key shortcut
// typed into it must type, not run, and Ctrl+Z must stay with its text
// instead of undoing a table edit. Pinned before the field's keys moved into
// ui/response, so it must pass on both sides of the move.
func TestTheFindFieldKeepsPlainKeysAndUndoToItself(t *testing.T) {
	t.Run("a plain-key shortcut types", func(t *testing.T) {
		a := jsonResponse(t, `{"id":1}`) // the find bar shows over a response
		a.settings.Shortcuts = map[string]string{"new-request": "Q"}
		count := len(a.tabs.Tabs())
		a.Step(combo(ebiten.KeyF, true, false, false)) // opens the find bar with its field focused
		a.Step(core.Input{Keys: []ebiten.Key{ebiten.KeyQ}, Typed: []rune("q")})
		if len(a.tabs.Tabs()) != count {
			t.Fatal("typing Q in the find field must not run its shortcut")
		}
		if got := a.resp.Field(a.cur().Tab).Text; got != "q" {
			t.Fatalf("find text %q, want the typed q", got)
		}
	})
	t.Run("undo stays with the field", func(t *testing.T) {
		a := testApp(t)
		req := a.cur().Req
		click(a, 739, paramsRow0Y) // a table change that could be undone
		if len(req.Params) != 2 {
			t.Fatal("test setup")
		}
		a.resp.Field(a.cur().Tab).Focused = true
		a.Step(core.Input{Ctrl: true, Undo: true})
		if len(req.Params) != 2 {
			t.Fatalf("params %+v: with the find field focused Ctrl+Z belongs to its text", req.Params)
		}
	})
}

// Cancelling a send replaces the response with the cancellation and empties
// the lines the body view shows, so a stale body is not left under it.
func TestCancellingASendClearsTheResponseLines(t *testing.T) {
	a := jsonResponse(t, `{"id":1}`)
	tab := a.cur()
	if len(a.resp.Code(tab.Tab).Lines) == 0 {
		t.Fatal("test setup: the response shows no lines")
	}
	setSending(a, tab, true)
	a.cancelSend(tab)
	if len(a.resp.Code(tab.Tab).Lines) != 0 {
		t.Fatalf("%d lines left after the cancel", len(a.resp.Code(tab.Tab).Lines))
	}
	if tab.Resp == nil || tab.Resp.Err != request.ErrCancelled {
		t.Fatalf("response %+v, want the cancellation", tab.Resp)
	}
}
