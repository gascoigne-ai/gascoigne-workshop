package ui

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func shotApp(t *testing.T, opts Options) *App {
	t.Helper()
	opts.SettingsPath = filepath.Join(t.TempDir(), "settings.json")
	a, err := NewApp(opts)
	if err != nil {
		t.Fatal(err)
	}
	a.w, a.h = winW, winH
	return a
}

func TestScreenshotIsOffByDefault(t *testing.T) {
	a := shotApp(t, Options{})
	for range 50 {
		if err := a.shotUpdate(); err != nil {
			t.Fatalf("a normal run must never terminate itself: %v", err)
		}
	}
	if a.shot.ready(a) {
		t.Fatal("no screenshot path, nothing to capture")
	}
	if a.Err() != nil {
		t.Fatalf("Err = %v", a.Err())
	}
}

func TestScreenshotWaitsForTheSettleFramesThenTerminates(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png")})

	for i := 1; i < settleFrames; i++ {
		if err := a.shotUpdate(); err != nil {
			t.Fatal(err)
		}
		if a.shot.ready(a) {
			t.Fatalf("frame %d: not settled yet", i)
		}
	}
	a.shotUpdate()
	if !a.shot.ready(a) {
		t.Fatal("must be ready once the frames have settled")
	}

	a.shot.captured = true // what shotDraw records after writing the file
	if err := a.shotUpdate(); !errors.Is(err, ebiten.Termination) {
		t.Fatalf("after the capture the next update must end the game, got %v", err)
	}
	if a.shot.ready(a) {
		t.Fatal("must not capture twice")
	}
}

func TestScreenshotWithSendWaitsForTheResponse(t *testing.T) {
	srv := jsonServer(t)
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), SendOnStart: true})
	tab := a.cur()
	pointTabAt(tab, srv.URL)

	for range settleFrames + 2 {
		a.shotUpdate()
	}
	if a.shot.ready(a) && tab.Resp == nil {
		t.Fatal("with -send the capture must wait for the response")
	}
	if !tab.Sending && tab.Resp == nil {
		t.Fatal("-send must start the request on the first frame")
	}

	waitForResponse(t, a, tab)
	if !a.shot.ready(a) {
		t.Fatal("ready once the response is in")
	}
}

func TestScreenshotGivesUpWaitingForAResponse(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), SendOnStart: true})
	a.shot.sent = true // pretend the request was sent and never answers
	a.shot.frames = shotTimeout
	if !a.shot.ready(a) {
		t.Fatal("after the timeout the capture must go ahead without a response")
	}
}

// A socket never gets a response, so -send waits for its connection.
func TestScreenshotWithSendOnASocketWaitsForTheConnection(t *testing.T) {
	a := shotApp(t, withAdapters(Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), SendOnStart: true}))
	tab := a.cur()
	tab.Req.Protocol = "websocket"
	tab.Req.URL = echoServer(t)
	for range settleFrames + 2 {
		a.shotUpdate()
	}
	if a.shot.ready(a) && tab.Session == nil {
		t.Fatal("with -send the capture must wait for the connection")
	}
	until(t, a, "the connection", func() bool { return tab.Session != nil })
	if !a.shot.ready(a) {
		t.Fatal("ready once connected")
	}
	a.disconnect(tab)
}

func TestScreenshotWithSendOnASocketGoesAheadAfterAFailedConnect(t *testing.T) {
	a := shotApp(t, withAdapters(Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), SendOnStart: true}))
	tab := a.cur()
	tab.Req.Protocol = "websocket"
	tab.Req.URL = "ws://127.0.0.1:1/"
	for range settleFrames + 2 {
		a.shotUpdate()
	}
	until(t, a, "the failure", func() bool { return !tab.Connecting && tab.Session == nil && len(tab.wsView.Lines) > 0 })
	if !a.shot.ready(a) {
		t.Fatal("the error is what the shot should show")
	}
}

func TestScreenshotOpensTheNamedRequest(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), ScreenshotRequest: "Echo"})
	for range setFrame {
		if err := a.shotUpdate(); err != nil {
			t.Fatal(err)
		}
	}
	if got := a.cur().Req.Name; got != "Echo" {
		t.Errorf("open request = %q, want Echo", got)
	}
}

