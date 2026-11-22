package app

import (
	"context"
	"errors"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
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
	pointTabAt(tab, srv.URL)

	a.dispatch(tab, tab.Req)
	a.clearCookies()
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
		focus func(*reqTab)
	}{
		{"url", func(tab *reqTab) { tab.url.Focused = true }},
		{"find", func(tab *reqTab) { tab.find.Focused = true }},
		{"timeout", func(tab *reqTab) { tab.timeout.Focused = true }},
		{"body", func(tab *reqTab) { tab.body.Focused = true }},
		{"tests", func(tab *reqTab) { tab.tests.Focused = true }},
		{"graphql variables", func(tab *reqTab) { tab.gqlVars.Focused = true }},
		{"file path", func(tab *reqTab) { tab.file.Focused = true }},
		{"message composer", func(tab *reqTab) { tab.msg.Focused = true }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := testApp(t)
			if a.textFocused() {
				t.Fatal("nothing is focused at the start")
			}
			c.focus(a.cur())
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
	if tab.look.Raw() {
		t.Fatal("start: the tab should not be raw")
	}

	a.saveSettings(func(prefs *settings.Settings) { prefs.ResponseView = "raw" })

	if !tab.look.Raw() {
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
	pointTabAt(tab, srv.URL+"/slow")
	click(a, sendX, urlBarY)
	if !tab.Sending() {
		t.Fatal("test setup: the request must be running")
	}
	a.closeTab(a.list.Active)
	if tab.Sending() {
		t.Error("the request of the closed tab is still running")
	}
}

func TestClosingATabClosesItsSession(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	session := &fakeSession{}
	openSession(tab, session)
	a.closeTab(a.list.Active)
	if !session.closed {
		t.Error("the session of the closed tab is still open")
	}
}

func TestAFailedCommandIsShownInTheStatusLine(t *testing.T) {
	a := testApp(t)
	a.handle(bus.Failed{Err: errors.New("a command panicked: boom")})
	if !strings.Contains(a.status, "boom") {
		t.Fatalf("status %q does not show the failure", a.status)
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
	a.askDesktop("Open", filedialog.OpenFile, "", func(string) {})
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

// An edit of a row past the end (the table shrank while the cell was
// open) appended one row and then indexed the row it had meant, which
// was out of range.
func TestCommittingACellEditOfAVanishedRowAppendsOneRow(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	before := len(tab.Req.Params)
	a.beginEdit(paramsPane, tab, before, 0)
	a.edit.row = before + 5 // the rows above were deleted meanwhile
	a.edit.field = widget.NewTextField("late")
	a.commitEdit()
	if len(tab.Req.Params) != before+1 || tab.Req.Params[before].Name != "late" {
		t.Fatalf("params %+v, want one new row named late", tab.Req.Params)
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
	if a.dialog == nil || !strings.Contains(a.dialog.Body, "/bin/true --flag") {
		t.Fatalf("no dialog naming the command: %+v", a.dialog)
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
	if a.dialog != nil {
		t.Fatal("an allowed command asked again")
	}
	if !slices.Contains(a.snapshot().Trusted, "command:/bin/true --flag") {
		t.Fatalf("trusted %v: the answer is not remembered in the state", a.snapshot().Trusted)
	}
	tab.Req.URL = "stdio:/bin/true --other"
	a.connect(tab)
	if a.dialog == nil {
		t.Fatal("a changed command must ask again")
	}
}

func TestAFileOutsideTheCollectionAsksBeforeItIsSent(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	tab.Req.Method, tab.Req.URL = "POST", "http://127.0.0.1:1/x"
	tab.Req.BodyType, tab.Req.Body = request.BodyFile, "/etc/hostname"
	a.send(tab)
	if a.dialog == nil || !strings.Contains(a.dialog.Body, "/etc/hostname") || tab.Sending() {
		t.Fatalf("dialog %+v sending %v: want a question and no request", a.dialog, tab.Sending())
	}
}
