package app

import (
	"errors"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/tabs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/adapter/folders"

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

// runScene builds an app that captures the scene made of steps, runs the
// frames up to the one that applies the scene and returns the app. A frame
// that ends the run with an error fails the test; a step that fails ends the
// capture through a.Err instead, which the caller checks.
func runScene(t *testing.T, opts Options, steps ...Step) *App {
	t.Helper()
	opts.ScreenshotPath = filepath.Join(t.TempDir(), "x.png")
	opts.Scene = steps
	a := shotApp(t, opts)
	for range setFrame {
		if err := a.shotUpdate(); err != nil {
			t.Fatal(err)
		}
	}
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

	a.shot.phase = shotOver // what shotDraw records after writing the file
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
	if !tab.Sending() && tab.Resp == nil {
		t.Fatal("-send must start the request on the first frame")
	}

	waitForResponse(t, a, tab)
	if !a.shot.ready(a) {
		t.Fatal("ready once the response is in")
	}
}

// -send with -request of the tab that is already open sends it twice, the
// second time after -type-url changed its URL. The first answer is still
// on the tab then, so the capture must wait until the second send ends,
// or it shows the old response.
func TestScreenshotWaitsForTheLatestSendOfATab(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png")})
	tab := a.cur()
	tab.Resp = &model.Response{Status: 200}
	tab.BeginSend()
	if answered(tab) {
		t.Fatal("a tab still sending is not answered by the response of its earlier send")
	}
	tab.Accept(tab.Seq)
	if !answered(tab) {
		t.Fatal("answered once the send ended")
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
	if a.shot.ready(a) && tab.Session() == nil {
		t.Fatal("with -send the capture must wait for the connection")
	}
	until(t, a, "the connection", func() bool { return tab.Session() != nil })
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
	until(t, a, "the failure", func() bool {
		return tab.State() != tabs.ConnConnecting && tab.Session() == nil && len(tab.wsView.Lines) > 0
	})
	if !a.shot.ready(a) {
		t.Fatal("the error is what the shot should show")
	}
}

// With -add-collection the capture opens each folder's collection beside
// the start one, in order, as Open collection folder does, so a scene shows
// several roots in the tree.
func TestScreenshotAddsCollectionsBesideTheStartOne(t *testing.T) {
	alphaDir, betaDir := savedCollection(t, "Alpha"), savedCollection(t, "Beta")
	a := shotApp(t, withAdapters(Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), Scene: []Step{AddCollection(alphaDir), AddCollection(betaDir)}}))
	start := a.col().Name
	for range setFrame {
		if err := a.shotUpdate(); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := openNames(a), []string{start, "Alpha", "Beta"}; !slices.Equal(got, want) {
		t.Fatalf("open %v, want %v", got, want)
	}
}

// A folder that holds no collection ends the run with an error: opening it
// would start an empty collection there, and a golden would pass for a tree
// without the collection it was meant to show.
func TestScreenshotWithAFolderWithoutACollectionFails(t *testing.T) {
	a := shotApp(t, withAdapters(Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), Scene: []Step{AddCollection(t.TempDir())}}))
	for range setFrame {
		a.shotUpdate()
	}
	if a.Err() == nil || a.shot.phase != shotOver || len(a.open.Entries) != 1 {
		t.Fatalf("err = %v, phase = %v, open %d, want the run ended with an error", a.Err(), a.shot.phase, len(a.open.Entries))
	}
}

// A capture of the Proto pane waits until its files are read, so what
// reading them found (a problem at its line) is in the picture.
func TestScreenshotOfTheProtoPaneWaitsForTheFilesToBeRead(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png")})
	tab := a.cur()
	tab.Req.Protocol = "grpc"
	tab.Req.Proto = model.ProtoSource{Files: []string{filepath.Join(t.TempDir(), "gone.proto")}}
	tab.paneTab = paneIndex(t, requestPanesFor(tab), "Proto")
	for range settleFrames {
		a.shotUpdate()
	}
	tab.proto.check = protoCheck{running: true}
	if a.shot.ready(a) {
		t.Fatal("ready while the files are being read")
	}
	tab.proto.check = protoCheck{source: tab.Req.Proto, problems: []string{"gone.proto: not found"}}
	if !a.shot.ready(a) {
		t.Fatal("the files are read, the capture should be ready")
	}
	tab.paneTab = paneIndex(t, requestPanesFor(tab), "Body")
	tab.proto.check = protoCheck{running: true}
	if !a.shot.ready(a) {
		t.Fatal("another pane is shown: nothing to wait for")
	}
}

