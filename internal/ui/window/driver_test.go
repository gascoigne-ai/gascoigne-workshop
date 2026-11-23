package window

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/sends"
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
func driverApp(t *testing.T, opts startOptions) *Component {
	t.Helper()
	opts.SettingsPath = filepath.Join(t.TempDir(), "settings.json")
	a, err := newWindow(opts)
	if err != nil {
		t.Fatal(err)
	}
	a.w, a.h = winW, winH
	return a
}

// A request the first tab does not show can be opened by name; a name that is
// not in the collection is an error, not the default tab.
func TestDriverOpensTheNamedRequest(t *testing.T) {
	a := driverApp(t, startOptions{})
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
			err := tc.run(driverApp(t, withAdapters(startOptions{})).Driver())
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
		opts  startOptions
		run   func(driver *Driver) error
		check func(t *testing.T, a *Component)
	}{
		{"open the named settings page", startOptions{}, func(driver *Driver) error { driver.OpenSettingsPage("colors"); return nil }, func(t *testing.T, a *Component) {
			if !a.over.Settings.Open() || a.over.Settings.Shell.Cat != "colors" {
				t.Errorf("settings open = %v, page = %q, want the dialog on colors", a.over.Settings.Open(), a.over.Settings.Shell.Cat)
			}
		}},
		{"open the settings menu", startOptions{}, func(driver *Driver) error { driver.OpenSettingsMenu(); return nil }, func(t *testing.T, a *Component) {
			if !a.over.Settings.Open() {
				t.Error("the settings dialog should be open")
			}
		}},
		{"open the settings as text", startOptions{}, func(driver *Driver) error { driver.OpenRawSettings(); return nil }, func(t *testing.T, a *Component) {
			if !a.over.Settings.Open() || !a.over.Settings.Shell.Raw {
				t.Errorf("settings open = %v, raw = %v, want the dialog showing settings.json", a.over.Settings.Open(), a.over.Settings.Shell.Raw)
			}
		}},
		{"show the sent log", startOptions{}, func(driver *Driver) error { driver.ShowSentLog(); return nil }, func(t *testing.T, a *Component) {
			if !a.side.ShowSent() {
				t.Error("the sidebar should list the sent requests")
			}
		}},
		// Closing the last collection leaves the empty tree.
		{"close every collection", startOptions{}, func(driver *Driver) error { driver.CloseAllCollections(); return nil }, func(t *testing.T, a *Component) {
			if len(a.cols.Set().Entries) != 0 {
				t.Errorf("%d collections still open", len(a.cols.Set().Entries))
			}
		}},
		// The Log dialog opens as the status bar's Log button does.
		{"open the call log", startOptions{}, func(driver *Driver) error { driver.OpenCallLog(); return nil }, func(t *testing.T, a *Component) {
			if !a.over.Calls().Shown() {
				t.Error("the Log dialog should be open")
			}
		}},
		{"open the file browser on a folder", startOptions{Folders: folders.Disk{}}, func(driver *Driver) error { return driver.BrowseFiles(browseDir) }, func(t *testing.T, a *Component) {
			if !a.over.Browser.Shown() || filepath.Clean(a.over.Browser.Path()) != browseDir {
				t.Fatalf("browser shown %v on %q, want it open on %s", a.over.Browser.Shown(), a.over.Browser.Path(), browseDir)
			}
		}},
		{"show the named request pane of an http request", startOptions{}, func(driver *Driver) error { return driver.ShowRequestPane("Settings") }, requestPaneIsSettings},
		{"show the named request pane of a socket request", startOptions{}, func(driver *Driver) error {
			if err := driver.OpenRequest("Echo"); err != nil {
				return err
			}
			return driver.ShowRequestPane("Settings")
		}, requestPaneIsSettings},
		{"show the named response pane", startOptions{}, func(driver *Driver) error { return driver.ShowResponsePane("Tests") }, func(t *testing.T, a *Component) {
			tab := a.cur()
			if got := a.responsePanesFor(tab)[tab.respTab].label; got != "Tests" {
				t.Errorf("response pane %q, want Tests", got)
			}
		}},
		{"choose the file body type", startOptions{}, func(driver *Driver) error { return driver.ChooseBodyType(string(request.BodyFile)) }, bodyTypeIs(request.BodyFile)},
		{"choose the graphql body type", startOptions{}, func(driver *Driver) error { return driver.ChooseBodyType(string(request.BodyGraphQL)) }, bodyTypeIs(request.BodyGraphQL)},
		{"wrap the response and the body", startOptions{}, func(driver *Driver) error {
			if err := driver.OpenRequest("Echo"); err != nil {
				return err
			}
			driver.WrapOn()
			return nil
		}, func(t *testing.T, a *Component) {
			tab := a.cur()
			if tab.Req.Name != "Echo" {
				t.Fatalf("open request %q, want Echo: wrap must apply after the request is opened", tab.Req.Name)
			}
			if !a.resp.Code(tab.Tab).Wrap || !a.editor.Body.Area(tab.Tab).Wrap {
				t.Errorf("response wrap %v, body wrap %v, want both on", a.resp.Code(tab.Tab).Wrap, a.editor.Body.Area(tab.Tab).Wrap)
			}
		}},
		// Without WrapOn both stay off, the default the user decided.
		{"leave wrap off by default", startOptions{}, func(driver *Driver) error { return driver.OpenRequest("Echo") }, func(t *testing.T, a *Component) {
			if tab := a.cur(); a.resp.Code(tab.Tab).Wrap || a.editor.Body.Area(tab.Tab).Wrap {
				t.Errorf("response wrap %v, body wrap %v, want both off", a.resp.Code(tab.Tab).Wrap, a.editor.Body.Area(tab.Tab).Wrap)
			}
		}},
		// Each text is an error toast, oldest first.
		{"show the toasts", startOptions{}, func(driver *Driver) error { driver.ShowToast("First"); driver.ShowToast("Second"); return nil }, func(t *testing.T, a *Component) {
			want := widget.ToastStack{{Text: "First", Kind: widget.ToastError}, {Text: "Second", Kind: widget.ToastError}}
			if !slices.Equal(a.over.Toasts.Stack(), want) {
				t.Fatalf("toasts %+v, want %+v", a.over.Toasts.Stack(), want)
			}
		}},
		{"filter the tree", startOptions{}, func(driver *Driver) error { driver.FilterTree("Echo"); return nil }, func(t *testing.T, a *Component) {
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

func requestPaneIsSettings(t *testing.T, a *Component) {
	tab := a.cur()
	if got := a.editor.PaneLabel(tab.Tab); got != "Settings" {
		t.Errorf("request pane %q, want Settings", got)
	}
}

// Choosing a body type comes from no input, so the tab must be marked as
// changed through the stale mark.
func bodyTypeIs(want request.BodyType) func(t *testing.T, a *Component) {
	return func(t *testing.T, a *Component) {
		if got := a.cur().Req.BodyType; got != want {
			t.Errorf("body type %q, want %q", got, want)
		}
		if !a.keys.TabDirty(a.cur().Tab) {
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
	a := driverApp(t, withAdapters(startOptions{}))
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
	a := driverApp(t, withAdapters(startOptions{}))
	if err := a.Driver().AddCollection(t.TempDir()); err == nil || len(a.cols.Set().Entries) != 1 {
		t.Fatalf("err = %v, open %d, want an error and the start collection alone", err, len(a.cols.Set().Entries))
	}
}

func TestDriverOpensEveryRequestInATab(t *testing.T) {
	a := driverApp(t, startOptions{})
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
	a := driverApp(t, startOptions{})
	url := a.cur().Req.URL
	if a.keys.TabDirty(a.cur().Tab) {
		t.Fatal("test setup: the tab must be unchanged before the edit")
	}
	a.Driver().TypeURL("?page=2")
	if got := a.cur().Req.URL; !strings.Contains(got, "?page=2") || len(got) != len(url)+len("?page=2") {
		t.Errorf("URL %q, want %q typed into %q", got, "?page=2", url)
	}
	if !a.keys.TabDirty(a.cur().Tab) {
		t.Error("the tab should be marked as changed on the frame of the edit")
	}
}

func TestDriverTypesTheBody(t *testing.T) {
	const typed = "{\n  \"note\": \"typed\"\n}"
	a := driverApp(t, startOptions{})
	if err := a.Driver().ChooseBodyType(string(request.BodyJSON)); err != nil {
		t.Fatal(err)
	}
	if err := a.Driver().TypeBody(typed); err != nil {
		t.Fatal(err)
	}
	tab := a.cur()
	if a.editor.Body.Area(tab.Tab).Text != typed || tab.Req.Body != typed {
		t.Errorf("editor %q, request %q, want both %q", a.editor.Body.Area(tab.Tab).Text, tab.Req.Body, typed)
	}
	if !a.keys.TabDirty(tab.Tab) {
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
			a := driverApp(t, startOptions{})
			if err := a.Driver().OpenDialog(tc.kind); err != nil {
				t.Fatal(err)
			}
			if tc.picker {
				if a.over.Picker() == nil || a.over.Dialog() != nil {
					t.Fatalf("picker %v dialog %v, want the picker alone", a.over.Picker() != nil, a.over.Dialog() != nil)
				}
				return
			}
			if a.over.Dialog() == nil || a.over.Dialog().Title != tc.title {
				t.Fatalf("dialog %+v, want the one titled %q", a.over.Dialog(), tc.title)
			}
			if got := a.over.Dialog().Input != nil; got != tc.withInput {
				t.Fatalf("dialog has an input: %v, want %v", got, tc.withInput)
			}
			if tc.withInput && a.over.Dialog().Input.Text != tc.input {
				t.Errorf("input %q, want %q", a.over.Dialog().Input.Text, tc.input)
			}
		})
	}
}

func TestDriverRecordsShortcutKeys(t *testing.T) {
	a := driverApp(t, startOptions{})
	driver := a.Driver()
	driver.OpenSettingsPage("shortcuts")
	for _, spec := range []string{"close-tab=Ctrl+L", "new-request=J", "send=Ctrl+C"} {
		if err := driver.RecordShortcut(spec); err != nil {
			t.Fatal(err)
		}
	}
	if pending, ok := a.over.Settings.Shortcuts.Pending("close-tab"); !ok || pending.OtherName != "Focus URL" {
		t.Errorf("close-tab pending %+v, want a conflict with Focus URL", pending)
	}
	if got := a.settings.Shortcuts["new-request"]; got != "J" {
		t.Errorf("new-request is %q, want J", got)
	}
	if a.over.Settings.Shortcuts.Recording() != "send" || !strings.Contains(a.over.Settings.Shortcuts.Note("send"), "text editing") {
		t.Errorf("recording %q, note %q, want send still recording after its refused key", a.over.Settings.Shortcuts.Recording(), a.over.Settings.Shortcuts.Note("send"))
	}
}

// A bare action is left recording, so a scene shows "Press keys...".
func TestDriverLeavesABareActionRecording(t *testing.T) {
	a := driverApp(t, startOptions{})
	if err := a.Driver().RecordShortcut("close-tab"); err != nil {
		t.Fatal(err)
	}
	if a.over.Settings.Shortcuts.Recording() != "close-tab" {
		t.Fatalf("recording %q, want close-tab", a.over.Settings.Shortcuts.Recording())
	}
}

// A tab sent again still holds its earlier response, which is not the
// outcome of the new send.
func TestAnAnsweredSendIsTheLatestSendOfATab(t *testing.T) {
	a := driverApp(t, startOptions{})
	tab := a.cur()
	tab.Resp = &request.Response{Status: 200}
	number, _ := a.sends.Begin(tab.Req, func() {})
	if a.answered(tab) {
		t.Fatal("a tab still sending is not answered by the response of its earlier send")
	}
	a.sends.Accept(sends.Result{Owner: tab.Req, Number: number})
	if !a.answered(tab) {
		t.Fatal("answered once the send ended")
	}
}

func TestDriverSendIsAnsweredOnceTheResponseIsIn(t *testing.T) {
	srv := jsonServer(t)
	a := driverApp(t, withAdapters(startOptions{}))
	tab := a.cur()
	a.pointTabAt(tab, srv.URL)
	a.Driver().Send()
	if a.Driver().AllAnswered() {
		t.Fatal("a send without its response is not answered")
	}
	if !a.sends.Sending(tab.Req) && tab.Resp == nil {
		t.Fatal("Send must start the request")
	}
	waitForResponse(t, a, tab)
	if !a.Driver().AllAnswered() {
		t.Fatal("answered once the response is in")
	}
}

// A socket never gets a response, so its send is answered by the connection.
func TestDriverSendOnASocketIsAnsweredByTheConnection(t *testing.T) {
	a := driverApp(t, withAdapters(startOptions{}))
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
	a := driverApp(t, withAdapters(startOptions{}))
	tab := a.cur()
	tab.Req.Protocol = "websocket"
	tab.Req.URL = "ws://127.0.0.1:1/"
	a.Driver().Send()
	until(t, a, "the failure", func() bool {
		return tab.State() != tabs.ConnConnecting && tab.Session() == nil && len(a.editor.WS.Lines(tab.Tab)) > 0
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
	a := driverApp(t, withAdapters(startOptions{}))
	tab := a.cur()
	a.pointTabAt(tab, srv.URL)
	a.Driver().SendWithoutWaiting()
	if !a.sends.Sending(tab.Req) {
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
	a := driverApp(t, withAdapters(startOptions{}))
	driver := a.Driver()
	first := a.cur().Req
	a.pointTabAt(a.cur(), srv.URL)
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
	a := driverApp(t, withAdapters(startOptions{}))
	tab := a.cur()
	a.pointTabAt(tab, srv.URL)
	a.Driver().Send()
	waitForResponse(t, a, tab)
	a.Driver().Find("ok")
	if !a.resp.Look(tab.Tab).FindOpen || a.resp.Field(tab.Tab).Text != "ok" {
		t.Fatalf("find open = %v with %q, want open with %q", a.resp.Look(tab.Tab).FindOpen, a.resp.Field(tab.Tab).Text, "ok")
	}
	if len(a.resp.Code(tab.Tab).Matches()) == 0 {
		t.Error("the query should match the response")
	}
}

// The call form of a tool opens once the server has announced it; before that
// the action reports it is not ready, and a tool the server does not have is
// an error.
func TestDriverOpensTheFormOfTheNamedTool(t *testing.T) {
	a := driverApp(t, withAdapters(startOptions{}))
	if opened, err := a.Driver().OpenToolForm("search"); opened || err != nil || a.over.Dialog() != nil {
		t.Fatalf("opened %v err %v dialog %+v before the server announced a tool", opened, err, a.over.Dialog())
	}
	a.cur().Tools = []request.MCPTool{{Name: "ping"}, searchTool()}
	if opened, err := a.Driver().OpenToolForm("nope"); opened || err == nil {
		t.Fatalf("opened %v err %v, want an error for a tool the server did not announce", opened, err)
	}
	opened, err := a.Driver().OpenToolForm("search")
	if err != nil || !opened {
		t.Fatalf("opened %v err %v, want the form open", opened, err)
	}
	if a.over.Dialog() == nil || !strings.Contains(dialogDump(a), `text="query *"`) || a.over.Dialog().Title != "Call search" {
		t.Fatalf("dialog %+v, want the form of search", a.over.Dialog())
	}
}

// A window on its Proto pane is not settled until its files are read, so a
// capture of it waits.
func TestDriverIsNotSettledWhileTheProtoFilesAreRead(t *testing.T) {
	a := driverApp(t, startOptions{})
	tab := a.cur()
	tab.Req.Protocol = "grpc"
	tab.Req.Proto = request.ProtoSource{Files: []string{filepath.Join(t.TempDir(), "gone.proto")}}
	showRequestPane(t, a, tab, "Proto")
	a.view() // draws the pane, which starts reading the files
	if a.Driver().Settled() {
		t.Fatal("settled while the files are being read")
	}
	until(t, a, "the files are read", func() bool { return a.Driver().Settled() })
	if !strings.Contains(layoutDump(a), "fg=error") {
		t.Fatal("the files are read and what was found should show as a problem")
	}
	tab.Req.Proto.Files = append(tab.Req.Proto.Files, filepath.Join(t.TempDir(), "other.proto"))
	if a.Driver().Settled() {
		t.Fatal("settled with a source nobody has read")
	}
	showRequestPane(t, a, tab, "Body")
	if !a.Driver().Settled() {
		t.Fatal("another pane is shown: nothing to wait for")
	}
}
