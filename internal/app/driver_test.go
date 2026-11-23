package app

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/adapter/folders"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// driverApp builds an app whose settings live in a temp dir, for tests of the
// Driver actions.
func driverApp(t *testing.T, opts Options) *App {
	t.Helper()
	opts.SettingsPath = filepath.Join(t.TempDir(), "settings.json")
	a, err := NewApp(opts)
	if err != nil {
		t.Fatal(err)
	}
	a.w, a.h = winW, winH
	return a
}

// A request the first tab does not show can be opened by name; a name that is
// not in the collection is an error, not the default tab.
func TestDriverOpensTheNamedRequest(t *testing.T) {
	a := driverApp(t, Options{})
	if err := a.Driver().OpenRequest("Echo"); err != nil {
		t.Fatal(err)
	}
	if got := a.cur().Req.Name; got != "Echo" {
		t.Errorf("open request = %q, want Echo", got)
	}
	if err := a.Driver().OpenRequest("No such request"); err == nil {
		t.Error("a request that is not in the collection must be reported, not silently shown as the default")
	}
}

// Each action that cannot apply returns its message, so a scene never shows
// a picture that looks right and is not what was asked for.
func TestDriverActionsThatCannotApplyReturnTheirMessage(t *testing.T) {
	cases := []struct {
		name string
		run  func(driver *Driver) error
		want string // empty: any error
	}{
		{"unknown request", func(driver *Driver) error { return driver.OpenRequest("Nope") }, `request "Nope" is not in the collection`},
		{"unknown pane", func(driver *Driver) error { return driver.ShowRequestPane("Nope") }, `pane "Nope" is not a pane of the open request`},
		{"unknown response pane", func(driver *Driver) error { return driver.ShowResponsePane("Nope") }, `response pane "Nope" is not a response pane of the open request`},
		{"unknown body type", func(driver *Driver) error { return driver.ChooseBodyType("Nope") }, `body type "Nope" is not one of the body chips`},
		{"unknown shortcut action", func(driver *Driver) error { return driver.RecordShortcut("nope") }, `record: "nope" is not a shortcut action`},
		{"record of an unknown action with keys", func(driver *Driver) error { return driver.RecordShortcut("nope=Ctrl+K") }, ""},
		{"record of keys that do not parse", func(driver *Driver) error { return driver.RecordShortcut("close-tab=Ctrl+Nope") }, ""},
		{"unknown dialog", func(driver *Driver) error { return driver.OpenDialog("nope") }, `dialog "nope" is not one of rename, delete, move, copy`},
		{"folder without a collection", func(driver *Driver) error { return driver.AddCollection("") }, ""},
		// A body type without a text editor (none, file, form...) would not
		// show the typed text.
		{"typing a body without a text editor", func(driver *Driver) error {
			if err := driver.ChooseBodyType(string(request.BodyFile)); err != nil {
				return err
			}
			return driver.TypeBody("x")
		}, `body text needs a JSON or Text body, the open request's is "file"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run(driverApp(t, withAdapters(Options{})).Driver())
			if err == nil {
				t.Fatal("want an error")
			}
			if tc.want != "" && err.Error() != tc.want {
				t.Errorf("err = %q, want %q", err, tc.want)
			}
		})
	}
}

// Actions that apply: each row runs them and checks the state the window is in.
func TestDriverActionsApply(t *testing.T) {
	browseDir := t.TempDir()
	cases := []struct {
		name  string
		opts  Options
		run   func(driver *Driver) error
		check func(t *testing.T, a *App)
	}{
		{"open the named settings page", Options{}, func(driver *Driver) error { driver.OpenSettingsPage("colors"); return nil }, func(t *testing.T, a *App) {
			if !a.menu || a.prefs.Cat != "colors" {
				t.Errorf("settings open = %v, page = %q, want the dialog on colors", a.menu, a.prefs.Cat)
			}
		}},
		{"open the settings menu", Options{}, func(driver *Driver) error { driver.OpenSettingsMenu(); return nil }, func(t *testing.T, a *App) {
			if !a.menu {
				t.Error("the settings dialog should be open")
			}
		}},
		{"open the settings as text", Options{}, func(driver *Driver) error { driver.OpenRawSettings(); return nil }, func(t *testing.T, a *App) {
			if !a.menu || !a.prefs.Raw {
				t.Errorf("settings open = %v, raw = %v, want the dialog showing settings.json", a.menu, a.prefs.Raw)
			}
		}},
		{"show the sent log", Options{}, func(driver *Driver) error { driver.ShowSentLog(); return nil }, func(t *testing.T, a *App) {
			if !a.side.ShowSent() {
				t.Error("the sidebar should list the sent requests")
			}
		}},
		// Closing the last collection leaves the empty tree.
		{"close every collection", Options{}, func(driver *Driver) error { driver.CloseAllCollections(); return nil }, func(t *testing.T, a *App) {
			if len(a.open.Entries) != 0 {
				t.Errorf("%d collections still open", len(a.open.Entries))
			}
		}},
		// The Log dialog opens as the status bar's Log button does.
		{"open the call log", Options{}, func(driver *Driver) error { driver.OpenCallLog(); return nil }, func(t *testing.T, a *App) {
			if !a.calls.Shown() {
				t.Error("the Log dialog should be open")
			}
		}},
		{"open the file browser on a folder", Options{Folders: folders.Disk{}}, func(driver *Driver) error { return driver.BrowseFiles(browseDir) }, func(t *testing.T, a *App) {
			if a.browser == nil || a.browser.model.Dir != browseDir {
				t.Fatalf("browser %+v, want it open on %s", a.browser, browseDir)
			}
		}},
		{"show the named request pane of an http request", Options{}, func(driver *Driver) error { return driver.ShowRequestPane("Settings") }, requestPaneIsSettings},
		{"show the named request pane of a socket request", Options{}, func(driver *Driver) error {
			if err := driver.OpenRequest("Echo"); err != nil {
				return err
			}
			return driver.ShowRequestPane("Settings")
		}, requestPaneIsSettings},
		{"show the named response pane", Options{}, func(driver *Driver) error { return driver.ShowResponsePane("Tests") }, func(t *testing.T, a *App) {
			tab := a.cur()
			if got := responsePanesFor(tab)[tab.respTab].label; got != "Tests" {
				t.Errorf("response pane %q, want Tests", got)
			}
		}},
		{"choose the file body type", Options{}, func(driver *Driver) error { return driver.ChooseBodyType(string(request.BodyFile)) }, bodyTypeIs(request.BodyFile)},
		{"choose the graphql body type", Options{}, func(driver *Driver) error { return driver.ChooseBodyType(string(request.BodyGraphQL)) }, bodyTypeIs(request.BodyGraphQL)},
		{"wrap the response and the body", Options{}, func(driver *Driver) error {
			if err := driver.OpenRequest("Echo"); err != nil {
				return err
			}
			driver.WrapOn()
			return nil
		}, func(t *testing.T, a *App) {
			tab := a.cur()
			if tab.Req.Name != "Echo" {
				t.Fatalf("open request %q, want Echo: wrap must apply after the request is opened", tab.Req.Name)
			}
			if !tab.view.Wrap || !tab.body.Wrap {
				t.Errorf("response wrap %v, body wrap %v, want both on", tab.view.Wrap, tab.body.Wrap)
			}
		}},
		// Without WrapOn both stay off, the default the user decided.
		{"leave wrap off by default", Options{}, func(driver *Driver) error { return driver.OpenRequest("Echo") }, func(t *testing.T, a *App) {
			if tab := a.cur(); tab.view.Wrap || tab.body.Wrap {
				t.Errorf("response wrap %v, body wrap %v, want both off", tab.view.Wrap, tab.body.Wrap)
			}
		}},
		// Each text is an error toast, oldest first.
		{"show the toasts", Options{}, func(driver *Driver) error { driver.ShowToast("First"); driver.ShowToast("Second"); return nil }, func(t *testing.T, a *App) {
			want := widget.ToastStack{{Text: "First", Kind: widget.ToastError}, {Text: "Second", Kind: widget.ToastError}}
			if !slices.Equal(a.toasts, want) {
				t.Fatalf("toasts %+v, want %+v", a.toasts, want)
			}
		}},
		{"filter the tree", Options{}, func(driver *Driver) error { driver.FilterTree("Echo"); return nil }, func(t *testing.T, a *App) {
			if a.side.Filter().Text != "Echo" {
				t.Fatalf("filter %q, want %q", a.side.Filter().Text, "Echo")
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := driverApp(t, tc.opts)
			if err := tc.run(a.Driver()); err != nil {
				t.Fatal(err)
			}
			tc.check(t, a)
		})
	}
}

func requestPaneIsSettings(t *testing.T, a *App) {
	tab := a.cur()
	if got := requestPanesFor(tab)[tab.paneTab].label; got != "Settings" {
		t.Errorf("request pane %q, want Settings", got)
	}
}

// Choosing a body type comes from no input, so the tab must be marked as
// changed through the stale mark.
func bodyTypeIs(want request.BodyType) func(t *testing.T, a *App) {
	return func(t *testing.T, a *App) {
		if got := a.cur().Req.BodyType; got != want {
			t.Errorf("body type %q, want %q", got, want)
		}
		if !a.tabDirty(a.cur()) {
			t.Error("the tab should be marked as changed on the frame of the choice")
		}
	}
}

// Each face has its own size key, and ChangeSetting reaches the live theme for all three.
func TestDriverCanSetTheSizeOfEachFace(t *testing.T) {
	a := testApp(t)
	for key, size := range map[string]int{"fontSize": 16, "smallFontSize": 8, "codeFontSize": 32} {
		if err := a.Driver().ChangeSetting(key, settings.Number(size)); err != nil {
			t.Fatal(err)
		}
	}
	th := a.ctx.Th
	if th.FontSize != 16 || th.SmallSize != 8 || th.MonoSize != 32 {
		t.Fatalf("theme sizes = %v, %v, %v; want 16, 8, 32", th.FontSize, th.SmallSize, th.MonoSize)
	}
}

func TestDriverChangesSettingsLiveThroughTheSamePathAsTheDialog(t *testing.T) {
	a := testApp(t)
	driver := a.Driver()
	for key, value := range map[string]settings.Value{"flat": settings.Flag(true), "contrast": settings.Text("high"), "uiScale": settings.Number(120)} {
		if err := driver.ChangeSetting(key, value); err != nil {
			t.Fatal(err)
		}
	}
	if !a.settings.Flat || a.settings.Contrast != "high" || a.settings.UIScalePct != 120 {
		t.Fatalf("flat %v contrast %q scale %d", a.settings.Flat, a.settings.Contrast, a.settings.UIScalePct)
	}
	if !a.ctx.Th.NoGradients {
		t.Fatal("the theme did not follow the live change")
	}
}

// AddCollection opens each folder's collection beside the start one.
func TestDriverAddsCollectionsBesideTheStartOne(t *testing.T) {
	alphaDir, betaDir := savedCollection(t, "Alpha"), savedCollection(t, "Beta")
	a := driverApp(t, withAdapters(Options{}))
	start := a.col().Name
	for _, dir := range []string{alphaDir, betaDir} {
		if err := a.Driver().AddCollection(dir); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := openNames(a), []string{start, "Alpha", "Beta"}; !slices.Equal(got, want) {
		t.Fatalf("open %v, want %v", got, want)
	}
}

// A folder that holds no collection is an error: opening it would start an
// empty one and the scene would pass with the wrong picture.
func TestDriverRefusesAFolderWithoutACollection(t *testing.T) {
	a := driverApp(t, withAdapters(Options{}))
	if err := a.Driver().AddCollection(t.TempDir()); err == nil || len(a.open.Entries) != 1 {
		t.Fatalf("err = %v, open %d, want an error and the start collection alone", err, len(a.open.Entries))
	}
}

func TestDriverOpensEveryRequestInATab(t *testing.T) {
	a := driverApp(t, Options{})
	a.Driver().OpenAllRequests()
	var want []string
	walkRequests(&a.col().Folder, "", func(_ string, call *request.Request) { want = append(want, call.Name) })
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

// Typing into the URL comes from no input, so the cached unsaved marks of
// the frames drawn before must not hide the edit.
func TestDriverTypesIntoTheURL(t *testing.T) {
	a := driverApp(t, Options{})
	url := a.cur().Req.URL
	if a.tabDirty(a.cur()) {
		t.Fatal("test setup: the tab must be unchanged before the edit")
	}
	a.Driver().TypeURL("?page=2")
	if got := a.cur().Req.URL; !strings.Contains(got, "?page=2") || len(got) != len(url)+len("?page=2") {
		t.Errorf("URL %q, want %q typed into %q", got, "?page=2", url)
	}
	if !a.tabDirty(a.cur()) {
		t.Error("the tab should be marked as changed on the frame of the edit")
	}
}

func TestDriverTypesTheBody(t *testing.T) {
	const typed = "{\n  \"note\": \"typed\"\n}"
	a := driverApp(t, Options{})
	if err := a.Driver().ChooseBodyType(string(request.BodyJSON)); err != nil {
		t.Fatal(err)
	}
	if err := a.Driver().TypeBody(typed); err != nil {
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

func TestDriverOpensTheNamedDialog(t *testing.T) {
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
			a := driverApp(t, Options{})
			if err := a.Driver().OpenDialog(tc.kind); err != nil {
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

func TestDriverRecordsShortcutKeys(t *testing.T) {
	a := driverApp(t, Options{})
	driver := a.Driver()
	driver.OpenSettingsPage("shortcuts")
	for _, spec := range []string{"close-tab=Ctrl+L", "new-request=J", "send=Ctrl+C"} {
		if err := driver.RecordShortcut(spec); err != nil {
			t.Fatal(err)
		}
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
func TestDriverLeavesABareActionRecording(t *testing.T) {
	a := driverApp(t, Options{})
	if err := a.Driver().RecordShortcut("close-tab"); err != nil {
		t.Fatal(err)
	}
	if a.shortcutKeys.Recording() != "close-tab" {
		t.Fatalf("recording %q, want close-tab", a.shortcutKeys.Recording())
	}
}

// A tab sent again still holds its earlier response, which is not the
// outcome of the new send.
func TestAnAnsweredSendIsTheLatestSendOfATab(t *testing.T) {
	a := driverApp(t, Options{})
	tab := a.cur()
	tab.Resp = &request.Response{Status: 200}
	tab.BeginSend()
	if answered(tab) {
		t.Fatal("a tab still sending is not answered by the response of its earlier send")
	}
	tab.Accept(tab.Seq)
	if !answered(tab) {
		t.Fatal("answered once the send ended")
	}
}

func TestDriverSendIsAnsweredOnceTheResponseIsIn(t *testing.T) {
	srv := jsonServer(t)
	a := driverApp(t, withAdapters(Options{}))
	tab := a.cur()
	pointTabAt(tab, srv.URL)
	a.Driver().Send()
	if a.Driver().AllAnswered() {
		t.Fatal("a send without its response is not answered")
	}
	if !tab.Sending() && tab.Resp == nil {
		t.Fatal("Send must start the request")
	}
	waitForResponse(t, a, tab)
	if !a.Driver().AllAnswered() {
		t.Fatal("answered once the response is in")
	}
}

// A socket never gets a response, so its send is answered by the connection.
func TestDriverSendOnASocketIsAnsweredByTheConnection(t *testing.T) {
	a := driverApp(t, withAdapters(Options{}))
	tab := a.cur()
	tab.Req.Protocol = "websocket"
	tab.Req.URL = echoServer(t)
	a.Driver().Send()
	if a.Driver().AllAnswered() && tab.Session() == nil {
		t.Fatal("the send must wait for the connection")
	}
	until(t, a, "the connection", func() bool { return tab.Session() != nil })
	if !a.Driver().AllAnswered() {
		t.Fatal("answered once connected")
	}
	a.disconnect(tab)
}

func TestDriverSendOnASocketIsAnsweredByAFailedConnect(t *testing.T) {
	a := driverApp(t, withAdapters(Options{}))
	tab := a.cur()
	tab.Req.Protocol = "websocket"
	tab.Req.URL = "ws://127.0.0.1:1/"
	a.Driver().Send()
	until(t, a, "the failure", func() bool {
		return tab.State() != tabs.ConnConnecting && tab.Session() == nil && len(tab.wsView.Lines) > 0
	})
	if !a.Driver().AllAnswered() {
		t.Fatal("the error is what the shot should show")
	}
}

// SendWithoutWaiting leaves the request in flight and does not hold a capture
// back for its outcome.
func TestDriverSendWithoutWaitingCapturesTheRequestInFlight(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, call *http.Request) {
		select { // never answers: the request must still be running
		case <-call.Context().Done():
		case <-release:
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })
	a := driverApp(t, withAdapters(Options{}))
	tab := a.cur()
	pointTabAt(tab, srv.URL)
	a.Driver().SendWithoutWaiting()
	if !tab.Sending() {
		t.Fatal("SendWithoutWaiting must start the request")
	}
	if !a.Driver().AllAnswered() {
		t.Fatal("nothing is waited for")
	}
}

// A request opened after the first send goes out when the first is answered,
// so the sent log keeps its order.
func TestDriverSendsTheNextRequestAfterTheEarlierAnswers(t *testing.T) {
	srv := jsonServer(t)
	a := driverApp(t, withAdapters(Options{}))
	driver := a.Driver()
	first := a.cur().Req
	pointTabAt(a.cur(), srv.URL)
	named := a.findRequestByName("Get User")
	named.URL = srv.URL
	driver.Send()
	if err := driver.OpenRequest("Get User"); err != nil {
		t.Fatal(err)
	}
	driver.SendAfterAnswers()
	done := func() bool { return !driver.SendsPending() && driver.AllAnswered() }
	for range 400 {
		driver.ServeSends()
		a.Step(core.Input{})
		if done() {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !done() {
		t.Fatal("never answered: the named request was not sent or not waited for")
	}
	rows := a.side.Sent().Rows("")
	if len(rows) != 2 || rows[0].Path != a.requestPath(named) || rows[1].Path != a.requestPath(first) {
		t.Fatalf("sent rows = %+v, want Get User then %s", rows, first.Name)
	}
}

// Find on the response waits for the response only because the caller does;
// the action itself opens the bar and runs the query.
func TestDriverFindsTheQueryInTheResponse(t *testing.T) {
	srv := jsonServer(t)
	a := driverApp(t, withAdapters(Options{}))
	tab := a.cur()
	pointTabAt(tab, srv.URL)
	a.Driver().Send()
	waitForResponse(t, a, tab)
	a.Driver().Find("ok")
	if !tab.look.FindOpen || tab.find.Text != "ok" {
		t.Fatalf("find open = %v with %q, want open with %q", tab.look.FindOpen, tab.find.Text, "ok")
	}
	if len(tab.view.Matches()) == 0 {
		t.Error("the query should match the response")
	}
}

// The call form of a tool opens once the server has announced it; before that
// the action reports it is not ready, and a tool the server does not have is
// an error.
func TestDriverOpensTheFormOfTheNamedTool(t *testing.T) {
	a := driverApp(t, withAdapters(Options{}))
	if opened, err := a.Driver().OpenToolForm("search"); opened || err != nil || a.dialog != nil {
		t.Fatalf("opened %v err %v dialog %+v before the server announced a tool", opened, err, a.dialog)
	}
	a.cur().Tools = []request.MCPTool{{Name: "ping"}, searchTool()}
	if opened, err := a.Driver().OpenToolForm("nope"); opened || err == nil {
		t.Fatalf("opened %v err %v, want an error for a tool the server did not announce", opened, err)
	}
	opened, err := a.Driver().OpenToolForm("search")
	if err != nil || !opened {
		t.Fatalf("opened %v err %v, want the form open", opened, err)
	}
	if a.dialog == nil || a.dialog.form == nil || a.dialog.Title != "Call search" {
		t.Fatalf("dialog %+v, want the form of search", a.dialog)
	}
}

// A window on its Proto pane is not settled until its files are read, so a
// capture of it waits.
func TestDriverIsNotSettledWhileTheProtoFilesAreRead(t *testing.T) {
	a := driverApp(t, Options{})
	tab := a.cur()
	tab.Req.Protocol = "grpc"
	tab.Req.Proto = request.ProtoSource{Files: []string{filepath.Join(t.TempDir(), "gone.proto")}}
	tab.paneTab = paneIndex(t, requestPanesFor(tab), "Proto")
	tab.proto.check = protoCheck{running: true}
	if a.Driver().Settled() {
		t.Fatal("settled while the files are being read")
	}
	tab.proto.check = protoCheck{source: tab.Req.Proto, problems: []string{"gone.proto: not found"}}
	if !a.Driver().Settled() {
		t.Fatal("the files are read, the window should be settled")
	}
	tab.paneTab = paneIndex(t, requestPanesFor(tab), "Body")
	tab.proto.check = protoCheck{running: true}
	if !a.Driver().Settled() {
		t.Fatal("another pane is shown: nothing to wait for")
	}
}