// With -tool the capture opens the call form of that tool of the open MCP
// request, as picking it from the composer's tool menu does. The server
// announces its tools after the connection opens, so the capture waits for
// them.
func TestScreenshotOpensTheFormOfTheNamedTool(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), Scene: []Step{ToolForm("search")}, SendOnStart: true})
	a.shot.sent = true // pretend the connection is made; its tools come below
	for range settleFrames {
		a.shotUpdate()
	}
	if a.dialog != nil || a.shot.ready(a) {
		t.Fatalf("dialog %+v, ready %v before the server announced the tool", a.dialog, a.shot.ready(a))
	}
	a.cur().Tools = []model.MCPTool{{Name: "ping"}, searchTool()}
	a.shotUpdate()
	if err := a.Err(); err != nil {
		t.Fatal(err)
	}
	if a.dialog == nil || a.dialog.form == nil || a.dialog.Title != "Call search" {
		t.Fatalf("dialog %+v, want the form of search", a.dialog)
	}
	if !a.shot.ready(a) {
		t.Fatal("the form is open, the capture should be ready")
	}
}

// A -tool the server does not announce, or -tool without -send (nothing
// connects, so nothing announces it), used to wait for the shot timeout
// and capture without the form; it ends the run with an error, as an
// unknown -dialog or -pane does.
func TestScreenshotWithAToolItCannotOpenFails(t *testing.T) {
	cases := []struct {
		name string
		tool string
		send bool
	}{
		{"unknown tool", "nope", true},
		{"tool without send", "search", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), Scene: []Step{ToolForm(tc.tool)}, SendOnStart: tc.send})
			a.shot.sent = true // pretend the connection is made and its tools announced
			a.cur().Tools = []model.MCPTool{{Name: "ping"}, searchTool()}
			for range setFrame {
				a.shotUpdate()
			}
			if a.Err() == nil || a.shot.phase != shotOver {
				t.Fatalf("err = %v, phase = %v, want the run ended with an error", a.Err(), a.shot.phase)
			}
		})
	}
}

// With -open-all the capture opens every request of the collection in its
// own tab, so the strip of open requests overflows.
func TestScreenshotOpensEveryRequestInATab(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), Scene: []Step{OpenAll()}})
	for range setFrame {
		if err := a.shotUpdate(); err != nil {
			t.Fatal(err)
		}
	}
	var want []string
	walkRequests(&a.col().Folder, "", func(_ string, request *model.Request) { want = append(want, request.Name) })
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
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), Scene: []Step{TypeURL("?page=2")}})
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
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), Scene: []Step{TypeURL("?page=2")}})
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

// The -body choice comes from no input, so without a stale mark the
// unsaved marks cached by earlier frames would hide it, as they once hid
// the -type-url edit.
func TestScreenshotBodyChoiceShowsOverMarksDrawnBefore(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), Scene: []Step{ChooseBody(string(model.BodyFile))}})
	for range setFrame - 1 {
		if err := a.shotUpdate(); err != nil {
			t.Fatal(err)
		}
	}
	if a.tabDirty(a.cur()) {
		t.Fatal("test setup: the tab must be unchanged before the choice")
	}
	a.shotUpdate()
	if !a.tabDirty(a.cur()) {
		t.Error("the tab should be marked as changed on the frame of the choice")
	}
}

