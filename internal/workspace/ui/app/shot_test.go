package ui

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
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

// With -open-all the capture opens every request of the collection in its
// own tab, so the strip of open requests overflows.
func TestScreenshotOpensEveryRequestInATab(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), ScreenshotOpenAll: true})
	for range setFrame {
		if err := a.shotUpdate(); err != nil {
			t.Fatal(err)
		}
	}
	var want []string
	walkRequests(&a.col.Folder, "", func(_ string, request *model.Request) { want = append(want, request.Name) })
	var got []string
	for _, tab := range a.openTabs() {
		got = append(got, tab.Req.Name)
	}
	slices.Sort(got) // the request open at start keeps its place
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("open tabs %q, want every request %q", got, want)
	}
}

// With -type-url the capture types the text into the open request's URL,
// so its tab shows the mark of unsaved changes.
func TestScreenshotTypesIntoTheURL(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), ScreenshotURL: "?page=2"})
	url := a.cur().Req.URL
	for range setFrame {
		if err := a.shotUpdate(); err != nil {
			t.Fatal(err)
		}
	}
	if got := a.cur().Req.URL; !strings.Contains(got, "?page=2") || len(got) != len(url)+len("?page=2") {
		t.Errorf("URL %q, want %q typed into %q", got, "?page=2", url)
	}
	if !a.tabDirty(a.cur()) {
		t.Error("the tab should be marked as changed")
	}
}

// The -type-url edit came from no input, so the cached unsaved marks of
// the frames drawn before it stayed: the golden scene showed the tab's
// mark only when a window event happened to refresh them.
func TestScreenshotURLEditShowsOverMarksDrawnBefore(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), ScreenshotURL: "?page=2"})
	for range setFrame - 1 {
		if err := a.shotUpdate(); err != nil {
			t.Fatal(err)
		}
	}
	if a.tabDirty(a.cur()) {
		t.Fatal("test setup: the tab must be unchanged before the edit")
	}
	a.shotUpdate()
	if !a.tabDirty(a.cur()) {
		t.Error("the tab should be marked as changed on the frame of the edit")
	}
}

// With -pane the capture shows the request pane with that label, after
// -request has opened its request, so a scene can draw a tab other than the
// first one.
func TestScreenshotShowsTheNamedRequestPane(t *testing.T) {
	cases := []struct {
		name    string
		request string
	}{
		{"http request", ""},
		{"socket request", "Echo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), ScreenshotRequest: tc.request, ScreenshotPane: "Settings"})
			for range setFrame {
				if err := a.shotUpdate(); err != nil {
					t.Fatal(err)
				}
			}
			if err := a.Err(); err != nil {
				t.Fatal(err)
			}
			tab := a.cur()
			if got := requestPanesFor(tab)[tab.paneTab].label; got != "Settings" {
				t.Errorf("request pane %q, want Settings", got)
			}
		})
	}
}

// An unknown pane ends the run with an error: capturing the first pane
// instead would let a golden pass for the wrong scene.
func TestScreenshotWithAnUnknownPaneFails(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), ScreenshotPane: "Nope"})
	for range setFrame {
		a.shotUpdate()
	}
	if a.Err() == nil || !a.shot.captured {
		t.Fatalf("err = %v, captured = %v, want the run ended with an error", a.Err(), a.shot.captured)
	}
}

// With -toast the capture shows each text as an error toast, oldest first:
// an error toast stays until it is clicked, so a slow frame cannot let it
// fade before the capture as an info toast would.
func TestScreenshotShowsTheToasts(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), ScreenshotToasts: []string{"First", "Second"}})
	for range setFrame {
		if err := a.shotUpdate(); err != nil {
			t.Fatal(err)
		}
	}
	want := widget.ToastStack{{Text: "First", IsErr: true}, {Text: "Second", IsErr: true}}
	if !slices.Equal(a.toasts, want) {
		t.Fatalf("toasts %+v, want %+v", a.toasts, want)
	}
}

// With -dialog the capture opens that dialog on the open request through
// the path its menu item takes, so a scene draws an app dialog.
func TestScreenshotOpensTheNamedDialog(t *testing.T) {
	cases := []struct {
		kind      string
		title     string
		input     string
		withInput bool
	}{
		{kind: "rename", title: "Rename", input: "Search Repos", withInput: true},
		{kind: "delete", title: "Delete Search Repos?"},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), ScreenshotDialog: tc.kind})
			for range setFrame {
				if err := a.shotUpdate(); err != nil {
					t.Fatal(err)
				}
			}
			if err := a.Err(); err != nil {
				t.Fatal(err)
			}
			if a.dialog == nil || a.dialog.Title != tc.title {
				t.Fatalf("dialog %+v, want the one titled %q", a.dialog, tc.title)
			}
			if got := a.dialog.Input != nil; got != tc.withInput {
				t.Fatalf("dialog has an input: %v, want %v", got, tc.withInput)
			}
			if tc.withInput && a.dialog.Input.Text != tc.input {
				t.Errorf("input %q, want %q", a.dialog.Input.Text, tc.input)
			}
		})
	}
}

// An unknown dialog ends the run with an error, so a golden never passes
// for a window without the dialog.
func TestScreenshotWithAnUnknownDialogFails(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), ScreenshotDialog: "nope"})
	for range setFrame {
		a.shotUpdate()
	}
	if a.Err() == nil || !a.shot.captured {
		t.Fatalf("err = %v, captured = %v, want the run ended with an error", a.Err(), a.shot.captured)
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