func TestScreenshotOpensTheNamedSettingsPage(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), ScreenshotPage: "colors"})
	for range setFrame {
		if err := a.shotUpdate(); err != nil {
			t.Fatal(err)
		}
	}
	if !a.menu || a.prefs.Cat != "colors" {
		t.Errorf("settings open = %v, page = %q, want the dialog on colors", a.menu, a.prefs.Cat)
	}
}

func TestScreenshotShowsTheSentLog(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), ScreenshotSentLog: true})
	for range setFrame {
		if err := a.shotUpdate(); err != nil {
			t.Fatal(err)
		}
	}
	if !a.showSent {
		t.Error("the sidebar should list the sent requests")
	}
}

// With -filter the capture types the text into the sidebar's filter, so the
// tree shows only the matches with the matched part highlighted.
func TestScreenshotFiltersTheTree(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), ScreenshotFilter: "Echo"})
	for range setFrame {
		if err := a.shotUpdate(); err != nil {
			t.Fatal(err)
		}
	}
	if a.filter.Text != "Echo" {
		t.Fatalf("filter %q, want %q", a.filter.Text, "Echo")
	}
}

// With -sending the capture sends the open request and waits until it is
// in flight, so the URL bar shows its Cancel button.
func TestScreenshotWithSendingCapturesTheRequestInFlight(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		select { // never answers: the capture must find the request still running
		case <-request.Context().Done():
		case <-release:
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })
	a := shotApp(t, withAdapters(Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), ScreenshotSending: true}))
	tab := a.cur()
	pointTabAt(tab, srv.URL)
	for range settleFrames {
		if err := a.shotUpdate(); err != nil {
			t.Fatal(err)
		}
	}
	if !tab.Sending {
		t.Fatal("-sending must start the request")
	}
	if !a.shot.ready(a) {
		t.Fatal("ready while the request is in flight")
	}
}

// Without its request in flight, -sending does not capture: the picture
// would show the Send button, not the Cancel one.
func TestScreenshotWithSendingWaitsForTheRequestToStart(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), ScreenshotSending: true})
	a.shot.frames = settleFrames
	if a.shot.ready(a) {
		t.Fatal("nothing is sending yet, the capture must wait")
	}
}

// With -send and -request the first request goes out at start-up and the
// named one once that is answered, so a capture of the sent log has two
// rows in a fixed order.
func TestScreenshotWithSendAndARequestSendsBothInTurn(t *testing.T) {
	srv := jsonServer(t)
	a := shotApp(t, withAdapters(Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), SendOnStart: true, ScreenshotRequest: "Get User"}))
	first := a.cur().Req
	pointTabAt(a.cur(), srv.URL)
	named := a.findRequestByName("Get User")
	named.URL = srv.URL
	for range 400 {
		a.shotUpdate()
		a.step(core.Input{})
		if a.shot.ready(a) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !a.shot.ready(a) {
		t.Fatal("never ready: the named request was not sent or not waited for")
	}
	rows := a.sent.Rows("") // as the capture finds them: both must be in
	if len(rows) != 2 || rows[0].Path != a.requestPath(named) || rows[1].Path != a.requestPath(first) {
		t.Fatalf("sent rows = %+v, want Get User then %s", rows, first.Name)
	}
}

// With -find the capture waits for the response, then opens the find bar on
// it and types the query, so the bar shows its matches.
func TestScreenshotFindsTheQueryInTheResponse(t *testing.T) {
	srv := jsonServer(t)
	a := shotApp(t, withAdapters(Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), SendOnStart: true, ScreenshotFind: "ok"}))
	tab := a.cur()
	pointTabAt(tab, srv.URL)
	for range 400 {
		a.shotUpdate()
		a.step(core.Input{})
		if a.shot.ready(a) {
			break
		}
		if tab.Resp == nil && tab.look.FindOpen {
			t.Fatal("the find bar opened before the response arrived")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !a.shot.ready(a) {
		t.Fatal("never ready")
	}
	if !tab.look.FindOpen || tab.find.Text != "ok" {
		t.Fatalf("find open = %v with %q, want open with %q", tab.look.FindOpen, tab.find.Text, "ok")
	}
	if len(tab.view.Matches()) == 0 {
		t.Error("the query should match the response")
	}
}

func TestScreenshotWithAnUnknownRequestFails(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), ScreenshotRequest: "No such request"})
	for range setFrame {
		a.shotUpdate()
	}
	if a.Err() == nil {
		t.Error("a request that is not in the collection must be reported, not silently shown as the default")
	}
}