// With -type-body the capture replaces the open request's body text, as
// selecting all and typing would, so the request and its tab see the edit.
// The first sample request has no body, so -body chooses JSON first.
func TestScreenshotTypesTheBody(t *testing.T) {
	const typed = "{\n  \"note\": \"typed\"\n}"
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), Scene: []Step{ChooseBody(string(model.BodyJSON)), TypeBody(typed)}})
	for range setFrame - 1 {
		a.shotUpdate()
	}
	if a.tabDirty(a.cur()) {
		t.Fatal("test setup: the tab must be unchanged before the edit")
	}
	a.shotUpdate()
	if err := a.Err(); err != nil {
		t.Fatal(err)
	}
	tab := a.cur()
	if tab.body.Text != typed || tab.Req.Body != typed {
		t.Errorf("editor %q, request %q, want both %q", tab.body.Text, tab.Req.Body, typed)
	}
	if !a.tabDirty(tab) {
		t.Error("the tab should be marked as changed on the frame of the edit")
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
		picker    bool // opens the destination picker, not a dialog
	}{
		{kind: "rename", title: "Rename", input: "Search Repos", withInput: true},
		{kind: "delete", title: "Delete Search Repos?"},
		{kind: "move", picker: true},
		{kind: "copy", picker: true},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), Scene: []Step{OpenDialog(tc.kind)}})
			for range setFrame {
				if err := a.shotUpdate(); err != nil {
					t.Fatal(err)
				}
			}
			if err := a.Err(); err != nil {
				t.Fatal(err)
			}
			if tc.picker {
				if a.picker == nil || a.dialog != nil {
					t.Fatalf("picker %v dialog %v, want the picker alone", a.picker != nil, a.dialog != nil)
				}
				return
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
	a := shotApp(t, withAdapters(Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), Scene: []Step{Send()}}))
	tab := a.cur()
	pointTabAt(tab, srv.URL)
	for range settleFrames {
		if err := a.shotUpdate(); err != nil {
			t.Fatal(err)
		}
	}
	if !tab.Sending() {
		t.Fatal("-sending must start the request")
	}
	if !a.shot.ready(a) {
		t.Fatal("ready while the request is in flight")
	}
}

// Without its request in flight, -sending does not capture: the picture
// would show the Send button, not the Cancel one.
func TestScreenshotWithSendingWaitsForTheRequestToStart(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), Scene: []Step{Send()}})
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
	a := shotApp(t, withAdapters(Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), SendOnStart: true, Scene: []Step{OpenRequest("Get User")}}))
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
	a := shotApp(t, withAdapters(Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), SendOnStart: true, Scene: []Step{Find("ok")}}))
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
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), Scene: []Step{OpenRequest("No such request")}})
	for range setFrame {
		a.shotUpdate()
	}
	if a.Err() == nil {
		t.Error("a request that is not in the collection must be reported, not silently shown as the default")
	}
}

// With -record the capture records keys on the Shortcuts page through the
// path a key press takes, so a scene draws a row's conflict, its notes and
// its recording state.
func TestScreenshotRecordsShortcutKeys(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), Scene: []Step{SettingsPage("shortcuts"), Record("close-tab=Ctrl+L"), Record("new-request=J"), Record("send=Ctrl+C")}})
	for range setFrame {
		if err := a.shotUpdate(); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.Err(); err != nil {
		t.Fatal(err)
	}
	if pending, ok := a.shortcutKeys.Pending("close-tab"); !ok || pending.OtherName != "Focus URL" {
		t.Errorf("close-tab pending %+v, want a conflict with Focus URL", pending)
	}
	if got := a.settings.Shortcuts["new-request"]; got != "J" {
		t.Errorf("new-request is %q, want J", got)
	}
	if a.shortcutKeys.Recording() != "send" || !strings.Contains(a.shortcutKeys.Note("send"), "text editing") {
		t.Errorf("recording %q, note %q, want send still recording after its refused key", a.shortcutKeys.Recording(), a.shortcutKeys.Note("send"))
	}
}

// A bare action is left recording, so a scene shows "Press keys...".
func TestScreenshotLeavesABareActionRecording(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), Scene: []Step{Record("close-tab")}})
	for range setFrame {
		if err := a.shotUpdate(); err != nil {
			t.Fatal(err)
		}
	}
	if a.shortcutKeys.Recording() != "close-tab" {
		t.Fatalf("recording %q, want close-tab", a.shortcutKeys.Recording())
	}
}

// Scenes whose steps apply: each row runs the scene up to the frame that
// shows it and checks what the step changed in the app.
func TestScreenshotScenesApplyTheirSteps(t *testing.T) {
	browseDir := t.TempDir()
	cases := []struct {
		name  string
		opts  Options
		scene []Step
		check func(t *testing.T, a *App)
	}{
		{"open the named request", Options{}, []Step{OpenRequest("Echo")}, func(t *testing.T, a *App) {
			if got := a.cur().Req.Name; got != "Echo" {
				t.Errorf("open request = %q, want Echo", got)
			}
		}},
		{"open the named settings page", Options{}, []Step{SettingsPage("colors")}, func(t *testing.T, a *App) {
			if !a.menu || a.prefs.Cat != "colors" {
				t.Errorf("settings open = %v, page = %q, want the dialog on colors", a.menu, a.prefs.Cat)
			}
		}},
		{"open the settings as text", Options{}, []Step{RawSettings()}, func(t *testing.T, a *App) {
			if !a.menu || !a.prefs.Raw {
				t.Errorf("settings open = %v, raw = %v, want the dialog showing settings.json", a.menu, a.prefs.Raw)
			}
		}},
		{"show the sent log", Options{}, []Step{SentLog()}, func(t *testing.T, a *App) {
			if !a.showSent {
				t.Error("the sidebar should list the sent requests")
			}
		}},
		// With -close-all the capture shows the empty tree that closing the
		// last collection leaves.
		{"close every collection", Options{}, []Step{CloseAll()}, func(t *testing.T, a *App) {
			if len(a.open.Entries) != 0 {
				t.Errorf("%d collections still open", len(a.open.Entries))
			}
		}},
		// With -log the capture opens the Log dialog, as the status bar's Log
		// button does.
		{"open the call log", Options{}, []Step{CallLog()}, func(t *testing.T, a *App) {
			if !a.calls.Shown() {
				t.Error("the Log dialog should be open")
			}
		}},
		// With -browse the capture opens the in-app file browser on that
		// folder, as choosing a file without a dialog program does.
		{"open the file browser on a folder", Options{Folders: folders.Disk{}}, []Step{Browse(browseDir)}, func(t *testing.T, a *App) {
			if a.browser == nil || a.browser.model.Dir != browseDir {
				t.Fatalf("browser %+v, want it open on %s", a.browser, browseDir)
			}
		}},
		{"show the named request pane of an http request", Options{}, []Step{ShowPane("Settings")}, requestPaneIsSettings},
		{"show the named request pane of a socket request", Options{}, []Step{OpenRequest("Echo"), ShowPane("Settings")}, requestPaneIsSettings},
		// With -response-pane the capture shows the response pane with that
		// label, so a scene can draw the response's Tests list; -pane cannot,
		// as both sides have a Tests pane.
		{"show the named response pane", Options{}, []Step{ShowResponsePane("Tests")}, func(t *testing.T, a *App) {
			if err := a.Err(); err != nil {
				t.Fatal(err)
			}
			tab := a.cur()
			if got := responsePanesFor(tab)[tab.respTab].label; got != "Tests" {
				t.Errorf("response pane %q, want Tests", got)
			}
		}},
		{"choose the file body type", Options{}, []Step{ChooseBody(string(model.BodyFile))}, bodyTypeIs(model.BodyFile)},
		{"choose the graphql body type", Options{}, []Step{ChooseBody(string(model.BodyGraphQL))}, bodyTypeIs(model.BodyGraphQL)},
		// With -wrap the capture turns on Wrap for the open request's
		// response and body, as their two Wrap chips would, so one scene flag
		// covers both.
		{"wrap the response and the body", Options{}, []Step{OpenRequest("Echo"), WrapOn()}, func(t *testing.T, a *App) {
			if err := a.Err(); err != nil {
				t.Fatal(err)
			}
			tab := a.cur()
			if tab.Req.Name != "Echo" {
				t.Fatalf("open request %q, want Echo: -wrap must apply after -request", tab.Req.Name)
			}
			if !tab.view.Wrap || !tab.body.Wrap {
				t.Errorf("response wrap %v, body wrap %v, want both on", tab.view.Wrap, tab.body.Wrap)
			}
		}},
		// Without -wrap both stay off, the default the user decided.
		{"leave wrap off by default", Options{}, []Step{OpenRequest("Echo")}, func(t *testing.T, a *App) {
			if tab := a.cur(); tab.view.Wrap || tab.body.Wrap {
				t.Errorf("response wrap %v, body wrap %v, want both off", tab.view.Wrap, tab.body.Wrap)
			}
		}},
		// With -toast the capture shows each text as an error toast, oldest
		// first: an error toast stays until it is clicked, so a slow frame
		// cannot let it fade before the capture as an info toast would.
		{"show the toasts", Options{}, []Step{ShowToast("First"), ShowToast("Second")}, func(t *testing.T, a *App) {
			want := widget.ToastStack{{Text: "First", Kind: widget.ToastError}, {Text: "Second", Kind: widget.ToastError}}
			if !slices.Equal(a.toasts, want) {
				t.Fatalf("toasts %+v, want %+v", a.toasts, want)
			}
		}},
		// With -filter the capture types the text into the sidebar's filter,
		// so the tree shows only the matches with the matched part
		// highlighted.
		{"filter the tree", Options{}, []Step{Filter("Echo")}, func(t *testing.T, a *App) {
			if a.filter.Text != "Echo" {
				t.Fatalf("filter %q, want %q", a.filter.Text, "Echo")
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.check(t, runScene(t, tc.opts, tc.scene...))
		})
	}
}

// With -pane the capture shows the request pane with that label, after
// -request has opened its request, so a scene can draw a tab other than the
// first one.
func requestPaneIsSettings(t *testing.T, a *App) {
	if err := a.Err(); err != nil {
		t.Fatal(err)
	}
	tab := a.cur()
	if got := requestPanesFor(tab)[tab.paneTab].label; got != "Settings" {
		t.Errorf("request pane %q, want Settings", got)
	}
}

// With -body the capture chooses that body type on the open request, as its
// chip does, so a scene can draw the File or GraphQL body editor that no
// sample request has. The tab shows its unsaved mark on that frame.
func bodyTypeIs(want model.BodyType) func(t *testing.T, a *App) {
	return func(t *testing.T, a *App) {
		if err := a.Err(); err != nil {
			t.Fatal(err)
		}
		if got := a.cur().Req.BodyType; got != want {
			t.Errorf("body type %q, want %q", got, want)
		}
		if !a.tabDirty(a.cur()) {
			t.Error("the tab should be marked as changed on the frame of the choice")
		}
	}
}

// Scenes whose step cannot apply end the run with an error instead of
// capturing the first pane, dialog or body type in its place: a golden would
// pass for the wrong picture.
func TestScreenshotScenesThatCannotApplyEndTheRunWithAnError(t *testing.T) {
	cases := []struct {
		name  string
		scene []Step
	}{
		{"unknown response pane", []Step{ShowResponsePane("Nope")}},
		{"unknown request pane", []Step{ShowPane("Nope")}},
		{"unknown body type", []Step{ChooseBody("Nope")}},
		// A body type without a text editor (none, file, form...) would not
		// show the typed text.
		{"typing a body without a text editor", []Step{ChooseBody(string(model.BodyFile)), TypeBody("x")}},
		{"unknown dialog", []Step{OpenDialog("nope")}},
		{"record of an unknown action", []Step{Record("nope=Ctrl+K")}},
		{"record of keys that do not parse", []Step{Record("close-tab=Ctrl+Nope")}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := runScene(t, Options{}, tc.scene...)
			if a.Err() == nil || a.shot.phase != shotOver {
				t.Fatalf("err = %v, phase = %v, want the run ended with an error", a.Err(), a.shot.phase)
			}
		})
	}
}
