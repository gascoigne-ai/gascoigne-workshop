package ui

import (
	"errors"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/adapter/store"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/collection"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/adapter/proto"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/adapter/history"
	rport "github.com/gascoigne-ai/gascoigne-workshop/internal/response/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/ui/respview"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/adapter/fonts"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/adapter/settingsfile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/ui/shortcutpage"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/adapter/statefile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/theming"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/hajimehoshi/ebiten/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// These tests drive the whole window without a GPU: step() builds the
// view, lays it out and delivers synthetic input exactly as Update does.
// Coordinates follow the layout of a 1280x800 window: sidebar 260 wide,
// header 44, request tabs 40, URL bar 56 (centre y 112), tree rows 28.
const (
	winW, winH = 1280, 800
	urlBarY    = headerH + tabsH + urlH/2
	sendX      = winW - 12 - 40
	menuBtnX   = winW - 12 - headerH/2
	menuBtnY   = headerH / 2
)

func treeRowY(i int) int { return headerH + i*core.RowH + core.RowH/2 }

func newTestApp(t *testing.T) (*App, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.json")
	// Most tests were written against the pretty-printed response; the
	// compact default has its own tests (newCompactTestApp).
	s := settings.Default()
	s.ResponseView = "pretty"
	if err := settingsfile.Save(path, s); err != nil {
		t.Fatal(err)
	}
	return startTestApp(t, path)
}

// newCompactTestApp is a test app with the settings defaults.
func newCompactTestApp(t *testing.T) *App {
	t.Helper()
	a, _ := startTestApp(t, filepath.Join(t.TempDir(), "settings.json"))
	return a
}

func startTestApp(t *testing.T, path string) (*App, string) {
	t.Helper()
	a, err := NewApp(withAdapters(Options{SettingsPath: path}))
	if err != nil {
		t.Fatal(err)
	}
	a.w, a.h = winW, winH
	a.step(core.Input{}) // first frame: lay everything out once
	return a, path
}

// withAdapters gives opts the real history and font adapters, as main does.
func withAdapters(opts Options) Options {
	opts.History = func(dir string) rport.History { return history.Dir(dir) }
	opts.Fonts = fonts.System{}
	opts.Protocols = proto.Configured
	opts.GRPCMethods = proto.ListGRPCMethods
	opts.Curl = proto.Curl
	opts.Store = store.Files{}
	opts.States = statefile.Files{}
	opts.SettingsFiles = settingsfile.Files{}
	return opts
}

// click is a press followed by a release at the same place.
func click(a *App, x, y int) {
	a.step(core.Input{X: x, Y: y, Click: true, Held: true})
	a.step(core.Input{X: x, Y: y})
}

// jsonServer answers every request with a small JSON body.
func jsonServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"ok":true}`)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// pointTabAt makes the open request target url, as if typed into the field.
func pointTabAt(t *reqTab, url string) {
	t.Req.URL = url
	t.url = view.NewTextField(url)
}

func waitForResponse(t *testing.T, a *App, tab *reqTab) {
	t.Helper()
	for range 400 {
		a.step(core.Input{})
		if tab.Resp != nil {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("no response arrived")
}

func TestClickingARequestInTheTreeOpensItsTab(t *testing.T) {
	a, _ := newTestApp(t)
	if len(a.tabs) != 1 || a.cur().Req.Name != "Search Repos" {
		t.Fatalf("start: %d tabs, current %q", len(a.tabs), a.cur().Req.Name)
	}

	click(a, 100, treeRowY(4)) // Repository Info
	if len(a.tabs) != 2 || a.cur().Req.Name != "Repository Info" {
		t.Fatalf("after click: %d tabs, current %q", len(a.tabs), a.cur().Req.Name)
	}

	click(a, 100, treeRowY(4)) // the same request again focuses its tab
	if len(a.tabs) != 2 {
		t.Fatalf("reopening must not add a tab, have %d", len(a.tabs))
	}
}

func TestClickingAFolderTogglesIt(t *testing.T) {
	a, _ := newTestApp(t)
	before := len(collection.Rows(&a.col.Folder, a.expanded))
	click(a, 100, treeRowY(1)) // User folder, collapsed at start
	if got := len(collection.Rows(&a.col.Folder, a.expanded)); got != before+1 {
		t.Fatalf("rows %d -> %d, want one more (Get User)", before, got)
	}
	click(a, 100, treeRowY(1))
	if got := len(collection.Rows(&a.col.Folder, a.expanded)); got != before {
		t.Fatalf("second click must collapse again, rows = %d", got)
	}
}

func TestNewRequestAndSwitchingTabs(t *testing.T) {
	a, _ := newTestApp(t)
	a.newRequest()
	if len(a.tabs) != 2 || a.active != 1 || a.cur().Req.Name != "New Request" {
		t.Fatalf("tabs %d active %d", len(a.tabs), a.active)
	}
	click(a, 280, headerH+tabsH/2) // the first tab sits at the left edge of the strip
	if a.active != 0 {
		t.Fatalf("active = %d, want 0", a.active)
	}
}

func TestProtocolAndMethodChipsCycle(t *testing.T) {
	a, _ := newTestApp(t)
	req := a.cur().Req

	click(a, 380, urlBarY) // method chip
	if req.Method != "POST" {
		t.Fatalf("method = %q, want POST", req.Method)
	}
	click(a, 300, urlBarY) // protocol chip
	if req.Protocol != "websocket" {
		t.Fatalf("protocol = %q, want websocket", req.Protocol)
	}
	click(a, 300, urlBarY)
	click(a, 300, urlBarY)
	click(a, 300, urlBarY)
	click(a, 300, urlBarY)
	if req.Protocol != "rest" || req.Method != "POST" {
		t.Fatalf("after a full cycle: %q %q", req.Protocol, req.Method)
	}
}

func TestSwitchingToRESTRestoresAMethod(t *testing.T) {
	a, _ := newTestApp(t)
	req := a.cur().Req
	req.Protocol, req.Method = "grpc", ""
	a.cycleProtocol(a.cur())
	if req.Protocol != "rest" || req.Method != "GET" {
		t.Fatalf("got %q %q, want rest GET", req.Protocol, req.Method)
	}
}

func TestTypingInTheURLFieldUpdatesParams(t *testing.T) {
	a, _ := newTestApp(t)
	req := a.cur().Req
	click(a, 1100, urlBarY) // inside the field, past the end of the text: cursor at the end
	a.step(core.Input{Typed: []rune("&page=2")})

	if !strings.HasSuffix(req.URL, "&page=2") {
		t.Fatalf("url = %q", req.URL)
	}
	last := req.Params[len(req.Params)-1]
	if last.Name != "page" || last.Value != "2" || !last.Enabled {
		t.Fatalf("params = %+v", req.Params)
	}
	if got := a.cur().url.Text; got != req.URL {
		t.Fatalf("field %q and request %q disagree", got, req.URL)
	}
}

func TestTogglingAParamRewritesTheURL(t *testing.T) {
	a, _ := newTestApp(t)
	req := a.cur().Req
	click(a, 310, paramsRow0Y) // checkbox of the first Params row (q)
	if req.Params[0].Enabled || strings.Contains(req.URL, "q=") {
		t.Fatalf("q must be disabled and gone from %q", req.URL)
	}
	if !strings.Contains(req.URL, "order=desc") {
		t.Fatalf("other params must stay in %q", req.URL)
	}
	click(a, 310, paramsRow0Y)
	if !req.Params[0].Enabled || !strings.Contains(req.URL, "q=gioui") {
		t.Fatalf("toggling back must restore q in %q", req.URL)
	}
}

func TestDeletingAParamRewritesTheURL(t *testing.T) {
	a, _ := newTestApp(t)
	req := a.cur().Req
	click(a, 739, paramsRow0Y) // delete button of the first Params row
	if len(req.Params) != 2 || req.Params[0].Name != "order" || strings.Contains(req.URL, "q=") {
		t.Fatalf("params %+v url %q", req.Params, req.URL)
	}
}

func TestSendShowsTheResponse(t *testing.T) {
	srv := jsonServer(t)
	a, _ := newTestApp(t)
	tab := a.cur()
	pointTabAt(tab, srv.URL)

	click(a, sendX, urlBarY)
	if !tab.Sending {
		t.Fatal("a click on Send must start a request")
	}
	waitForResponse(t, a, tab)

	if tab.Sending || tab.Resp.Status != 200 || tab.Resp.Err != "" {
		t.Fatalf("sending %v resp %+v", tab.Sending, tab.Resp)
	}
	if len(tab.view.Lines) != 3 { // pretty-printed {"ok":true}
		t.Fatalf("response view has %d lines", len(tab.view.Lines))
	}
	if !strings.Contains(a.status, "200 OK") {
		t.Fatalf("status = %q", a.status)
	}
}

func TestEnterInTheURLFieldSends(t *testing.T) {
	srv := jsonServer(t)
	a, _ := newTestApp(t)
	tab := a.cur()
	pointTabAt(tab, srv.URL)

	click(a, 1100, urlBarY) // focus the field
	a.step(core.Input{Enter: true})
	waitForResponse(t, a, tab)
	if tab.Resp.Status != 200 {
		t.Fatalf("resp = %+v", tab.Resp)
	}
}

// A second send while one is running (the shortcut, since a click on the
// button now cancels) must not start another request.
func TestSendingTwiceWhileRunningStartsOneRequest(t *testing.T) {
	hits := make(chan struct{}, 4)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits <- struct{}{}
		<-release
	}))
	defer srv.Close()
	a, _ := newTestApp(t)
	tab := a.cur()
	pointTabAt(tab, srv.URL)

	click(a, sendX, urlBarY)
	a.step(key(ebiten.KeyEnter, true, false))
	a.step(key(ebiten.KeyEnter, true, false))
	<-hits
	close(release)
	waitForResponse(t, a, tab)
	if len(hits) != 0 {
		t.Fatal("the second click must not start another request")
	}
}

func TestFailedSendShowsTheError(t *testing.T) {
	srv := jsonServer(t)
	url := srv.URL
	srv.Close() // nothing listens any more
	a, _ := newTestApp(t)
	tab := a.cur()
	pointTabAt(tab, url)

	click(a, sendX, urlBarY)
	waitForResponse(t, a, tab)
	if tab.Resp.Err == "" || a.status != "Request failed" {
		t.Fatalf("err %q status %q", tab.Resp.Err, a.status)
	}
}

func TestOtherProtocolsAnswerThroughTheSameUI(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	click(a, 300, urlBarY)   // rest -> websocket
	click(a, sendX, urlBarY) // connect: an https URL is no WebSocket address
	until(t, a, "failed connect", func() bool { return !tab.connecting && strings.Contains(logText(tab), "not ws or wss") })
	if tab.sess != nil {
		t.Fatal("connected to an https URL")
	}
	click(a, 300, urlBarY) // websocket -> mcp
	if tab.Req.Protocol != "mcp" {
		t.Fatalf("protocol %q", tab.Req.Protocol)
	}
	tab.Req.URL = "ftp://nowhere.test" // no network in tests
	click(a, sendX, urlBarY)
	until(t, a, "failed connect", func() bool { return !tab.connecting && strings.Contains(logText(tab), "not an http or https URL") })
}

func TestSettingsMenuOpensChangesThemeAndSaves(t *testing.T) {
	a, path := newTestApp(t)
	click(a, menuBtnX, menuBtnY)
	if !a.menu {
		t.Fatal("the three-dot button must open the menu")
	}

	a.prefs.Search = view.NewTextField("theme") // brings the Theme setting to the top
	clickText(t, a, "Light", 0)
	if a.settings.Theme != "light" || !a.menu {
		t.Fatalf("theme %q, menu open %v (choosing must keep it open)", a.settings.Theme, a.menu)
	}
	if a.ctx.Th.Color(core.Bg) != theming.Light().Color(core.Bg) {
		t.Fatal("the live theme did not change")
	}
	saved, err := settingsfile.Load(path)
	if err != nil || saved.Theme != "light" {
		t.Fatalf("saved %+v err %v", saved, err)
	}

	a.prefs.Search = view.NewTextField("text size")
	clickText(t, a, "Large", 0)
	if a.ctx.Th.FontSize != 16 {
		t.Fatalf("font size = %v, want 16", a.ctx.Th.FontSize)
	}
	if saved, _ := settingsfile.Load(path); saved.TextSize != "large" || saved.Theme != "light" {
		t.Fatalf("saved %+v", saved)
	}
}

func TestMenuClosesOnEscapeAndOnOutsideClick(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, menuBtnX, menuBtnY)
	a.step(core.Input{Escape: true})
	if a.menu {
		t.Fatal("Escape must close the menu")
	}

	click(a, menuBtnX, menuBtnY)
	click(a, 100, treeRowY(4)) // outside the menu, on a request row
	if a.menu {
		t.Fatal("an outside click must close the menu")
	}
	if len(a.tabs) != 1 {
		t.Fatal("the click that closes the menu must not reach the tree below")
	}

	click(a, menuBtnX, menuBtnY)
	click(a, menuBtnX, menuBtnY) // the button again toggles it shut
	if a.menu {
		t.Fatal("the button must toggle the menu")
	}
}

func TestSettingsSaveFailureIsReported(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := NewApp(withAdapters(Options{SettingsPath: filepath.Join(blocker, "settings.json")})) // a file where a directory is needed
	if err != nil {
		t.Fatal(err)
	}
	a.changeSettings(func(s *settings.Settings) { s.Theme = "light" })
	if !strings.HasPrefix(a.status, "Could not save settings:") {
		t.Fatalf("status = %q", a.status)
	}
	if a.settings.Theme != "light" {
		t.Fatal("the change must still apply in memory")
	}
}

func TestBrokenSettingsFileFallsBackAndSaysSo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte("{nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := NewApp(withAdapters(Options{SettingsPath: path}))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a.settings, settings.Default()) || !strings.Contains(a.status, "parse settings") {
		t.Fatalf("settings %+v status %q", a.settings, a.status)
	}
}

func TestSettingsStartFromTheSavedChoice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := settingsfile.Save(path, settings.Settings{Theme: "light", TextSize: "small"}); err != nil {
		t.Fatal(err)
	}
	a, err := NewApp(withAdapters(Options{SettingsPath: path, OpenMenu: true}))
	if err != nil {
		t.Fatal(err)
	}
	if a.ctx.Th.Color(core.Bg) != theming.Light().Color(core.Bg) || a.ctx.Th.FontSize != 13 || !a.menu {
		t.Fatalf("theme/size/menu not restored: font %v menu %v", a.ctx.Th.FontSize, a.menu)
	}
}

// Every pane of both areas, with and without a response, must build and
// lay out; a pane that reads a nil response would panic here.
func TestEveryPaneBuildsInEveryResponseState(t *testing.T) {
	srv := jsonServer(t)
	a, _ := newTestApp(t)
	tab := a.cur()

	states := map[string]func(){
		"no response": func() { tab.Resp, tab.Sending = nil, false },
		"sending":     func() { tab.Resp, tab.Sending = nil, true },
		"error":       func() { tab.Resp, tab.Sending = &respErr, false },
		"ok": func() {
			tab.Sending = false
			pointTabAt(tab, srv.URL)
			a.send(tab)
			waitForResponse(t, a, tab)
		},
	}
	for _, name := range []string{"no response", "sending", "error", "ok"} {
		states[name]()
		for i := range requestPanes {
			for j := range responsePanes {
				tab.paneTab, tab.respTab = i, j
				a.step(core.Input{}) // panics on a bad pane
			}
		}
	}
}

func TestTabItemsCountEnabledRowsAndHeaders(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	items := a.tabItems(requestPanes, tab)
	want := map[string]int{"Params": 3, "Body": 0, "Headers": 0, "Auth": 0, "Vars": 1}
	for _, it := range items {
		if it.Badge != want[it.Label] {
			t.Errorf("%s badge = %d, want %d", it.Label, it.Badge, want[it.Label])
		}
	}
	tab.Req.Params[0].Enabled = false
	if got := a.tabItems(requestPanes, tab)[0].Badge; got != 2 {
		t.Errorf("disabled params must not count, badge = %d", got)
	}
}

func TestNoResponseStates(t *testing.T) {
	tests := []struct {
		name    string
		tab     reqTab
		wantMsg string
		wantErr bool
	}{
		{"sending", reqTab{Tab: &tabs.Tab{Sending: true}}, "Sending...", false},
		{"not sent", reqTab{Tab: &tabs.Tab{}}, "Send the request to see the response", false},
		{"failed", reqTab{Tab: &tabs.Tab{Resp: &respErr}}, "boom", true},
		{"ok", reqTab{Tab: &tabs.Tab{Resp: &respOK}}, "", false},
	}
	for _, tt := range tests {
		msg, isErr := noResponse(&tt.tab)
		if msg != tt.wantMsg || isErr != tt.wantErr {
			t.Errorf("%s: got %q %v", tt.name, msg, isErr)
		}
	}
}

// Cells of the Params table at 1280x800: request pane starts at x=260,
// the table at x=272, columns: grip 20, actions 36, name 110, value 157, description 126.
const (
	paramsRow0Y   = 264
	paramsRow3Y   = 348 // the placeholder row below the three params
	nameCellX     = 365
	valueCellX    = 500
	descCellX     = 650
	farAwayX      = 900
	farAwayY      = 600
	headersTabIdx = 2
)

func (a *App) typeText(s string) { a.step(core.Input{Typed: []rune(s)}) }

func TestEditingAParamValueUpdatesTheURL(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, valueCellX, paramsRow0Y)
	if a.edit == nil || a.edit.row != 0 || a.edit.col != 1 {
		t.Fatalf("edit = %+v", a.edit)
	}
	a.typeText("rust") // the old text is selected, so typing replaces it
	a.step(core.Input{Enter: true})

	req := a.cur().Req
	if a.edit != nil || req.Params[0].Value != "rust" || !strings.Contains(req.URL, "q=rust") {
		t.Fatalf("edit %+v params %+v url %q", a.edit, req.Params, req.URL)
	}
	if a.cur().url.Text != req.URL {
		t.Fatal("the URL field must show the rewritten URL")
	}
}

func TestEditingAParamNameAndDescription(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, nameCellX, paramsRow0Y)
	a.typeText("search")
	a.step(core.Input{Enter: true})
	click(a, descCellX, paramsRow0Y)
	a.typeText("what to find")
	a.step(core.Input{Enter: true})

	p := a.cur().Req.Params[0]
	if p.Name != "search" || p.Description != "what to find" || !strings.Contains(a.cur().Req.URL, "search=gioui") {
		t.Fatalf("param %+v url %q", p, a.cur().Req.URL)
	}
}

func TestEscapeCancelsACellEdit(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, valueCellX, paramsRow0Y)
	a.typeText("zzz")
	a.step(core.Input{Escape: true})
	if a.edit != nil || a.cur().Req.Params[0].Value != "gioui" {
		t.Fatalf("edit %+v value %q", a.edit, a.cur().Req.Params[0].Value)
	}
}

func TestClickingElsewhereCommitsACellEdit(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, valueCellX, paramsRow0Y)
	a.typeText("go")
	click(a, farAwayX, farAwayY)
	a.step(core.Input{})
	if a.edit != nil || a.cur().Req.Params[0].Value != "go" {
		t.Fatalf("edit %+v value %q", a.edit, a.cur().Req.Params[0].Value)
	}
}

func TestClickingAnotherCellCommitsAndMovesTheEdit(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, valueCellX, paramsRow0Y)
	a.typeText("go")
	click(a, valueCellX, paramsRow0Y+core.RowH) // the order row
	if a.cur().Req.Params[0].Value != "go" || a.edit == nil || a.edit.row != 1 {
		t.Fatalf("value %q edit %+v", a.cur().Req.Params[0].Value, a.edit)
	}
}

func TestTabMovesToTheNextCellAndShiftTabBack(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, nameCellX, paramsRow0Y)
	a.typeText("k")
	a.step(core.Input{Tab: true})
	if a.edit == nil || a.edit.row != 0 || a.edit.col != 1 || a.cur().Req.Params[0].Name != "k" {
		t.Fatalf("after Tab: edit %+v name %q", a.edit, a.cur().Req.Params[0].Name)
	}
	a.step(core.Input{Tab: true})
	a.step(core.Input{Tab: true}) // description, then the next row's name
	if a.edit == nil || a.edit.row != 1 || a.edit.col != 0 {
		t.Fatalf("Tab past the last column must wrap to the next row, edit %+v", a.edit)
	}
	a.step(core.Input{Tab: true, Shift: true})
	if a.edit == nil || a.edit.row != 0 || a.edit.col != 2 {
		t.Fatalf("Shift+Tab goes back, edit %+v", a.edit)
	}
}

func TestTypingInThePlaceholderRowAddsARow(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, nameCellX, paramsRow3Y)
	a.typeText("page")
	a.step(core.Input{Tab: true})
	a.typeText("2")
	a.step(core.Input{Enter: true})

	req := a.cur().Req
	if len(req.Params) != 4 {
		t.Fatalf("params = %+v", req.Params)
	}
	if got := req.Params[3]; got.Name != "page" || got.Value != "2" || !got.Enabled {
		t.Fatalf("new row = %+v", got)
	}
	if !strings.HasSuffix(req.URL, "&page=2") {
		t.Fatalf("url = %q", req.URL)
	}
}

func TestLeavingThePlaceholderRowEmptyAddsNothing(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, nameCellX, paramsRow3Y)
	a.step(core.Input{Enter: true})
	if len(a.cur().Req.Params) != 3 || a.edit != nil {
		t.Fatalf("params %+v edit %+v", a.cur().Req.Params, a.edit)
	}
}

func TestHeadersTableIsEditableToo(t *testing.T) {
	a, _ := newTestApp(t)
	a.cur().paneTab = headersTabIdx
	a.step(core.Input{})
	click(a, nameCellX, paramsRow0Y) // headers have no rows: the placeholder is the first row
	a.typeText("X-Test")
	a.step(core.Input{Tab: true})
	a.typeText("1")
	a.step(core.Input{Enter: true})
	h := a.cur().Req.Headers
	if len(h) != 1 || h[0].Name != "X-Test" || h[0].Value != "1" || !h[0].Enabled {
		t.Fatalf("headers = %+v", h)
	}
}

func TestResponseHeadersAreNotEditable(t *testing.T) {
	srv := jsonServer(t)
	a, _ := newTestApp(t)
	tab := a.cur()
	pointTabAt(tab, srv.URL)
	a.send(tab)
	waitForResponse(t, a, tab)
	tab.respTab = 1 // Headers
	a.step(core.Input{})
	click(a, 1000, paramsRow0Y)
	if a.edit != nil {
		t.Fatalf("response headers must stay read-only, edit %+v", a.edit)
	}
}

func key(k ebiten.Key, ctrl, shift bool) core.Input {
	return core.Input{Keys: []ebiten.Key{k}, Ctrl: ctrl, Shift: shift}
}

func TestCtrlEnterSendsTheRequest(t *testing.T) {
	srv := jsonServer(t)
	a, _ := newTestApp(t)
	tab := a.cur()
	pointTabAt(tab, srv.URL)
	a.step(key(ebiten.KeyEnter, true, false))
	waitForResponse(t, a, tab)
	if tab.Resp.Status != 200 {
		t.Fatalf("resp = %+v", tab.Resp)
	}
}

func TestCtrlLFocusesAndSelectsTheURL(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, 1100, 600) // click on empty space so nothing is focused
	a.step(key(ebiten.KeyL, true, false))
	url := &a.cur().url
	if !url.Focused || url.SelectedText() != url.Text {
		t.Fatalf("focused %v selection %q", url.Focused, url.SelectedText())
	}
}

func TestCtrlTOpensAndCtrlWClosesTabs(t *testing.T) {
	a, _ := newTestApp(t)
	a.step(key(ebiten.KeyT, true, false))
	if len(a.tabs) != 2 || a.active != 1 {
		t.Fatalf("Ctrl+T: %d tabs, active %d", len(a.tabs), a.active)
	}
	a.step(key(ebiten.KeyW, true, false))
	if len(a.tabs) != 1 || a.active != 0 || a.cur().Req.Name != "Search Repos" {
		t.Fatalf("Ctrl+W: %d tabs, active %d (%s)", len(a.tabs), a.active, a.cur().Req.Name)
	}
	a.step(key(ebiten.KeyW, true, false))
	if len(a.tabs) != 1 || a.cur().Req.Name != "New Request" {
		t.Fatalf("closing the last tab must leave a blank request, got %d tabs (%s)", len(a.tabs), a.cur().Req.Name)
	}
}

func TestCtrlTabSwitchesTabsAndWraps(t *testing.T) {
	a, _ := newTestApp(t)
	a.newRequest()
	a.newRequest() // three tabs, the last active
	a.step(key(ebiten.KeyTab, true, false))
	if a.active != 0 {
		t.Fatalf("Ctrl+Tab from the last tab must wrap to the first, active %d", a.active)
	}
	a.step(key(ebiten.KeyTab, true, true))
	if a.active != 2 {
		t.Fatalf("Ctrl+Shift+Tab must go back and wrap, active %d", a.active)
	}
}

func TestCtrlCommaOpensTheSettingsMenu(t *testing.T) {
	a, _ := newTestApp(t)
	a.step(key(ebiten.KeyComma, true, false))
	if !a.menu {
		t.Fatal("Ctrl+, must open the settings menu")
	}
}

func TestShortcutsNeedTheirModifiers(t *testing.T) {
	a, _ := newTestApp(t)
	for _, k := range []ebiten.Key{ebiten.KeyT, ebiten.KeyW, ebiten.KeyL, ebiten.KeyEnter, ebiten.KeyComma, ebiten.KeyTab} {
		a.step(key(k, false, false)) // plain letters must never trigger anything
	}
	if len(a.tabs) != 1 || a.menu || a.cur().Sending {
		t.Fatalf("plain keys changed state: %d tabs, menu %v", len(a.tabs), a.menu)
	}
}

func TestShortcutTableHasUniqueBindingsAndLabels(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range shortcuts {
		id := s.label()
		if s.name == "" || s.run == nil || seen[id] {
			t.Errorf("bad or duplicate shortcut %q (%q)", s.name, id)
		}
		seen[id] = true
	}
	if !seen["Ctrl+Enter"] || !seen["Ctrl+Shift+Tab"] {
		t.Fatalf("labels = %v", seen)
	}
}

func TestTreeScrollsWhenItDoesNotFit(t *testing.T) {
	a, _ := newTestApp(t)
	a.h = 200                       // the tree area is about 128px: five rows
	for _, i := range []int{0, 2} { // User and Realtime
		a.expanded[a.col.Folders[i]] = true
	}
	a.step(core.Input{})
	rowsN := len(collection.Rows(&a.col.Folder, a.expanded))
	if rowsN*core.RowH <= a.h-headerH-statusH {
		t.Fatalf("test setup: %d rows fit, nothing to scroll", rowsN)
	}

	a.step(core.Input{X: 100, Y: 100, Wheel: -5})
	if a.treeScroll.Y <= 0 {
		t.Fatalf("tree scroll = %d after wheeling over it", a.treeScroll.Y)
	}

	// scroll to the end, then click the last row: it must open that request
	a.step(core.Input{X: 100, Y: 100, Wheel: -50})
	last := collection.Rows(&a.col.Folder, a.expanded)[rowsN-1]
	click(a, 100, a.h-statusH-core.RowH/2)
	if a.cur().Req != last.Req {
		t.Fatalf("clicking the bottom row opened %q, want %q", a.cur().Req.Name, last.Name)
	}
}

// Body pane at 1280x800: type chips at y=206 (None, JSON, Text, Form,
// Multipart from x=272), the Format button at the right end of that
// row, the editor below y=228, a form table's placeholder row at y=270.
const (
	bodyTabIdx = 1
	chipsY     = 208
	noneChipX  = 300
	jsonChipX  = 374
	textChipX  = 444
	formChipX  = 514
	multiChipX = 600
	formatX    = 722
	editorX    = 500
	editorY    = 320
	formRow0Y  = 304
)

func openBodyPane(t *testing.T) *App {
	a, _ := newTestApp(t)
	a.cur().paneTab = bodyTabIdx
	a.step(core.Input{})
	return a
}

func TestBodyTypeChipsSwitchTheTypeAndKeepTheText(t *testing.T) {
	a := openBodyPane(t)
	req := a.cur().Req
	if req.BodyType != model.BodyNone {
		t.Fatalf("a request without a body starts as none, got %q", req.BodyType)
	}
	click(a, textChipX, chipsY)
	if req.BodyType != model.BodyText {
		t.Fatalf("type = %q", req.BodyType)
	}
	click(a, editorX, editorY)
	a.typeText("hello")
	click(a, jsonChipX, chipsY)
	click(a, formChipX, chipsY)
	click(a, textChipX, chipsY)
	if req.Body != "hello" {
		t.Fatalf("the text must survive switching types, Body = %q", req.Body)
	}
	for chip, want := range map[int]string{noneChipX: model.BodyNone, multiChipX: model.BodyMultipart, formChipX: model.BodyForm} {
		click(a, chip, chipsY)
		if req.BodyType != want {
			t.Errorf("chip at %d gave %q, want %q", chip, req.BodyType, want)
		}
	}
}

func TestTypingInTheBodyEditor(t *testing.T) {
	a := openBodyPane(t)
	click(a, jsonChipX, chipsY)
	click(a, editorX, editorY)
	a.typeText(`{"a":1}`)
	a.step(core.Input{Enter: true})
	a.typeText("x")
	if got := a.cur().Req.Body; got != "{\"a\":1}\nx" {
		t.Fatalf("Body = %q", got)
	}
}

func TestFormatButtonIndentsValidJSONOnly(t *testing.T) {
	a := openBodyPane(t)
	req := a.cur().Req
	click(a, jsonChipX, chipsY)
	req.Body = `{"a":1,"b":[1,2]}`
	a.cur().body.SelectAll()
	a.cur().body.Insert(req.Body)
	req.Body = a.cur().body.Text

	click(a, formatX, chipsY)
	if want := "{\n  \"a\": 1,\n  \"b\": [\n    1,\n    2\n  ]\n}"; req.Body != want {
		t.Fatalf("Body = %q", req.Body)
	}

	req.Body = `{"a":`
	a.cur().body.SelectAll()
	a.cur().body.Insert(req.Body)
	click(a, formatX, chipsY)
	if req.Body != `{"a":` || !strings.Contains(a.status, "Cannot format") {
		t.Fatalf("Body %q status %q: invalid JSON must be left alone and reported", req.Body, a.status)
	}
}

func TestFormatButtonOnlyExistsForJSON(t *testing.T) {
	a := openBodyPane(t)
	req := a.cur().Req
	click(a, textChipX, chipsY)
	req.Body = `{"a":1}`
	a.cur().body.SelectAll()
	a.cur().body.Insert(req.Body)
	click(a, formatX, chipsY)
	if req.Body != `{"a":1}` {
		t.Fatalf("a text body must not be reformatted, Body = %q", req.Body)
	}
}

func TestFormBodyFieldsAreEditable(t *testing.T) {
	a := openBodyPane(t)
	req := a.cur().Req
	click(a, formChipX, chipsY)
	click(a, nameCellX, formRow0Y)
	a.typeText("user")
	a.step(core.Input{Tab: true})
	a.typeText("ann")
	a.step(core.Input{Enter: true})
	if len(req.Form) != 1 || req.Form[0].Name != "user" || req.Form[0].Value != "ann" || !req.Form[0].Enabled {
		t.Fatalf("Form = %+v", req.Form)
	}
	if req.Body != "" {
		t.Fatalf("form fields must not touch the text body, Body = %q", req.Body)
	}
}

func TestSendingAFormBodyUsesTheRows(t *testing.T) {
	got := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got <- r.Header.Get("Content-Type") + "|" + string(b)
	}))
	defer srv.Close()
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.Req.Method, tab.Req.BodyType = "POST", model.BodyForm
	tab.Req.Form = []model.KV{{Name: "q", Value: "go lang", Enabled: true}}
	pointTabAt(tab, srv.URL)
	a.send(tab)
	waitForResponse(t, a, tab)
	if want := "application/x-www-form-urlencoded|q=go+lang"; <-got != want {
		t.Fatal("the sent request must carry the form body")
	}
}

func TestEachTabKeepsItsOwnBodyEditor(t *testing.T) {
	a := openBodyPane(t)
	click(a, textChipX, chipsY)
	click(a, editorX, editorY)
	a.typeText("first")
	a.newRequest()
	second := a.cur()
	second.paneTab = bodyTabIdx
	a.step(core.Input{})
	click(a, textChipX, chipsY)
	click(a, editorX, editorY)
	a.typeText("second")
	a.switchTab(-1)
	if a.cur().Req.Body != "first" || second.Req.Body != "second" {
		t.Fatalf("bodies %q and %q", a.cur().Req.Body, second.Req.Body)
	}
}

// tabX is the left edge and width of tab i in the request-tab strip,
// from the layout the view uses: 14px padding, a fixed badge slot, 8px
// gaps, the name, a fixed slot for the unsaved dot and a 24px close
// button. Nothing but the name changes a tab's width.
func (a *App) tabX(i int) (left, width int) {
	x := 260
	for j, t := range a.tabs {
		wd := 14 + tabBadgeW + 8 + a.ctx.Width(a.ctx.F.UI, t.Req.Name) + 8 + tabDirtyW + 24
		if j == i {
			return x, wd
		}
		x += wd
	}
	return x, 0
}

const tabStripY = headerH + tabsH/2

func twoTabs(t *testing.T) *App {
	a, _ := newTestApp(t)
	click(a, 100, treeRowY(4)) // opens "Repository Info" as the second tab
	if len(a.tabs) != 2 {
		t.Fatal("test setup: want two tabs")
	}
	return a
}

func TestCloseButtonClosesOnlyThatTab(t *testing.T) {
	a := twoTabs(t) // active: the second
	left, width := a.tabX(0)
	click(a, left+width-12, tabStripY) // the × of the first tab
	if len(a.tabs) != 1 || a.cur().Req.Name != "Repository Info" {
		t.Fatalf("%d tabs, current %q", len(a.tabs), a.cur().Req.Name)
	}
}

func TestClickingTheLabelSelectsAndDoesNotClose(t *testing.T) {
	a := twoTabs(t)
	left, _ := a.tabX(0)
	click(a, left+30, tabStripY)
	if len(a.tabs) != 2 || a.active != 0 {
		t.Fatalf("%d tabs, active %d", len(a.tabs), a.active)
	}
}

func TestMiddleButtonDragReordersTabsAndDoesNotCloseThem(t *testing.T) {
	a := twoTabs(t)
	first := a.tabs[0]
	l0, _ := a.tabX(0)
	l1, w1 := a.tabX(1)
	a.step(core.Input{X: l0 + 20, Y: tabStripY, MiddleClick: true, MiddleHeld: true}) // press the first tab
	if len(a.tabs) != 2 {
		t.Fatal("a middle press must not close the tab any more")
	}
	a.step(core.Input{X: l1 + w1/2, Y: tabStripY, MiddleHeld: true}) // drag over the second
	if a.tabs[1] != first {
		t.Fatal("the dragged tab must move to the second place")
	}
	a.step(core.Input{X: l1 + w1/2, Y: tabStripY}) // release
	if a.dragTab != -1 {
		t.Fatalf("dragTab = %d after the release", a.dragTab)
	}
	a.step(core.Input{X: l0 + 20, Y: tabStripY, MiddleHeld: true}) // held again, but not pressed on a tab
	if a.tabs[1] != first {
		t.Fatal("after the release, moving must not drag")
	}
}

func TestMiddleClickWithoutMovingLeavesTheTabsAlone(t *testing.T) {
	a := twoTabs(t)
	left, width := a.tabX(1)
	a.step(core.Input{X: left + width/2, Y: tabStripY, MiddleClick: true, MiddleHeld: true})
	a.step(core.Input{X: left + width/2, Y: tabStripY})
	if len(a.tabs) != 2 || a.cur().Req.Name != "Repository Info" {
		t.Fatalf("%d tabs, current %q", len(a.tabs), a.cur().Req.Name)
	}
}

func TestClosingATabBeforeTheActiveOneKeepsTheSameTabActive(t *testing.T) {
	a := twoTabs(t)
	a.closeTab(0)
	if a.active != 0 || a.cur().Req.Name != "Repository Info" {
		t.Fatalf("active %d current %q", a.active, a.cur().Req.Name)
	}
}

func TestDirtyFollowsEditsAndUndoingThemByHand(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	if dirty(tab) {
		t.Fatal("a freshly opened request is clean")
	}
	click(a, 1100, urlBarY)
	a.typeText("x")
	if !dirty(tab) {
		t.Fatal("editing the URL makes the tab dirty")
	}
	a.step(core.Input{Edits: []core.Edit{core.EditBackspace}})
	if dirty(tab) {
		t.Fatal("restoring the original text makes it clean again")
	}
	click(a, 739, paramsRow0Y) // delete a param
	if !dirty(tab) {
		t.Fatal("deleting a param is a change")
	}
}

func TestTabWidthDoesNotChangeWhenTheRequestChanges(t *testing.T) {
	a := twoTabs(t)
	left, width := a.tabX(0)
	req := a.tabs[0].Req
	req.Method = "DELETE"                                                 // a longer badge
	req.Headers = append(req.Headers, model.KV{Name: "X", Enabled: true}) // and an unsaved dot
	a.step(core.Input{})
	if !dirty(a.tabs[0]) {
		t.Fatal("test setup: the tab must be dirty")
	}

	// The × sits where it did before the edits; if the tab had grown it
	// would be somewhere else and the click would only select the tab.
	click(a, left+width-12, tabStripY)
	if len(a.tabs) != 1 || a.cur().Req.Name != "Repository Info" {
		t.Fatalf("%d tabs: the tab changed size when its request changed", len(a.tabs))
	}
}

func TestTheTabSlotsFitTheirContentAtEveryTextSize(t *testing.T) {
	for _, size := range appearance.TextSizeChoices {
		ctx, err := core.NewCtx(appearance.ThemeFor("dark", size.ID))
		if err != nil {
			t.Fatal(err)
		}
		for _, badge := range []string{"GET", "POST", "PUT", "PATCH", "DEL", "WS", "MCP"} {
			if got := ctx.Width(ctx.F.Small, badge); got > tabBadgeW {
				t.Errorf("%s: %q is %dpx wide, the badge slot is %dpx", size.ID, badge, got, tabBadgeW)
			}
		}
		if got := ctx.Width(ctx.F.UI, dirtyMark); got > tabDirtyW {
			t.Errorf("%s: the dirty mark is %dpx wide, the slot is %dpx", size.ID, got, tabDirtyW)
		}
	}
}

func TestDraggingATabReordersIt(t *testing.T) {
	a := twoTabs(t)
	first := a.tabs[0]
	l0, w0 := a.tabX(0)
	l1, w1 := a.tabX(1)

	a.step(core.Input{X: l0 + 20, Y: tabStripY, Click: true, Held: true}) // press the first tab
	a.step(core.Input{X: l1 + w1/2, Y: tabStripY, Held: true})            // drag over the second
	if a.tabs[1] != first {
		t.Fatalf("the dragged tab must move to the second place")
	}
	a.step(core.Input{X: l1 + w1/2, Y: tabStripY}) // release
	a.step(core.Input{X: l0 + w0/2, Y: tabStripY, Held: true})
	if a.tabs[1] != first {
		t.Fatal("after release, moving over another tab must not drag")
	}
	if a.dragTab != -1 {
		t.Fatalf("dragTab = %d after release", a.dragTab)
	}
}

func TestDraggingKeepsTheActiveTabActive(t *testing.T) {
	a := twoTabs(t) // second tab active
	active := a.cur()
	l0, _ := a.tabX(0)
	_, w1 := a.tabX(1)
	l1, _ := a.tabX(1)
	a.step(core.Input{X: l0 + 20, Y: tabStripY, Click: true, Held: true})
	a.step(core.Input{X: l1 + w1/2, Y: tabStripY, Held: true})
	if a.cur() == active {
		t.Fatal("pressing the first tab activates it, so it is the active one")
	}
	if a.cur() != a.tabs[1] {
		t.Fatal("the active tab must travel with the drag")
	}
}

// pathServer serves /fast at once and /slow only when released (or when
// the client gives up), so tests control which request finishes when.
func pathServer(t *testing.T) (srv *httptest.Server, release func()) {
	t.Helper()
	gate := make(chan struct{})
	var once sync.Once
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/slow":
			select {
			case <-gate:
				io.WriteString(w, "slow")
			case <-r.Context().Done():
			}
		case "/redirect":
			http.Redirect(w, r, "/fast", http.StatusFound)
		default:
			io.WriteString(w, "fast")
		}
	}))
	release = func() { once.Do(func() { close(gate) }) }
	t.Cleanup(func() { release(); srv.Close() })
	return srv, release
}

func TestSendingTurnsIntoCancelAndStopsTheRequest(t *testing.T) {
	srv, _ := pathServer(t)
	a, _ := newTestApp(t)
	tab := a.cur()
	pointTabAt(tab, srv.URL+"/slow")

	click(a, sendX, urlBarY)
	if !tab.Sending {
		t.Fatal("test setup: the request must be running")
	}
	click(a, sendX, urlBarY) // the same button now cancels
	if tab.Sending || tab.Resp == nil || tab.Resp.Err != "Cancelled" {
		t.Fatalf("sending %v resp %+v", tab.Sending, tab.Resp)
	}
	if a.status != "Cancelled" {
		t.Fatalf("status = %q", a.status)
	}
}

func TestEscapeCancelsARunningRequestButClosesTheMenuFirst(t *testing.T) {
	srv, _ := pathServer(t)
	a, _ := newTestApp(t)
	tab := a.cur()
	pointTabAt(tab, srv.URL+"/slow")
	click(a, sendX, urlBarY)

	a.menu = true
	a.step(core.Input{Escape: true})
	if a.menu || !tab.Sending {
		t.Fatalf("the first Escape closes the menu only: menu %v sending %v", a.menu, tab.Sending)
	}
	a.step(core.Input{Escape: true})
	if tab.Sending || tab.Resp == nil || tab.Resp.Err != "Cancelled" {
		t.Fatalf("the second Escape cancels: sending %v resp %+v", tab.Sending, tab.Resp)
	}
}

func TestEscapeWithNothingRunningDoesNothing(t *testing.T) {
	a, _ := newTestApp(t)
	a.step(core.Input{Escape: true})
	if a.cur().Resp != nil {
		t.Fatal("Escape must not invent a response")
	}
}

func TestALateResultFromACancelledRequestIsIgnored(t *testing.T) {
	srv, release := pathServer(t)
	a, _ := newTestApp(t)
	tab := a.cur()

	pointTabAt(tab, srv.URL+"/slow")
	click(a, sendX, urlBarY)
	click(a, sendX, urlBarY) // cancel
	pointTabAt(tab, srv.URL+"/fast")
	click(a, sendX, urlBarY)
	waitForResponse(t, a, tab)
	for tab.Resp.Err != "" || string(tab.Resp.Body) != "fast" { // the cancelled one may still be in flight
		a.step(core.Input{})
		time.Sleep(5 * time.Millisecond)
	}

	release()
	for range 40 {
		a.step(core.Input{})
		time.Sleep(5 * time.Millisecond)
	}
	if tab.Resp.Err != "" || string(tab.Resp.Body) != "fast" {
		t.Fatalf("the cancelled request's late result replaced the new one: %+v", tab.Resp)
	}
}

func TestRequestTimeoutStopsASlowRequest(t *testing.T) {
	srv, _ := pathServer(t)
	a, _ := newTestApp(t)
	tab := a.cur()
	pointTabAt(tab, srv.URL+"/slow")
	tab.Req.TimeoutSec = 1

	start := time.Now()
	click(a, sendX, urlBarY)
	waitForResponse(t, a, tab)
	if tab.Resp.Err != "Timed out" || time.Since(start) > 4*time.Second {
		t.Fatalf("resp %+v after %v", tab.Resp, time.Since(start))
	}
}

func TestTimeoutFallsBackToTheSettingsDefault(t *testing.T) {
	a, _ := newTestApp(t)
	tests := []struct {
		name         string
		req, setting int
		want         time.Duration
	}{
		{"the request's own value wins", 5, 60, 5 * time.Second},
		{"zero uses the default", 0, 30, 30 * time.Second},
		{"no default either uses a minute", 0, 0, 60 * time.Second},
		{"a negative value is not a timeout", -3, 30, 30 * time.Second},
	}
	for _, tt := range tests {
		a.settings.TimeoutSec = tt.setting
		if got := a.timeoutFor(model.Request{TimeoutSec: tt.req}); got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

// Settings pane (the sixth request tab): the timeout field at y=206 and
// the follow-redirects row below it.
const (
	settingsTabIdx = 6
	timeoutFieldX  = 480
	followRowY     = 242
	followRowX     = 290
)

func openSettingsPane(t *testing.T) *App {
	a, _ := newTestApp(t)
	a.cur().paneTab = settingsTabIdx
	a.step(core.Input{})
	return a
}

func TestTimeoutFieldAcceptsWholeSecondsOnly(t *testing.T) {
	a := openSettingsPane(t)
	req := a.cur().Req
	click(a, timeoutFieldX, chipsY)
	a.typeText("7")
	if req.TimeoutSec != 7 {
		t.Fatalf("TimeoutSec = %d", req.TimeoutSec)
	}
	a.typeText("x")
	if req.TimeoutSec != 7 {
		t.Fatalf("a non-numeric edit must not change the timeout, got %d", req.TimeoutSec)
	}
	a.step(core.Input{Edits: []core.Edit{core.EditBackspace, core.EditBackspace}}) // clears the field
	if req.TimeoutSec != 0 {
		t.Fatalf("an empty field means the default, got %d", req.TimeoutSec)
	}
}

func TestFollowRedirectsToggleControlsRedirects(t *testing.T) {
	srv, _ := pathServer(t)
	a := openSettingsPane(t)
	tab := a.cur()
	pointTabAt(tab, srv.URL+"/redirect")

	a.send(tab)
	waitForResponse(t, a, tab)
	if tab.Resp.Status != 200 {
		t.Fatalf("redirects are followed by default, status %d", tab.Resp.Status)
	}

	click(a, followRowX, followRowY)
	if !tab.Req.NoRedirects {
		t.Fatal("the checkbox row must turn following off")
	}
	tab.Resp = nil
	a.send(tab)
	waitForResponse(t, a, tab)
	if tab.Resp.Status != 302 {
		t.Fatalf("redirects are not followed now, status %d", tab.Resp.Status)
	}
}

// recordServer remembers the last request it served.
type recorded struct {
	header  http.Header
	query   string
	user    string
	pass    string
	hasAuth bool
}

func recordServer(t *testing.T) (*httptest.Server, func() recorded) {
	t.Helper()
	var mu sync.Mutex
	var last recorded
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		last.header, last.query = r.Header.Clone(), r.URL.RawQuery
		last.user, last.pass, last.hasAuth = r.BasicAuth()
	}))
	t.Cleanup(srv.Close)
	return srv, func() recorded { mu.Lock(); defer mu.Unlock(); return last }
}

// Auth pane (the fourth request tab): type chips at y=206 (None, Bearer,
// Basic, API key), field rows at y=242, 278, 314, then the show-secrets row.
const (
	authTabIdx  = 3
	authNoneX   = 304
	authBearerX = 378
	authBasicX  = 452
	authKeyX    = 530
	authFieldX  = 700
	authRow1Y   = 242
	authRow2Y   = 278
	authRow3Y   = 314
)

func openAuthPane(t *testing.T) *App {
	a, _ := newTestApp(t)
	a.cur().paneTab = authTabIdx
	a.step(core.Input{})
	return a
}

func sendAndWait(t *testing.T, a *App, url string) {
	t.Helper()
	tab := a.cur()
	pointTabAt(tab, url)
	tab.Resp = nil
	a.send(tab)
	waitForResponse(t, a, tab)
}

func TestBearerAuthSendsTheToken(t *testing.T) {
	srv, last := recordServer(t)
	a := openAuthPane(t)
	click(a, authBearerX, chipsY)
	click(a, authFieldX, authRow1Y)
	a.typeText("abc")
	req := a.cur().Req
	if req.Auth.Type != model.AuthBearer || req.Auth.Token != "abc" {
		t.Fatalf("auth = %+v", req.Auth)
	}
	if len(req.Headers) != 0 {
		t.Fatalf("credentials must not appear in the Headers table: %+v", req.Headers)
	}
	sendAndWait(t, a, srv.URL)
	if got := last().header.Get("Authorization"); got != "Bearer abc" {
		t.Fatalf("Authorization = %q", got)
	}
	if !dirty(a.cur()) {
		t.Fatal("typing credentials is a change")
	}
}

func TestBasicAuthSendsUserAndPassword(t *testing.T) {
	srv, last := recordServer(t)
	a := openAuthPane(t)
	click(a, authBasicX, chipsY)
	click(a, authFieldX, authRow1Y)
	a.typeText("ann")
	click(a, authFieldX, authRow2Y)
	a.typeText("pw")
	sendAndWait(t, a, srv.URL)
	if r := last(); !r.hasAuth || r.user != "ann" || r.pass != "pw" {
		t.Fatalf("basic auth = %+v", r)
	}
}

func TestAPIKeyGoesInTheHeaderOrTheQuery(t *testing.T) {
	srv, last := recordServer(t)
	a := openAuthPane(t)
	click(a, authKeyX, chipsY)
	click(a, authFieldX, authRow1Y)
	a.typeText("X-Key")
	click(a, authFieldX, authRow2Y)
	a.typeText("v1")

	sendAndWait(t, a, srv.URL)
	if got := last().header.Get("X-Key"); got != "v1" {
		t.Fatalf("header X-Key = %q", got)
	}

	click(a, 516, authRow3Y) // the Query choice
	if a.cur().Req.Auth.KeyIn != model.KeyInQuery {
		t.Fatalf("KeyIn = %q", a.cur().Req.Auth.KeyIn)
	}
	sendAndWait(t, a, srv.URL)
	if r := last(); r.query != "X-Key=v1" || r.header.Get("X-Key") != "" {
		t.Fatalf("query %q header %q", r.query, r.header.Get("X-Key"))
	}
}

func TestSecretFieldsAreMaskedUntilShown(t *testing.T) {
	a := openAuthPane(t)
	click(a, authBasicX, chipsY)
	user, pass := a.cur().authInput("user"), a.cur().authInput("password")
	if user.Masked || !pass.Masked {
		t.Fatalf("user masked %v, password masked %v", user.Masked, pass.Masked)
	}
	click(a, 290, authRow3Y) // the show-secrets row, under the two fields
	if a.cur().authInput("password").Masked {
		t.Fatal("the toggle must show the secret")
	}
	click(a, 290, authRow3Y)
	if !a.cur().authInput("password").Masked {
		t.Fatal("and hide it again")
	}
}

func TestVariablesInCredentialsAndHeadersAreExpandedAtSend(t *testing.T) {
	srv, last := recordServer(t)
	a := openAuthPane(t)
	a.col.Vars = append(a.col.Vars, model.KV{Name: "tok", Value: "sekret", Enabled: true})
	a.cur().Req.Headers = []model.KV{{Name: "X-Token", Value: "{{tok}}", Enabled: true}}
	click(a, authBearerX, chipsY)
	click(a, authFieldX, authRow1Y)
	a.typeText("{{tok}}")
	sendAndWait(t, a, srv.URL)
	r := last()
	if r.header.Get("Authorization") != "Bearer sekret" || r.header.Get("X-Token") != "sekret" {
		t.Fatalf("headers %v", r.header)
	}
	if a.cur().Req.Auth.Token != "{{tok}}" {
		t.Fatal("the stored request keeps the variable")
	}
}

func TestSwitchingAuthTypesKeepsEachTypesValues(t *testing.T) {
	a := openAuthPane(t)
	click(a, authBearerX, chipsY)
	click(a, authFieldX, authRow1Y)
	a.typeText("abc")
	click(a, authBasicX, chipsY)
	click(a, authNoneX, chipsY)
	click(a, authBearerX, chipsY)
	if a.cur().Req.Auth.Token != "abc" || a.cur().authInput("token").Text != "abc" {
		t.Fatalf("auth %+v field %q", a.cur().Req.Auth, a.cur().authInput("token").Text)
	}
}

// Response pane toolbar at 1280x800 (y=194): Pretty, Raw on the left;
// Find, Copy, cURL on the right. The find bar is the row below it.
const (
	toolbarY   = chipsY // the response toolbar shares a band with the request pane's first row
	compactX   = 818    // the response toolbar's chips: Compact, Pretty, Raw
	prettyX    = 888
	rawX       = 944
	findBtnX   = 1176
	copyBtnX   = 1214
	curlBtnX   = 1252
	findBarY   = chipsY + core.RowH // the find bar is the row below the toolbar
	findFieldX = 900
	findPrevX  = 1176
	findNextX  = 1214
	findCloseX = 1252
)

// jsonResponse points the current tab at a server returning body and sends.
func jsonResponse(t *testing.T, body string) *App {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) }))
	t.Cleanup(srv.Close)
	a, _ := newTestApp(t)
	sendAndWait(t, a, srv.URL)
	return a
}

func TestRawAndPrettyToggleTheResponseView(t *testing.T) {
	a := jsonResponse(t, `{"a":1,"b":[2]}`)
	tab := a.cur()
	if len(tab.view.Lines) != 6 {
		t.Fatalf("pretty view has %d lines", len(tab.view.Lines))
	}
	click(a, rawX, toolbarY)
	if !tab.raw || len(tab.view.Lines) != 1 || tab.view.AllText() != `{"a":1,"b":[2]}` {
		t.Fatalf("raw: %d lines %q", len(tab.view.Lines), tab.view.AllText())
	}
	click(a, prettyX, toolbarY)
	if tab.raw || len(tab.view.Lines) != 6 {
		t.Fatalf("back to pretty: raw %v, %d lines", tab.raw, len(tab.view.Lines))
	}
}

func TestBinaryBodiesAreNotShownAsText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte{0x89, 'P', 'N', 'G', 0, 1, 2, 0xff})
	}))
	defer srv.Close()
	a, _ := newTestApp(t)
	sendAndWait(t, a, srv.URL)
	msg, isErr := paneEmpty(responsePanes[0], a.cur())
	if !strings.Contains(msg, "Binary body") || !strings.Contains(msg, "8B") || isErr {
		t.Fatalf("message %q error %v", msg, isErr)
	}
	if msg, _ := paneEmpty(responsePanes[1], a.cur()); msg != "" {
		t.Fatalf("the Headers pane must still show headers, got %q", msg)
	}
}

func TestCopyBodyAndCopyAsCurl(t *testing.T) {
	a := jsonResponse(t, `{"ok":true}`)
	clip := &fakeClip{}
	a.ctx.Clipboard = clip

	click(a, copyBtnX, toolbarY)
	if clip.text != `{"ok":true}` {
		t.Fatalf("copy body: %q (the body as received, not the pretty one)", clip.text)
	}
	click(a, curlBtnX, toolbarY)
	if !strings.HasPrefix(clip.text, "curl -L 'http://127.0.0.1:") {
		t.Fatalf("copy as cURL: %q", clip.text)
	}
}

func TestCurlCopyExpandsVariables(t *testing.T) {
	a, _ := newTestApp(t) // the sample request uses {{baseUrl}}
	a.cur().Resp = &model.Response{Status: 200, Body: []byte("{}")}
	a.cur().view.SetLines(respview.CodeLines([]byte("{}")))
	clip := &fakeClip{}
	a.ctx.Clipboard = clip
	a.step(core.Input{})
	click(a, curlBtnX, toolbarY)
	if !strings.Contains(clip.text, "https://api.github.com/search/repositories?q=gioui") || strings.Contains(clip.text, "{{") {
		t.Fatalf("cURL = %q", clip.text)
	}
}

func TestCtrlFOpensFindAndTypingSearches(t *testing.T) {
	a := jsonResponse(t, `{"id":1,"name":"x","ids":[1,2]}`)
	tab := a.cur()
	a.step(key(ebiten.KeyF, true, false))
	if !tab.findOpen || !tab.find.Focused {
		t.Fatalf("findOpen %v focused %v", tab.findOpen, tab.find.Focused)
	}
	a.typeText("id")
	if tab.matchIdx != 0 || len(tab.view.Find("id")) != 2 {
		t.Fatalf("matchIdx %d, matches %d", tab.matchIdx, len(tab.view.Find("id")))
	}
	a.step(core.Input{Enter: true})
	if tab.matchIdx != 1 {
		t.Fatalf("Enter goes to the next match, idx %d", tab.matchIdx)
	}
	a.step(core.Input{Enter: true}) // wraps
	if tab.matchIdx != 0 {
		t.Fatalf("wrap: idx %d", tab.matchIdx)
	}
	a.step(core.Input{Enter: true, Shift: true})
	if tab.matchIdx != 1 {
		t.Fatalf("Shift+Enter goes back and wraps, idx %d", tab.matchIdx)
	}
}

func TestFindButtonsAndClose(t *testing.T) {
	a := jsonResponse(t, `{"id":1,"ids":[1]}`)
	tab := a.cur()
	click(a, findBtnX, toolbarY)
	if !tab.findOpen {
		t.Fatal("the Find button opens the bar")
	}
	click(a, findFieldX, findBarY)
	a.typeText("id")
	click(a, findNextX, findBarY)
	if tab.matchIdx != 1 {
		t.Fatalf("next: %d", tab.matchIdx)
	}
	click(a, findPrevX, findBarY)
	if tab.matchIdx != 0 {
		t.Fatalf("previous: %d", tab.matchIdx)
	}
	click(a, findCloseX, findBarY)
	if tab.findOpen {
		t.Fatal("× closes the bar")
	}
}

func TestEscapeClosesFindBeforeAnythingElse(t *testing.T) {
	a := jsonResponse(t, `{"id":1}`)
	tab := a.cur()
	a.step(key(ebiten.KeyF, true, false))
	a.typeText("id")
	a.step(core.Input{Escape: true})
	if tab.findOpen || len(tab.view.Find("")) != 0 {
		t.Fatal("Escape closes the find bar")
	}
	if tab.Resp == nil {
		t.Fatal("and must not touch the response")
	}
}

func TestANewResponseRerunsAnOpenSearch(t *testing.T) {
	a := jsonResponse(t, `{"id":1}`)
	tab := a.cur()
	a.step(key(ebiten.KeyF, true, false))
	a.typeText("id")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, `{"id":1,"idx":2,"idy":3}`) }))
	defer srv.Close()
	sendAndWait(t, a, srv.URL)
	if got := len(tab.view.Find("id")); got != 3 {
		t.Fatalf("matches after the new response: %d", got)
	}
	if tab.matchIdx != 0 {
		t.Fatalf("matchIdx = %d", tab.matchIdx)
	}
}

const dividerY = 400 // anywhere along the tall dividers

func drag(a *App, fromX, toX, y int) {
	a.step(core.Input{X: fromX, Y: y, Click: true, Held: true})
	a.step(core.Input{X: toX, Y: y, Held: true})
	a.step(core.Input{X: toX, Y: y}) // release
}

func TestDraggingTheSidebarDividerResizesTheSidebar(t *testing.T) {
	a, _ := newTestApp(t)
	drag(a, 259, 400, dividerY)
	if a.sidebarSplit.Size < 390 {
		t.Fatalf("sidebar size = %v", a.sidebarSplit.Size)
	}
	click(a, 350, treeRowY(4)) // x=350 was the main column before; it is the tree now
	if a.cur().Req.Name != "Repository Info" {
		t.Fatalf("a click at x=350 opened %q: the sidebar did not grow", a.cur().Req.Name)
	}
}

func TestSidebarKeepsAMinimumWidthAndDoubleClickResets(t *testing.T) {
	a, _ := newTestApp(t)
	drag(a, 259, 5, dividerY)
	click(a, 150, treeRowY(4)) // the minimum is 160px wide
	if a.cur().Req.Name != "Repository Info" {
		t.Fatal("the sidebar must not shrink below its minimum")
	}
	click(a, 200, treeRowY(5))
	if a.cur().Req.Name == "Repository Tags" {
		t.Fatal("x=200 is outside a minimum-width sidebar")
	}

	a.step(core.Input{X: 159, Y: dividerY, Click: true, Held: true, Clicks: 2})
	a.step(core.Input{})
	if a.sidebarSplit.Size != 0 {
		t.Fatalf("double click must reset the width, Size = %v", a.sidebarSplit.Size)
	}
	click(a, 250, treeRowY(5))
	if a.cur().Req.Name != "Repository Tags" {
		t.Fatal("back at the default width x=250 is in the tree again")
	}
}

func TestDraggingThePaneDividerChangesTheShare(t *testing.T) {
	a, _ := newTestApp(t)
	drag(a, 769, 600, dividerY)
	if s := a.paneSplit.Size; s <= 0.25 || s >= 0.4 {
		t.Fatalf("share = %v, want about 0.33 (the request pane is now narrower)", s)
	}
}

func TestResizeCursorOverTheDividers(t *testing.T) {
	a, _ := newTestApp(t)
	for _, tt := range []struct {
		name string
		x, y int
		want core.Cursor
	}{
		{"sidebar divider", 259, dividerY, core.CursorResizeH},
		{"pane divider", 769, dividerY, core.CursorResizeH},
		{"elsewhere", 500, dividerY, core.CursorDefault},
	} {
		a.step(core.Input{X: tt.x, Y: tt.y})
		if a.ctx.Cursor != tt.want {
			t.Errorf("%s: cursor %v, want %v", tt.name, a.ctx.Cursor, tt.want)
		}
	}
}

// Layout chooser at the bottom of the settings menu (below the text size row).
const ()

func TestLayoutSettingStacksThePanesAndIsSaved(t *testing.T) {
	a, path := newTestApp(t)
	click(a, menuBtnX, menuBtnY)
	a.prefs.Search = view.NewTextField("request and response")
	clickText(t, a, "Stacked", 0)
	if a.settings.Layout != "stacked" {
		t.Fatalf("Layout = %q", a.settings.Layout)
	}
	if saved, _ := settingsfile.Load(path); saved.Layout != "stacked" {
		t.Fatalf("saved %+v", saved)
	}
	click(a, 100, 700) // close the menu
	clip := &fakeClip{}
	a.ctx.Clipboard = clip
	tab := a.cur()
	srv := jsonServer(t)
	sendAndWait(t, a, srv.URL)

	// stacked: the response pane is the lower half, so its toolbar is at y=522
	click(a, copyBtnX, 510+12)
	if clip.text != `{"ok":true}` {
		t.Fatalf("the Copy button was not where a stacked layout puts it, clipboard %q", clip.text)
	}
	_ = tab
	click(a, menuBtnX, menuBtnY)
	a.prefs.Search = view.NewTextField("request and response") // the reopened dialog is fresh
	clickText(t, a, "Side by side", 0)
	if a.settings.Layout != "side" {
		t.Fatalf("Layout = %q", a.settings.Layout)
	}
}

// ---- context menus, dialogs and toasts --------------------------------

type ran struct{ log []string }

func (r *ran) item(label string) menuItem {
	return menuItem{label: label, run: func() { r.log = append(r.log, label) }}
}

// A menu opened at (500, 300): rows are 24px under 4px of padding, a
// separator is 9px: Copy 304..328, separator, Paste 337..361, Delete 361..385.
func openTestMenu(t *testing.T) (*App, *ran) {
	t.Helper()
	a, _ := newTestApp(t)
	r := &ran{}
	paste := r.item("Paste")
	paste.disabled = true
	a.openMenu(500, 300, []menuItem{r.item("Copy"), {separator: true}, paste, r.item("Delete")})
	a.step(core.Input{})
	if a.ctxMenu == nil {
		t.Fatal("test setup: the menu must be open")
	}
	return a, r
}

const (
	menuItemX   = 600
	menuCopyY   = 316
	menuPasteY  = 349
	menuDeleteY = 373
)

func TestClickingAMenuItemRunsItAndClosesTheMenu(t *testing.T) {
	a, r := openTestMenu(t)
	click(a, menuItemX, menuCopyY)
	if len(r.log) != 1 || r.log[0] != "Copy" || a.ctxMenu != nil {
		t.Fatalf("ran %v, menu open %v", r.log, a.ctxMenu != nil)
	}
}

func TestDisabledItemsDoNothingAndKeepTheMenuOpen(t *testing.T) {
	a, r := openTestMenu(t)
	click(a, menuItemX, menuPasteY)
	if len(r.log) != 0 || a.ctxMenu == nil {
		t.Fatalf("ran %v, menu open %v", r.log, a.ctxMenu != nil)
	}
}

func TestClickingOutsideClosesTheMenuWithoutReachingWhatIsBelow(t *testing.T) {
	a, r := openTestMenu(t)
	click(a, 100, treeRowY(4)) // a request row in the sidebar
	if a.ctxMenu != nil || len(r.log) != 0 {
		t.Fatalf("menu open %v ran %v", a.ctxMenu != nil, r.log)
	}
	if len(a.tabs) != 1 {
		t.Fatal("the click that closes a menu must not also open the request under it")
	}
}

func TestMenuKeyboardNavigationSkipsSeparatorsAndDisabledRows(t *testing.T) {
	a, r := openTestMenu(t)
	down := func() { a.step(key(ebiten.KeyArrowDown, false, false)) }
	up := func() { a.step(key(ebiten.KeyArrowUp, false, false)) }
	if a.ctxMenu.hot != 0 {
		t.Fatalf("the first enabled row starts highlighted, hot = %d", a.ctxMenu.hot)
	}
	down()
	if a.ctxMenu.hot != 3 {
		t.Fatalf("Down skips the separator and the disabled row: hot = %d", a.ctxMenu.hot)
	}
	down()
	if a.ctxMenu.hot != 0 {
		t.Fatalf("Down wraps: hot = %d", a.ctxMenu.hot)
	}
	up()
	if a.ctxMenu.hot != 3 {
		t.Fatalf("Up wraps backwards: hot = %d", a.ctxMenu.hot)
	}
	a.step(key(ebiten.KeyEnter, false, false))
	if len(r.log) != 1 || r.log[0] != "Delete" || a.ctxMenu != nil {
		t.Fatalf("Enter runs the highlighted row: ran %v", r.log)
	}
}

func TestEscapeClosesTheMenuFirst(t *testing.T) {
	a, r := openTestMenu(t)
	a.menu = true // the settings menu is open too
	a.step(core.Input{Escape: true})
	if a.ctxMenu != nil || !a.menu || len(r.log) != 0 {
		t.Fatalf("context menu open %v, settings menu open %v: Escape closes the top one only", a.ctxMenu != nil, a.menu)
	}
}

func TestMenuNearTheWindowEdgeStaysOnScreen(t *testing.T) {
	a, _ := newTestApp(t)
	r := &ran{}
	a.openMenu(1270, 790, []menuItem{r.item("Copy"), {separator: true}, r.item("Paste"), r.item("Delete")})
	a.step(core.Input{})
	// the menu is as wide as its content now: moved left so its right edge is
	// at the window's, and up to y=701 (89 high); the Copy row is 705..729
	click(a, 1280-a.menuWidth(a.ctxMenu)/2-4, 717)
	if len(r.log) != 1 || r.log[0] != "Copy" {
		t.Fatalf("ran %v: the menu must be flipped inside the window", r.log)
	}
}

func TestShortcutsAreIgnoredWhileAMenuIsOpen(t *testing.T) {
	a, _ := openTestMenu(t)
	a.step(key(ebiten.KeyT, true, false))
	if len(a.tabs) != 1 {
		t.Fatal("Ctrl+T must not open a tab behind a menu")
	}
}

// A confirm dialog is 380px wide, centered; with one line of body it is
// 128px high, so its buttons sit at y=434: OK at x=774, Cancel at x=686.
const (
	dialogOKX     = 774
	dialogCancelX = 686
	dialogButtonY = 434
)

func TestConfirmRunsOnEnterAndOnTheButton(t *testing.T) {
	for name, act := range map[string]func(a *App){
		"Enter":  func(a *App) { a.step(core.Input{Enter: true}) },
		"button": func(a *App) { click(a, dialogOKX, dialogButtonY) },
	} {
		a, _ := newTestApp(t)
		var ok int
		a.confirm("Delete request?", "This cannot be undone.", "Delete", func() { ok++ })
		a.step(core.Input{})
		if a.dialog == nil {
			t.Fatalf("%s: no dialog", name)
		}
		act(a)
		if ok != 1 || a.dialog != nil {
			t.Fatalf("%s: onOK ran %d times, dialog open %v", name, ok, a.dialog != nil)
		}
	}
}

func TestConfirmCancelsOnEscapeCancelButtonAndOutsideClick(t *testing.T) {
	for name, act := range map[string]func(a *App){
		"Escape":  func(a *App) { a.step(core.Input{Escape: true}) },
		"Cancel":  func(a *App) { click(a, dialogCancelX, dialogButtonY) },
		"outside": func(a *App) { click(a, 100, 700) },
	} {
		a, _ := newTestApp(t)
		var ok int
		a.confirm("Delete request?", "This cannot be undone.", "Delete", func() { ok++ })
		a.step(core.Input{})
		act(a)
		if ok != 0 || a.dialog != nil {
			t.Fatalf("%s: onOK ran %d times, dialog open %v", name, ok, a.dialog != nil)
		}
	}
}

func TestPromptReturnsTheTypedText(t *testing.T) {
	a, _ := newTestApp(t)
	var got string
	a.prompt("Rename", "Old name", "Rename", func(text string) { got = text })
	a.step(core.Input{})
	if a.dialog == nil || !a.dialog.input.Focused || a.dialog.input.SelectedText() != "Old name" {
		t.Fatalf("the field starts focused with its text selected: %+v", a.dialog)
	}
	a.typeText("New name")
	a.step(core.Input{Enter: true})
	if got != "New name" || a.dialog != nil {
		t.Fatalf("got %q, dialog open %v", got, a.dialog != nil)
	}
}

func TestDialogTrapsFocusAndBlocksShortcuts(t *testing.T) {
	a, _ := newTestApp(t)
	a.prompt("Rename", "x", "Rename", func(string) {})
	a.step(core.Input{})
	a.step(core.Input{Tab: true})
	if !a.dialog.input.Focused || a.cur().url.Focused {
		t.Fatal("Tab must stay inside the dialog")
	}
	a.step(key(ebiten.KeyT, true, false))
	if len(a.tabs) != 1 {
		t.Fatal("global shortcuts are off while a dialog is open")
	}
}

// Toasts stack up from above the status bar at the right: 320x36 each
// with 6px between them; the newest is lowest.
func toastRect(n, i int) (x, y, w, h int) {
	total := n*36 + (n-1)*6
	return 1280 - 12 - 320, 800 - statusH - 8 - total + i*42, 320, 36
}

func TestInfoToastsExpireAndErrorsStayUntilClicked(t *testing.T) {
	a, _ := newTestApp(t)
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	a.now = func() time.Time { return clock }
	a.toast("Saved", false)
	a.toast("Disk full", true)
	a.step(core.Input{})
	if len(a.toasts) != 2 {
		t.Fatalf("toasts = %d", len(a.toasts))
	}

	clock = clock.Add(2 * time.Second)
	a.step(core.Input{})
	if len(a.toasts) != 2 {
		t.Fatal("an info toast stays for a few seconds")
	}
	clock = clock.Add(2 * time.Second)
	a.step(core.Input{})
	if len(a.toasts) != 1 || !a.toasts[0].isErr {
		t.Fatalf("after 4s only the error is left: %+v", a.toasts)
	}
	clock = clock.Add(10 * time.Minute)
	a.step(core.Input{})
	if len(a.toasts) != 1 {
		t.Fatal("errors persist")
	}
	x, y, w, h := toastRect(1, 0)
	click(a, x+w/2, y+h/2)
	if len(a.toasts) != 0 {
		t.Fatal("a click dismisses a toast")
	}
}

func TestClickingOneToastDismissesOnlyThatOne(t *testing.T) {
	a, _ := newTestApp(t)
	a.toast("first", true)
	a.toast("second", true)
	a.step(core.Input{})
	x, y, w, h := toastRect(2, 0)
	click(a, x+w/2, y+h/2)
	if len(a.toasts) != 1 || a.toasts[0].text != "second" {
		t.Fatalf("toasts = %+v", a.toasts)
	}
}

func TestOnlyTheNewestToastsAreKept(t *testing.T) {
	a, _ := newTestApp(t)
	for i := 0; i < 8; i++ {
		a.toast(strings.Repeat("x", i+1), true)
	}
	if len(a.toasts) != maxToasts || len(a.toasts[len(a.toasts)-1].text) != 8 {
		t.Fatalf("%d toasts, newest %q", len(a.toasts), a.toasts[len(a.toasts)-1].text)
	}
}

func TestClipboardFailureBecomesAnErrorToast(t *testing.T) {
	a, _ := newTestApp(t)
	a.ctx.Clipboard = statusClipboard{inner: &fakeClip{err: errors.New("no clipboard tool found")}, report: a.clipboardNote}
	a.ctx.Clipboard.Write("x")
	if len(a.toasts) != 1 || !a.toasts[0].isErr || !strings.Contains(a.toasts[0].text, "no clipboard tool") {
		t.Fatalf("toasts = %+v", a.toasts)
	}
	a.ctx.Clipboard = statusClipboard{inner: &fakeClip{}, report: a.clipboardNote}
	a.ctx.Clipboard.Write("hello")
	if len(a.toasts) != 1 || a.status != "Copied 5 characters" {
		t.Fatalf("a successful copy is a status line, not a toast: toasts %d status %q", len(a.toasts), a.status)
	}
}

// ---- collection tree: create, rename, delete, move ----------------------

func rightClick(a *App, x, y int) { a.step(core.Input{X: x, Y: y, RightClick: true}) }

// Sample tree rows: 0 root, 1 User, 2 Repository, 3 Search Repos,
// 4 Repository Info, 5 Repository Tags, 6 Realtime.
const (
	rowRoot, rowUser, rowRepo, rowSearch, rowInfo, rowTags, rowRealtime = 0, 1, 2, 3, 4, 5, 6
)

func menuLabels(a *App) []string {
	var out []string
	for _, it := range a.ctxMenu.items {
		out = append(out, it.label)
	}
	return out
}

// pickMenu clicks the menu item with the given label.
func pickMenu(t *testing.T, a *App, label string) {
	t.Helper()
	m := a.ctxMenu
	if m == nil {
		t.Fatal("no menu is open")
	}
	y := m.y + menuPad
	for _, it := range m.items {
		h := menuRowH
		if it.separator {
			h = menuSepH
		}
		if it.label == label {
			click(a, m.x+100, y+h/2)
			return
		}
		y += h
	}
	t.Fatalf("no menu item %q in %v", label, menuLabels(a))
}

func findReq(a *App, name string) *model.Request {
	for _, r := range collection.Rows(&a.col.Folder, allOpen(a)) {
		if r.Req != nil && r.Req.Name == name {
			return r.Req
		}
	}
	return nil
}

func allOpen(a *App) map[*model.Folder]bool {
	open := map[*model.Folder]bool{}
	var walk func(*model.Folder)
	walk = func(f *model.Folder) {
		open[f] = true
		for _, s := range f.Folders {
			walk(s)
		}
	}
	walk(&a.col.Folder)
	return open
}

func TestRightClickOnATreeRowOpensItsMenu(t *testing.T) {
	a, _ := newTestApp(t)
	rightClick(a, 100, treeRowY(rowInfo))
	if a.ctxMenu == nil {
		t.Fatal("no menu")
	}
	if got, want := menuLabels(a), []string{"Rename", "Duplicate", "Copy URL", "", "Delete"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("request menu = %q, want %q", got, want)
	}
	a.step(core.Input{Escape: true})
	rightClick(a, 100, treeRowY(rowRepo))
	if got, want := menuLabels(a), []string{"New request", "New folder", "Import cURL from clipboard", "Import OpenAPI from clipboard", "Import Postman from clipboard", "Export as Postman...", "Variables", "", "Rename", "Duplicate", "Delete"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("folder menu = %q, want %q", got, want)
	}
	a.step(core.Input{Escape: true})
	rightClick(a, 100, treeRowY(rowRoot))
	if got, want := menuLabels(a), []string{"New request", "New folder", "Import cURL from clipboard", "Import OpenAPI from clipboard", "Import Postman from clipboard", "Export as Postman...", "Variables", "", "Rename"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("root menu = %q, want %q (the collection itself cannot be deleted or duplicated)", got, want)
	}
}

func TestRenameThroughTheMenuUpdatesTheTreeAndTheTab(t *testing.T) {
	a, _ := newTestApp(t)
	info := findReq(a, "Repository Info")
	rightClick(a, 100, treeRowY(rowInfo))
	pickMenu(t, a, "Rename")
	if a.dialog == nil || a.dialog.input.Text != "Repository Info" {
		t.Fatalf("dialog = %+v", a.dialog)
	}
	a.typeText("Repo details")
	a.step(core.Input{Enter: true})
	if info.Name != "Repo details" {
		t.Fatalf("name = %q", info.Name)
	}
	click(a, 100, treeRowY(rowInfo))
	if a.cur().Req != info || a.cur().Req.Name != "Repo details" {
		t.Fatal("the open tab must show the new name")
	}
}

func TestRenameToATakenNameIsRefusedWithAToast(t *testing.T) {
	a, _ := newTestApp(t)
	info := findReq(a, "Repository Info")
	rightClick(a, 100, treeRowY(rowInfo))
	pickMenu(t, a, "Rename")
	a.typeText("Repository Tags")
	a.step(core.Input{Enter: true})
	if info.Name != "Repository Info" || len(a.toasts) != 1 || !a.toasts[0].isErr {
		t.Fatalf("name %q toasts %+v", info.Name, a.toasts)
	}
}

func TestDuplicateAddsACopyRightAfter(t *testing.T) {
	a, _ := newTestApp(t)
	before := len(collection.Rows(&a.col.Folder, a.expanded))
	rightClick(a, 100, treeRowY(rowInfo))
	pickMenu(t, a, "Duplicate")
	rows := collection.Rows(&a.col.Folder, a.expanded)
	if len(rows) != before+1 || rows[rowInfo+1].Name != "Repository Info copy" {
		t.Fatalf("rows = %d, next row %q", len(rows), rows[rowInfo+1].Name)
	}
}

func TestDeleteAsksFirstAndClosesTheRequestsTab(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, 100, treeRowY(rowInfo)) // opens it in a tab
	info := a.cur().Req
	rightClick(a, 100, treeRowY(rowInfo))
	pickMenu(t, a, "Delete")
	if a.dialog == nil || findReq(a, "Repository Info") == nil {
		t.Fatal("deleting must ask first and change nothing yet")
	}
	a.step(core.Input{Escape: true})
	if findReq(a, "Repository Info") == nil {
		t.Fatal("cancelling keeps the request")
	}

	rightClick(a, 100, treeRowY(rowInfo))
	pickMenu(t, a, "Delete")
	a.step(core.Input{Enter: true})
	if findReq(a, "Repository Info") != nil {
		t.Fatal("the request must be gone")
	}
	for _, tab := range a.tabs {
		if tab.Req == info {
			t.Fatal("its tab must be closed")
		}
	}
}

func TestDeletingAFolderClosesTheTabsOfEverythingInIt(t *testing.T) {
	a, _ := newTestApp(t) // Search Repos (inside Repository) is open
	rightClick(a, 100, treeRowY(rowRepo))
	pickMenu(t, a, "Delete")
	a.step(core.Input{Enter: true})
	if len(a.col.Folders) != 2 {
		t.Fatalf("folders = %d", len(a.col.Folders))
	}
	if len(a.tabs) != 1 || a.cur().Req.Name != "New Request" {
		t.Fatalf("the open request was inside: %d tabs, current %q", len(a.tabs), a.cur().Req.Name)
	}
}

func TestCopyURLPutsTheExpandedURLOnTheClipboard(t *testing.T) {
	a, _ := newTestApp(t)
	clip := &fakeClip{}
	a.ctx.Clipboard = clip
	rightClick(a, 100, treeRowY(rowInfo))
	pickMenu(t, a, "Copy URL")
	if clip.text != "https://api.github.com/repos/golang/go" {
		t.Fatalf("clipboard = %q", clip.text)
	}
}

func TestNewRequestAndNewFolderFromTheFolderMenu(t *testing.T) {
	a, _ := newTestApp(t)
	repo := a.col.Folders[1]
	rightClick(a, 100, treeRowY(rowRepo))
	pickMenu(t, a, "New request")
	if len(repo.Requests) != 4 || repo.Requests[3].Name != "New Request" || a.cur().Req != repo.Requests[3] {
		t.Fatalf("requests %v, current %q", requestNames(repo), a.cur().Req.Name)
	}
	rightClick(a, 100, treeRowY(rowRepo))
	pickMenu(t, a, "New folder")
	if len(repo.Folders) != 1 || repo.Folders[0].Name != "New Folder" || !a.expanded[repo] {
		t.Fatalf("folders %v expanded %v", folderNames(repo), a.expanded[repo])
	}
}

func TestPlusCreatesTheRequestInTheSelectedFolder(t *testing.T) {
	a, _ := newTestApp(t)
	user := a.col.Folders[0]
	click(a, 100, treeRowY(rowUser)) // selects (and expands) the User folder
	a.step(key(ebiten.KeyT, true, false))
	if len(user.Requests) != 2 || a.cur().Req != user.Requests[1] {
		t.Fatalf("User now has %v", requestNames(user))
	}
}

func TestDraggingARequestOntoAFolderMovesIt(t *testing.T) {
	a, _ := newTestApp(t)
	tags := findReq(a, "Repository Tags")
	user := a.col.Folders[0]
	a.step(core.Input{X: 100, Y: treeRowY(rowTags), Click: true, Held: true})
	a.step(core.Input{X: 100, Y: treeRowY(rowUser), Held: true})
	a.step(core.Input{X: 100, Y: treeRowY(rowUser)}) // release over the User folder
	if got, ok := a.col.Folder.Parent(tags); !ok || got != user {
		t.Fatalf("Tags is in %v", got)
	}
	if a.cur().Req != tags {
		t.Fatal("pressing the row opened it, and it stays the open request")
	}
}

func TestDraggingARequestOntoARequestInsertsBeforeIt(t *testing.T) {
	a, _ := newTestApp(t)
	repo := a.col.Folders[1]
	a.step(core.Input{X: 100, Y: treeRowY(rowTags), Click: true, Held: true})
	a.step(core.Input{X: 100, Y: treeRowY(rowSearch), Held: true})
	a.step(core.Input{X: 100, Y: treeRowY(rowSearch)})
	if got := requestNames(repo); !reflect.DeepEqual(got, []string{"Repository Tags", "Search Repos", "Repository Info"}) {
		t.Fatalf("Repository = %v", got)
	}
}

func TestADropThatIsNotAllowedChangesNothingAndSaysWhy(t *testing.T) {
	a, _ := newTestApp(t)
	a.expanded[a.col.Folders[1]] = true
	// drag the Repository folder onto its own request: that is moving it into itself's parent, fine;
	// drag it onto a request inside itself is the refused case
	a.step(core.Input{X: 100, Y: treeRowY(rowRepo), Click: true, Held: true})
	a.step(core.Input{X: 100, Y: treeRowY(rowInfo), Held: true})
	a.step(core.Input{X: 100, Y: treeRowY(rowInfo)})
	if len(a.col.Folders) != 3 || len(a.col.Folders[1].Requests) != 3 {
		t.Fatal("a refused move changes nothing")
	}
	if len(a.toasts) != 1 || !a.toasts[0].isErr {
		t.Fatalf("toasts = %+v", a.toasts)
	}
}

func TestReleasingOnTheSameRowIsJustAClick(t *testing.T) {
	a, _ := newTestApp(t)
	a.step(core.Input{X: 100, Y: treeRowY(rowInfo), Click: true, Held: true})
	a.step(core.Input{X: 100, Y: treeRowY(rowInfo)})
	if got := requestNames(a.col.Folders[1]); !reflect.DeepEqual(got, []string{"Search Repos", "Repository Info", "Repository Tags"}) {
		t.Fatalf("Repository = %v", got)
	}
}

func TestFilterShowsMatchesByNameAndURL(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, 180, headerH/2) // the filter field in the sidebar header
	a.typeText("tags")
	rows := collection.Rows(&a.col.Folder, a.expanded)
	_ = rows
	got := a.visibleRows()
	var names []string
	for _, r := range got {
		names = append(names, r.Name)
	}
	if !reflect.DeepEqual(names, []string{"GitHub REST API", "Repository", "Repository Tags"}) {
		t.Fatalf("filter %q shows %q", "tags", names)
	}

	a.filter.SelectAll()
	a.typeText("search/repos") // part of a URL, not of any name
	names = nil
	for _, r := range a.visibleRows() {
		names = append(names, r.Name)
	}
	if !reflect.DeepEqual(names, []string{"GitHub REST API", "Repository", "Search Repos"}) {
		t.Fatalf("by URL: %q", names)
	}

	a.filter.SelectAll()
	a.typeText("zzz")
	if got := a.visibleRows(); len(got) != 1 {
		t.Fatalf("no match leaves just the root, got %d rows", len(got))
	}
	a.filter.SelectAll()
	a.step(core.Input{Edits: []core.Edit{core.EditBackspace}})
	if got := a.visibleRows(); len(got) != len(collection.Rows(&a.col.Folder, a.expanded)) {
		t.Fatal("an empty filter shows the tree as usual")
	}
}

func TestFilteredRowsStillOpenTheRightRequest(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, 180, headerH/2)
	a.typeText("tags")
	click(a, 100, treeRowY(2)) // third visible row: Repository Tags
	if a.cur().Req.Name != "Repository Tags" {
		t.Fatalf("opened %q", a.cur().Req.Name)
	}
}

func TestTreeKeyboardNavigation(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, 100, treeRowY(rowInfo)) // focuses the tree and selects Repository Info
	a.step(key(ebiten.KeyArrowDown, false, false))
	if a.selected != any(findReq(a, "Repository Tags")) {
		t.Fatalf("Down selects the next row, selected %v", a.selected)
	}
	a.step(key(ebiten.KeyArrowUp, false, false))
	a.step(key(ebiten.KeyArrowUp, false, false))
	if a.selected != any(findReq(a, "Search Repos")) {
		t.Fatal("Up moves back")
	}
	a.step(key(ebiten.KeyArrowDown, false, false))
	a.step(key(ebiten.KeyEnter, false, false))
	if a.cur().Req.Name != "Repository Info" {
		t.Fatalf("Enter opens the selected request, got %q", a.cur().Req.Name)
	}

	a.step(key(ebiten.KeyF2, false, false))
	if a.dialog == nil || a.dialog.input.Text != "Repository Info" {
		t.Fatal("F2 renames the selection")
	}
	a.step(core.Input{Escape: true})
	a.step(key(ebiten.KeyDelete, false, false))
	if a.dialog == nil {
		t.Fatal("Delete asks before deleting the selection")
	}
}

func TestTreeKeysDoNothingWhileTyping(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, 100, treeRowY(rowInfo))
	click(a, 1100, urlBarY) // now the URL field has the keyboard
	a.step(key(ebiten.KeyDelete, false, false))
	a.step(key(ebiten.KeyF2, false, false))
	if a.dialog != nil {
		t.Fatal("Delete and F2 belong to the text field, not the tree")
	}
}

// ---- table rows: keyboard operations, enable all, undo, bulk edit -------

// Params table at 1280x800: header cells at y=244 (the action cell at
// x=310), rows at y=272, 300, 328 and the placeholder at 356.
const (
	paramsHeaderY = 236
	checkAllX     = 310
	row1Y         = 292
	row2Y         = 320
	bulkBtnX      = 715
)

func altKey(k ebiten.Key) core.Input { return core.Input{Keys: []ebiten.Key{k}, Alt: true} }

func paramNames(a *App) []string {
	var out []string
	for _, p := range a.cur().Req.Params {
		out = append(out, p.Name)
	}
	return out
}

func TestCtrlEnterWhileEditingInsertsARowBelowAndEditsIt(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, valueCellX, paramsRow0Y)
	a.step(key(ebiten.KeyEnter, true, false))
	req := a.cur().Req
	if len(req.Params) != 4 || req.Params[1].Name != "" || !req.Params[1].Enabled || req.Params[0].Name != "q" {
		t.Fatalf("params = %+v", req.Params)
	}
	if a.edit == nil || a.edit.row != 1 || a.edit.col != 1 {
		t.Fatalf("edit = %+v: the new row is edited in the same column", a.edit)
	}
	if strings.Contains(req.URL, "&=") || strings.Contains(req.URL, "?=") {
		t.Fatalf("an empty name must stay out of the URL: %q", req.URL)
	}
}

func TestCtrlDDuplicatesTheEditedRow(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, valueCellX, paramsRow0Y)
	a.step(key(ebiten.KeyD, true, false))
	p := a.cur().Req.Params
	if len(p) != 4 || p[1] != p[0] || p[1].Name != "q" {
		t.Fatalf("params = %+v", p)
	}
	if a.edit == nil || a.edit.row != 1 {
		t.Fatalf("the copy is the edited row, edit = %+v", a.edit)
	}
}

func TestAltArrowsMoveTheRowAndTheEditFollows(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, valueCellX, paramsRow0Y)
	a.typeText("rust") // an uncommitted edit travels with the row
	a.step(altKey(ebiten.KeyArrowDown))
	if got := paramNames(a); !reflect.DeepEqual(got, []string{"order", "q", "sort"}) {
		t.Fatalf("params = %v", got)
	}
	if a.edit == nil || a.edit.row != 1 || a.cur().Req.Params[1].Value != "rust" {
		t.Fatalf("edit %+v value %q", a.edit, a.cur().Req.Params[1].Value)
	}
	if !strings.Contains(a.cur().Req.URL, "order=desc&q=rust&sort=stars") {
		t.Fatalf("the URL follows the order: %q", a.cur().Req.URL)
	}
	a.step(altKey(ebiten.KeyArrowUp))
	a.step(altKey(ebiten.KeyArrowUp)) // already at the top: stays
	if got := paramNames(a); !reflect.DeepEqual(got, []string{"q", "order", "sort"}) {
		t.Fatalf("params = %v", got)
	}
}

// The drag handle sits in the first 20 px of the table, which starts at x=272.
const gripCellX = 282

func TestDraggingARowByItsGripMovesItAndAFurtherDragKeepsMovingIt(t *testing.T) {
	a, _ := newTestApp(t)
	rowY := func(row int) int { return paramsRow0Y + row*core.RowH }
	a.step(core.Input{X: gripCellX, Y: rowY(0), Click: true, Held: true}) // press the first row's grip
	a.step(core.Input{X: gripCellX, Y: rowY(1), Held: true})
	if got := paramNames(a); !reflect.DeepEqual(got, []string{"order", "q", "sort"}) {
		t.Fatalf("after one row down: %v", got)
	}
	a.step(core.Input{X: gripCellX, Y: rowY(2), Held: true})
	if got := paramNames(a); !reflect.DeepEqual(got, []string{"order", "sort", "q"}) {
		t.Fatalf("after two rows down: %v", got)
	}
	if !strings.Contains(a.cur().Req.URL, "order=desc&sort=stars&q=gioui") {
		t.Fatalf("the URL follows the order: %q", a.cur().Req.URL)
	}
	a.step(core.Input{X: gripCellX, Y: rowY(0)}) // release elsewhere
	a.step(core.Input{X: gripCellX, Y: rowY(0), Held: true})
	if got := paramNames(a); !reflect.DeepEqual(got, []string{"order", "sort", "q"}) {
		t.Fatalf("a held button that did not start on a grip must not move rows: %v", got)
	}
}

func TestDeleteRemovesTheSelectedRowWhenNothingIsBeingTyped(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, valueCellX, paramsRow0Y)
	a.step(core.Input{Enter: true}) // commit; the row stays selected
	a.step(key(ebiten.KeyDelete, false, false))
	if got := paramNames(a); !reflect.DeepEqual(got, []string{"order", "sort"}) || strings.Contains(a.cur().Req.URL, "q=") {
		t.Fatalf("params %v url %q", got, a.cur().Req.URL)
	}
	a.step(key(ebiten.KeyDelete, false, false)) // the selection moved on: a second Delete removes the next row
	if len(a.cur().Req.Params) > 2 {
		t.Fatal("unreachable")
	}
}

func TestDeleteDoesNothingWithoutASelectedRowOrWhileTyping(t *testing.T) {
	a, _ := newTestApp(t)
	a.step(key(ebiten.KeyDelete, false, false))
	if len(a.cur().Req.Params) != 3 {
		t.Fatal("no row is selected")
	}
	click(a, valueCellX, paramsRow0Y) // editing: Delete is a text key
	a.step(key(ebiten.KeyDelete, false, false))
	if len(a.cur().Req.Params) != 3 {
		t.Fatal("while a cell is edited Delete belongs to the text")
	}
}

func TestHeaderCheckboxEnablesOrDisablesEveryRow(t *testing.T) {
	a, _ := newTestApp(t)
	req := a.cur().Req
	click(a, checkAllX, paramsHeaderY)
	for _, p := range req.Params {
		if p.Enabled {
			t.Fatalf("all must be disabled: %+v", req.Params)
		}
	}
	if strings.Contains(req.URL, "?") {
		t.Fatalf("no enabled params, no query: %q", req.URL)
	}
	click(a, checkAllX, paramsHeaderY)
	if !model.AllEnabled(req.Params) || !strings.Contains(req.URL, "q=gioui") {
		t.Fatalf("all enabled again: %+v url %q", req.Params, req.URL)
	}
}

func TestUndoAndRedoOfTableChanges(t *testing.T) {
	a, _ := newTestApp(t)
	req := a.cur().Req
	click(a, 739, paramsRow0Y) // delete the first row
	if len(req.Params) != 2 {
		t.Fatal("test setup")
	}
	a.step(core.Input{Ctrl: true, Undo: true})
	if len(req.Params) != 3 || req.Params[0].Name != "q" || !strings.Contains(req.URL, "q=gioui") {
		t.Fatalf("undo: %+v url %q", req.Params, req.URL)
	}
	a.step(core.Input{Ctrl: true, Redo: true})
	if len(req.Params) != 2 || strings.Contains(req.URL, "q=") {
		t.Fatalf("redo: %+v url %q", req.Params, req.URL)
	}
	a.step(core.Input{Ctrl: true, Undo: true})
	click(a, 310, row1Y) // a new change after an undo clears redo
	a.step(core.Input{Ctrl: true, Redo: true})
	if req.Params[1].Enabled {
		t.Fatalf("redo must be gone after a new change: %+v", req.Params)
	}
	a.step(core.Input{Ctrl: true, Undo: true})
	a.step(core.Input{Ctrl: true, Undo: true})
	a.step(core.Input{Ctrl: true, Undo: true}) // more undos than changes: stops at the start
	if len(req.Params) != 3 {
		t.Fatalf("params = %+v", req.Params)
	}
}

func TestUndoGoesToTheCellTextWhileACellIsEdited(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, 739, paramsRow0Y) // a table change to undo
	click(a, valueCellX, paramsRow0Y)
	a.typeText("zz")
	a.step(core.Input{Ctrl: true, Undo: true})
	if len(a.cur().Req.Params) != 2 {
		t.Fatal("Ctrl+Z inside a cell undoes the typing, not the table")
	}
}

func TestEveryTableChangeGoesThroughOnePlace(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, valueCellX, paramsRow0Y+2*core.RowH) // edit row 2
	if a.edit == nil || a.edit.row != 2 {
		t.Fatal("test setup")
	}
	a.applyRows(requestPanes[0], a.cur(), func(rows []model.KV) []model.KV { return model.DeleteRow(model.DeleteRow(rows, 2), 1) })
	if a.edit != nil {
		t.Fatalf("an edit pointing at a row that is gone must be dropped: %+v", a.edit)
	}
	if got := paramNames(a); !reflect.DeepEqual(got, []string{"q"}) || strings.Contains(a.cur().Req.URL, "order") {
		t.Fatalf("params %v url %q", got, a.cur().Req.URL)
	}
}

func TestBulkEditShowsLinesAndAppliesThemBack(t *testing.T) {
	a, _ := newTestApp(t)
	req := a.cur().Req
	req.Params[1].Enabled = false
	click(a, bulkBtnX, chipsY)
	if a.bulk == nil || a.bulk.area.Text != "q: gioui\n# order: desc\nsort: stars" {
		t.Fatalf("bulk = %+v", a.bulk)
	}
	a.bulk.area.SelectAll()
	a.bulk.area.Insert("q: rust\npage: 2\n# off: 1")
	click(a, bulkBtnX, chipsY) // back to the table
	if a.bulk != nil {
		t.Fatal("the button leaves bulk mode")
	}
	want := []model.KV{
		{Name: "q", Value: "rust", Description: "repo name", Enabled: true},
		{Name: "page", Value: "2", Enabled: true},
		{Name: "off", Value: "1", Enabled: false},
	}
	if !reflect.DeepEqual(req.Params, want) || !strings.HasSuffix(req.URL, "?q=rust&page=2") {
		t.Fatalf("params %+v url %q", req.Params, req.URL)
	}
	a.step(core.Input{Ctrl: true, Undo: true})
	if len(req.Params) != 3 || req.Params[0].Value != "gioui" {
		t.Fatalf("a bulk edit is one undo step: %+v", req.Params)
	}
}

func TestLeavingTheTabWhileBulkEditingAppliesTheText(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, bulkBtnX, chipsY)
	a.bulk.area.SelectAll()
	a.bulk.area.Insert("only: one")
	a.cur().paneTab = headersTabIdx // as clicking the Headers tab does
	a.step(core.Input{})
	if a.bulk != nil || !reflect.DeepEqual(paramNames(a), []string{"only"}) {
		t.Fatalf("bulk %v params %v: nothing typed may be lost", a.bulk != nil, paramNames(a))
	}
}

func TestDuplicateHeaderNamesAreFlagged(t *testing.T) {
	rows := []model.KV{{Name: "X-Id", Enabled: true}, {Name: "x-id", Enabled: true}, {Name: "Other", Enabled: true}}
	flagged := kvRows(rows, true, true)
	if !flagged[0].Flagged || !flagged[1].Flagged || flagged[2].Flagged {
		t.Fatalf("flags = %v %v %v", flagged[0].Flagged, flagged[1].Flagged, flagged[2].Flagged)
	}
	if kvRows(rows, true, false)[0].Flagged {
		t.Fatal("params may repeat a key: no flag there")
	}
}

// ---- right-click menus: tabs, table rows, text, response ----------------

func itemByLabel(a *App, label string) *menuItem {
	for i := range a.ctxMenu.items {
		if a.ctxMenu.items[i].label == label {
			return &a.ctxMenu.items[i]
		}
	}
	return nil
}

func TestTabMenu(t *testing.T) {
	a := twoTabs(t)
	left, _ := a.tabX(0)
	rightClick(a, left+20, tabStripY)
	want := []string{"Close", "Close others", "Close tabs to the right", "", "Duplicate", "Copy URL", "Cookies..."}
	if got := menuLabels(a); !reflect.DeepEqual(got, want) {
		t.Fatalf("tab menu = %q", got)
	}
	pickMenu(t, a, "Close tabs to the right")
	if len(a.tabs) != 1 || a.cur().Req.Name != "Search Repos" {
		t.Fatalf("%d tabs, current %q", len(a.tabs), a.cur().Req.Name)
	}
	rightClick(a, left+20, tabStripY)
	if !itemByLabel(a, "Close others").disabled || !itemByLabel(a, "Close tabs to the right").disabled {
		t.Fatal("with one tab there is nothing else to close")
	}
}

func TestCloseOthersKeepsOnlyThatTab(t *testing.T) {
	a := twoTabs(t)
	a.newRequest() // three tabs
	left, _ := a.tabX(1)
	keep := a.tabs[1]
	rightClick(a, left+20, tabStripY)
	pickMenu(t, a, "Close others")
	if len(a.tabs) != 1 || a.tabs[0] != keep || a.cur() != keep {
		t.Fatalf("%d tabs, current %q", len(a.tabs), a.cur().Req.Name)
	}
}

func TestDuplicateTabAddsACopyToTheTreeAndOpensIt(t *testing.T) {
	a, _ := newTestApp(t)
	left, _ := a.tabX(0)
	rightClick(a, left+20, tabStripY)
	pickMenu(t, a, "Duplicate")
	if got := requestNames(a.col.Folders[1]); !reflect.DeepEqual(got, []string{"Search Repos", "Search Repos copy", "Repository Info", "Repository Tags"}) {
		t.Fatalf("Repository = %v", got)
	}
	if a.cur().Req.Name != "Search Repos copy" || len(a.tabs) != 2 {
		t.Fatalf("current %q, %d tabs", a.cur().Req.Name, len(a.tabs))
	}
}

func TestTabMenuCopyURL(t *testing.T) {
	a, _ := newTestApp(t)
	clip := &fakeClip{}
	a.ctx.Clipboard = clip
	left, _ := a.tabX(0)
	rightClick(a, left+20, tabStripY)
	pickMenu(t, a, "Copy URL")
	if !strings.HasPrefix(clip.text, "https://api.github.com/search/repositories?q=gioui") {
		t.Fatalf("clipboard = %q", clip.text)
	}
}

func TestTableRowMenu(t *testing.T) {
	a, _ := newTestApp(t)
	rightClick(a, 400, paramsRow0Y)
	want := []string{"Copy value", "Copy name", "Copy as Name: Value", "", "Insert row above", "Insert row below", "Duplicate", "Move up", "Move down", "", "Enable all", "Disable all", "", "Delete"}
	if got := menuLabels(a); !reflect.DeepEqual(got, want) {
		t.Fatalf("row menu = %q", got)
	}
	if !itemByLabel(a, "Move up").disabled || itemByLabel(a, "Move down").disabled {
		t.Fatal("the first row cannot move up, but can move down")
	}
	pickMenu(t, a, "Insert row above")
	req := a.cur().Req
	if len(req.Params) != 4 || req.Params[0].Name != "" || req.Params[1].Name != "q" {
		t.Fatalf("params = %+v", req.Params)
	}
}

func TestTableRowMenuActionsUseTheSameUndoableChanges(t *testing.T) {
	a, _ := newTestApp(t)
	req := a.cur().Req
	rightClick(a, 400, row1Y) // the "order" row
	pickMenu(t, a, "Duplicate")
	if got := paramNames(a); !reflect.DeepEqual(got, []string{"q", "order", "order", "sort"}) {
		t.Fatalf("duplicate: %v", got)
	}
	rightClick(a, 400, row1Y)
	pickMenu(t, a, "Move down")
	rightClick(a, 400, paramsRow0Y)
	pickMenu(t, a, "Disable all")
	if model.AllEnabled(req.Params) || strings.Contains(req.URL, "?") {
		t.Fatalf("disable all: %+v url %q", req.Params, req.URL)
	}
	rightClick(a, 400, paramsRow0Y)
	pickMenu(t, a, "Delete")
	if len(req.Params) != 3 {
		t.Fatalf("delete: %v", paramNames(a))
	}
	for range 4 {
		a.step(core.Input{Ctrl: true, Undo: true})
	}
	if got := paramNames(a); !reflect.DeepEqual(got, []string{"q", "order", "sort"}) || !strings.Contains(req.URL, "q=gioui") {
		t.Fatalf("undoing every menu action restores the table: %v url %q", got, req.URL)
	}
}

func TestPlaceholderRowHasNoMenu(t *testing.T) {
	a, _ := newTestApp(t)
	rightClick(a, 400, paramsRow3Y)
	if a.ctxMenu != nil {
		t.Fatal("the empty placeholder row has no row menu")
	}
}

func TestTextFieldMenuStatesAndActions(t *testing.T) {
	a, _ := newTestApp(t)
	clip := &fakeClip{text: "PASTED"}
	a.ctx.Clipboard = clip
	rightClick(a, 1100, urlBarY)
	want := []string{"Undo", "Redo", "", "Cut", "Copy", "Paste", "", "Select all"}
	if got := menuLabels(a); !reflect.DeepEqual(got, want) {
		t.Fatalf("text menu = %q", got)
	}
	for _, l := range []string{"Undo", "Redo", "Cut", "Copy"} {
		if !itemByLabel(a, l).disabled {
			t.Errorf("%s must be disabled with nothing to undo or selected", l)
		}
	}
	if itemByLabel(a, "Paste").disabled || itemByLabel(a, "Select all").disabled {
		t.Fatal("Paste and Select all are available")
	}
	if itemByLabel(a, "Copy").shortcut != "Ctrl+C" {
		t.Fatalf("shortcut text = %q", itemByLabel(a, "Copy").shortcut)
	}

	pickMenu(t, a, "Select all")
	url := &a.cur().url
	if url.SelectedText() != url.Text {
		t.Fatal("Select all")
	}
	rightClick(a, 1100, urlBarY)
	if url.SelectedText() != url.Text {
		t.Fatal("a right click must not clear the selection")
	}
	if itemByLabel(a, "Copy").disabled || itemByLabel(a, "Cut").disabled {
		t.Fatal("Cut and Copy are available with a selection")
	}
	original := url.Text
	pickMenu(t, a, "Copy")
	if clip.text != original {
		t.Fatalf("clipboard = %q", clip.text)
	}
	rightClick(a, 1100, urlBarY)
	pickMenu(t, a, "Cut")
	if url.Text != "" || a.cur().Req.URL != "" {
		t.Fatalf("cut: field %q request %q (the model must follow)", url.Text, a.cur().Req.URL)
	}
	clip.text = "https://x.test"
	rightClick(a, 1100, urlBarY)
	pickMenu(t, a, "Paste")
	if url.Text != "https://x.test" || a.cur().Req.URL != "https://x.test" {
		t.Fatalf("paste: field %q request %q", url.Text, a.cur().Req.URL)
	}
	rightClick(a, 1100, urlBarY)
	pickMenu(t, a, "Undo")
	if url.Text != "" {
		t.Fatalf("undo: %q", url.Text)
	}
}

func TestSecretFieldsCannotBeCutOrCopiedFromTheMenu(t *testing.T) {
	a := openAuthPane(t)
	click(a, authBearerX, chipsY)
	click(a, authFieldX, authRow1Y)
	a.typeText("s3cret")
	a.cur().authInput("token").SelectAll()
	rightClick(a, authFieldX, authRow1Y)
	if !itemByLabel(a, "Cut").disabled || !itemByLabel(a, "Copy").disabled {
		t.Fatal("a masked field must not offer Cut or Copy")
	}
	if itemByLabel(a, "Paste").disabled {
		t.Fatal("pasting a secret in is fine")
	}
}

func TestCellEditFieldGetsTheTextMenuNotTheRowMenu(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, valueCellX, paramsRow0Y) // editing
	rightClick(a, valueCellX, paramsRow0Y)
	if a.ctxMenu == nil || menuLabels(a)[0] != "Undo" {
		t.Fatalf("menu = %v", a.ctxMenu)
	}
}

func TestBodyEditorMenuPastesLinesAndKeepsTheModelInSync(t *testing.T) {
	a := openBodyPane(t)
	clip := &fakeClip{text: "{\n  \"a\": 1\n}"}
	a.ctx.Clipboard = clip
	click(a, jsonChipX, chipsY)
	rightClick(a, editorX, editorY)
	pickMenu(t, a, "Paste")
	if a.cur().Req.Body != "{\n  \"a\": 1\n}" || a.cur().body.LineCount() != 3 {
		t.Fatalf("Body = %q", a.cur().Req.Body)
	}
}

func TestResponseMenu(t *testing.T) {
	a := jsonResponse(t, `{"ok":true}`)
	clip := &fakeClip{}
	a.ctx.Clipboard = clip
	rightClick(a, 1000, 400)
	want := []string{"Copy", "Copy body", "Save body...", "Open in browser", "Copy as cURL", "", "Select all", "Find"}
	if got := menuLabels(a); !reflect.DeepEqual(got, want) {
		t.Fatalf("response menu = %q", got)
	}
	if !itemByLabel(a, "Copy").disabled {
		t.Fatal("Copy needs a selection")
	}
	pickMenu(t, a, "Copy body")
	if clip.text != `{"ok":true}` {
		t.Fatalf("clipboard = %q", clip.text)
	}
	rightClick(a, 1000, 400)
	pickMenu(t, a, "Select all")
	if a.cur().view.SelectedText() != a.cur().view.AllText() {
		t.Fatal("Select all")
	}
	rightClick(a, 1000, 400)
	if itemByLabel(a, "Copy").disabled {
		t.Fatal("Copy works with a selection")
	}
	pickMenu(t, a, "Copy as cURL")
	if !strings.HasPrefix(clip.text, "curl -L ") {
		t.Fatalf("clipboard = %q", clip.text)
	}
	rightClick(a, 1000, 400)
	pickMenu(t, a, "Find")
	if !a.cur().findOpen {
		t.Fatal("Find opens the find bar")
	}
}

// ---- resizable table columns ------------------------------------------------

// The Params table: columns name|value|description at weights 28/40/32;
// the name|value divider is at x=438, the header row at y=234.
const (
	nameValueDividerX = 438
	headersPaneTabIdx = 2
)

func TestDraggingAParamsDividerResizesItsColumns(t *testing.T) {
	a, _ := newTestApp(t)
	drag(a, nameValueDividerX, 480, paramsHeaderY)
	w := a.widthsFor(requestPanes[0])
	if w == nil || len(w.Weights) != 3 || w.Weights[0] <= 28 || w.Weights[1] >= 40 {
		t.Fatalf("weights = %+v", w)
	}
	if sum := w.Weights[0] + w.Weights[1] + w.Weights[2]; sum < 99.9 || sum > 100.1 {
		t.Fatalf("the total must stay 100, got %v", sum)
	}
	click(a, 450, paramsRow0Y) // was in the value column; now it is the wider name column
	if a.edit == nil || a.edit.col != 0 {
		t.Fatalf("a click at x=450 edits %+v: the name column must have grown over it", a.edit)
	}
}

func TestEachTableKeepsItsOwnColumnWidths(t *testing.T) {
	a, _ := newTestApp(t)
	drag(a, nameValueDividerX, 480, paramsHeaderY)
	if a.widthsFor(requestPanes[0]).Weights == nil {
		t.Fatal("test setup")
	}
	if a.widthsFor(requestPanes[headersPaneTabIdx]).Weights != nil {
		t.Fatal("the Headers table must keep its default widths")
	}
	if a.widthsFor(responsePanes[1]) == a.widthsFor(requestPanes[headersPaneTabIdx]) {
		t.Fatal("the response Headers table is not the request Headers table")
	}
	if a.widthsFor(requestPanes[0]) != a.widthsFor(requestPanes[0]) {
		t.Fatal("asking twice must give the same state")
	}
}

func TestDoubleClickOnAParamsDividerResetsIt(t *testing.T) {
	a, _ := newTestApp(t)
	drag(a, nameValueDividerX, 480, paramsHeaderY)
	a.step(core.Input{X: 480, Y: paramsHeaderY, Click: true, Held: true, Clicks: 2})
	a.step(core.Input{})
	if a.widthsFor(requestPanes[0]).Weights != nil {
		t.Fatal("double click resets the widths")
	}
}

func TestNarrowColumnsKeepAMinimum(t *testing.T) {
	a, _ := newTestApp(t)
	drag(a, nameValueDividerX, 300, paramsHeaderY) // far left
	w := a.widthsFor(requestPanes[0])
	// the name column is at least MinColumnPx of the 393px of columns
	if min := float64(view.MinColumnPx) / 393 * 100; w.Weights[0] < min-0.5 {
		t.Fatalf("name weight %v is below the minimum share %v", w.Weights[0], min)
	}
}

// ---- colors in the settings menu ------------------------------------------

// Below the layout chooser: Contrast chips at y=374, then Colors rows from
// y=560 (Selection), 588, 616, 644 (Accent); field at x=1170, Reset at
// x=1234; the "Reset colors" button under the last row.
const (
	colorRow0Y   = 216
	colorFieldX  = 589
	colorResetX  = 652
	resetColorsX = 496
	resetColorsY = 328
)

// The settings dialog at 1280x800: 760x560 at (260, 120), a 44px title
// bar, a 170px list of pages on the left and the page's content from
// x=446, y=180. Appearance sections are 62px apart: Theme, Text size,
// Interface scale, Corners, Gradients, Request / response, Contrast.
const ()

// openSettingsPage opens the settings dialog on the page with the given id.
func openSettingsPage(t *testing.T, id string) (*App, string) {
	a, path := openSettingsMenu(t)
	a.prefs.Cat = id
	a.step(core.Input{})
	return a, path
}

func openSettingsMenu(t *testing.T) (*App, string) {
	a, path := newTestApp(t)
	click(a, menuBtnX, menuBtnY)
	if !a.menu {
		t.Fatal("test setup")
	}
	return a, path
}

func typeColor(a *App, tok core.Token, row int, text string) {
	in := a.colors.Input(pageHost{a}, tok)
	click(a, colorFieldX, colorRow0Y+row*core.RowH)
	in.SelectAll()
	a.typeText(text)
}

func TestContrastChooserStrengthensTheColorsAndIsSaved(t *testing.T) {
	a, path := openSettingsMenu(t)
	a.prefs.Search = view.NewTextField("contrast")
	before := theming.ContrastRatio(a.ctx.Th.Color(core.Selection), a.ctx.Th.Color(core.Text))
	clickText(t, a, "High", 0)
	if a.settings.Contrast != "high" {
		t.Fatalf("Contrast = %q", a.settings.Contrast)
	}
	after := theming.ContrastRatio(a.ctx.Th.Color(core.Selection), a.ctx.Th.Color(core.Text))
	if after < appearance.HighRatio-0.01 || after <= before {
		t.Fatalf("selection contrast %.2f -> %.2f", before, after)
	}
	if saved, _ := settingsfile.Load(path); saved.Contrast != "high" {
		t.Fatalf("saved %+v", saved)
	}
	clickText(t, a, "Normal", 0)
	if a.settings.Contrast != "normal" || theming.ContrastRatio(a.ctx.Th.Color(core.Selection), a.ctx.Th.Color(core.Text)) != before {
		t.Fatal("back to normal restores the default colors")
	}
}

func TestTypingAColorAppliesItLiveAndSavesIt(t *testing.T) {
	a, path := openSettingsPage(t, "colors")
	typeColor(a, core.Selection, 0, "#223344")
	if got := a.ctx.Th.Color(core.Selection); got != core.RGB(0x223344) {
		t.Fatalf("live color = %v", got)
	}
	saved, _ := settingsfile.Load(path)
	if saved.Colors["dark"]["selection"] != "#223344" {
		t.Fatalf("saved %+v", saved.Colors)
	}
	if a.colors.Note(core.Selection) != "" {
		t.Fatalf("note = %q", a.colors.Note(core.Selection))
	}
}

func TestAnUnreadableColorIsRefusedWithItsRatio(t *testing.T) {
	a, path := openSettingsPage(t, "colors")
	def := a.ctx.Th.Color(core.FindMatch)
	typeColor(a, core.FindMatch, 1, "#ffffff") // white under light text
	note := a.colors.Note(core.FindMatch)
	if !strings.Contains(note, "below 4.5:1") || !strings.Contains(note, "1.") {
		t.Fatalf("note = %q: it must say how far off the contrast is", note)
	}
	if a.ctx.Th.Color(core.FindMatch) != def {
		t.Fatal("a refused color must not apply")
	}
	if saved, _ := settingsfile.Load(path); len(saved.Colors["dark"]) != 0 {
		t.Fatalf("a refused color must not be saved: %+v", saved.Colors)
	}
	typeColor(a, core.FindMatch, 1, "zzz")
	if !strings.Contains(a.colors.Note(core.FindMatch), "#rrggbb") {
		t.Fatalf("note = %q", a.colors.Note(core.FindMatch))
	}
	typeColor(a, core.FindMatch, 1, "#334455")
	if a.colors.Note(core.FindMatch) != "" || a.ctx.Th.Color(core.FindMatch) != core.RGB(0x334455) {
		t.Fatalf("a readable color clears the note and applies: %q %v", a.colors.Note(core.FindMatch), a.ctx.Th.Color(core.FindMatch))
	}
}

func TestResetButtonsRestoreTheDefaults(t *testing.T) {
	a, path := openSettingsPage(t, "colors")
	def := a.ctx.Th.Color(core.Selection)
	typeColor(a, core.Selection, 0, "#223344")
	typeColor(a, core.FindMatch, 1, "#334455")
	click(a, colorResetX, colorRow0Y)
	if a.ctx.Th.Color(core.Selection) != def || a.colors.Input(pageHost{a}, core.Selection).Text != theming.FormatHex(def) {
		t.Fatalf("selection %v, field %q", a.ctx.Th.Color(core.Selection), a.colors.Input(pageHost{a}, core.Selection).Text)
	}
	if a.ctx.Th.Color(core.FindMatch) != core.RGB(0x334455) {
		t.Fatal("resetting one color must leave the others")
	}
	click(a, resetColorsX, resetColorsY)
	if a.ctx.Th.Color(core.FindMatch) == core.RGB(0x334455) {
		t.Fatal("Reset colors drops every override")
	}
	if saved, _ := settingsfile.Load(path); len(saved.Colors["dark"]) != 0 {
		t.Fatalf("saved %+v", saved.Colors)
	}
}

func TestSwitchingThemeShowsThatThemesColorsAndKeepsEachThemesOverrides(t *testing.T) {
	a, _ := openSettingsPage(t, "colors")
	typeColor(a, core.Selection, 0, "#223344")
	a.prefs.Cat, a.prefs.Search = "appearance", view.NewTextField("theme")
	a.step(core.Input{})
	clickText(t, a, "Light", 0)
	if a.ctx.Th.Color(core.Selection) != appearance.ThemeFor("light", "medium").Color(core.Selection) {
		t.Fatal("the dark override must not leak into the light theme")
	}
	if a.colors.Input(pageHost{a}, core.Selection).Text != theming.FormatHex(appearance.ThemeFor("light", "medium").Color(core.Selection)) {
		t.Fatalf("the field must show the light theme's color, got %q", a.colors.Input(pageHost{a}, core.Selection).Text)
	}
	clickText(t, a, "Dark", 0) // back to Dark
	if a.ctx.Th.Color(core.Selection) != core.RGB(0x223344) {
		t.Fatal("the dark override is remembered")
	}
}

func TestColorsAreRestoredAtStart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s := settings.Default()
	s.Contrast = "high"
	s.Colors = map[string]map[string]string{"dark": {"accent": "#aaccee"}}
	if err := settingsfile.Save(path, s); err != nil {
		t.Fatal(err)
	}
	a, err := NewApp(withAdapters(Options{SettingsPath: path}))
	if err != nil {
		t.Fatal(err)
	}
	if a.ctx.Th.Color(core.Accent) != core.RGB(0xaaccee) {
		t.Fatalf("accent = %v", a.ctx.Th.Color(core.Accent))
	}
	if theming.ContrastRatio(a.ctx.Th.Color(core.Selection), a.ctx.Th.Color(core.Text)) < appearance.HighRatio-0.01 {
		t.Fatal("high contrast is applied at start")
	}
}

func TestShortcutLabelsReadNaturally(t *testing.T) {
	labels := map[string]string{}
	for _, s := range shortcuts {
		labels[s.name] = s.label()
	}
	want := map[string]string{"Move row up": "Alt+Up", "Move row down": "Alt+Down", "Delete row": "Delete", "Settings": "Ctrl+,", "Next tab": "Ctrl+Tab"}
	for name, label := range want {
		if labels[name] != label {
			t.Errorf("%s: %q, want %q", name, labels[name], label)
		}
	}
}

// ---- configurable shortcuts ---------------------------------------------------

func combo(k ebiten.Key, ctrl, shift, alt bool) core.Input {
	return core.Input{Keys: []ebiten.Key{k}, Ctrl: ctrl, Shift: shift, Alt: alt}
}

func TestRecordingANewBindingRebindsTheAction(t *testing.T) {
	a, path := newTestApp(t)
	a.newRequest() // two tabs
	a.shortcutKeys.Start("close-tab")
	if a.shortcutKeys.Recording() != "close-tab" {
		t.Fatal("recording must start")
	}
	a.step(combo(ebiten.KeyQ, true, false, false))
	if a.shortcutKeys.Recording() != "" || a.settings.Shortcuts["close-tab"] != "Ctrl+Q" {
		t.Fatalf("recording %q shortcuts %v", a.shortcutKeys.Recording(), a.settings.Shortcuts)
	}
	if saved, _ := settingsfile.Load(path); saved.Shortcuts["close-tab"] != "Ctrl+Q" {
		t.Fatalf("saved %+v", saved.Shortcuts)
	}
	a.step(key(ebiten.KeyW, true, false)) // the old key does nothing now
	if len(a.tabs) != 2 {
		t.Fatal("Ctrl+W must no longer close a tab")
	}
	a.step(key(ebiten.KeyQ, true, false))
	if len(a.tabs) != 1 {
		t.Fatal("Ctrl+Q closes a tab")
	}
	if shortcutpage.KeyText(pageHost{a}, "close-tab") != "Ctrl+Q" {
		t.Fatalf("text = %q", shortcutpage.KeyText(pageHost{a}, "close-tab"))
	}
}

func TestEscapeCancelsRecordingAndBackspaceUnbinds(t *testing.T) {
	a, _ := newTestApp(t)
	a.shortcutKeys.Start("close-tab")
	a.step(core.Input{Escape: true})
	if a.shortcutKeys.Recording() != "" || len(a.settings.Shortcuts) != 0 || a.menu {
		t.Fatalf("recording %q shortcuts %v", a.shortcutKeys.Recording(), a.settings.Shortcuts)
	}

	a.newRequest()
	a.shortcutKeys.Start("close-tab")
	a.step(combo(ebiten.KeyBackspace, false, false, false))
	if v, ok := a.settings.Shortcuts["close-tab"]; !ok || v != "" {
		t.Fatalf("shortcuts = %v: Backspace stores an empty (unbound) binding", a.settings.Shortcuts)
	}
	a.step(key(ebiten.KeyW, true, false))
	if len(a.tabs) != 2 {
		t.Fatal("an unbound action fires nothing")
	}
	if shortcutpage.KeyText(pageHost{a}, "close-tab") != "Unbound" {
		t.Fatalf("text = %q", shortcutpage.KeyText(pageHost{a}, "close-tab"))
	}
}

// A key a text widget answers is refused and recording goes on; a key
// another action uses is not refused but offered with a Replace button (see
// shortcuteditor_test.go).
func TestRecordingRefusesKeysThatBelongToTextEditingAndKeepsListening(t *testing.T) {
	a, _ := newTestApp(t)
	a.shortcutKeys.Start("close-tab")
	a.step(combo(ebiten.KeyC, true, false, false))
	if !strings.Contains(a.shortcutKeys.Note("close-tab"), "text") || a.shortcutKeys.Recording() != "close-tab" {
		t.Fatalf("note %q recording %q", a.shortcutKeys.Note("close-tab"), a.shortcutKeys.Recording())
	}
	if len(a.settings.Shortcuts) != 0 {
		t.Fatal("refused combinations must change nothing")
	}
	a.step(combo(ebiten.KeyQ, true, false, false)) // a good one ends it and clears the note
	if a.shortcutKeys.Recording() != "" || a.shortcutKeys.Note("close-tab") != "" || a.settings.Shortcuts["close-tab"] != "Ctrl+Q" {
		t.Fatalf("recording %q note %q shortcuts %v", a.shortcutKeys.Recording(), a.shortcutKeys.Note("close-tab"), a.settings.Shortcuts)
	}
}

func TestAModifierKeyAloneDoesNotEndRecording(t *testing.T) {
	a, _ := newTestApp(t)
	a.shortcutKeys.Start("close-tab")
	a.step(core.Input{Keys: []ebiten.Key{ebiten.KeyControlLeft}, Ctrl: true})
	if a.shortcutKeys.Recording() != "close-tab" || len(a.settings.Shortcuts) != 0 {
		t.Fatalf("recording %q shortcuts %v", a.shortcutKeys.Recording(), a.settings.Shortcuts)
	}
}

func TestRebindingToTheDefaultStoresNothingAndResetRestoresIt(t *testing.T) {
	a, _ := newTestApp(t)
	a.shortcutKeys.Start("close-tab")
	a.step(combo(ebiten.KeyQ, true, false, false))
	a.shortcutKeys.Reset(pageHost{a}, "close-tab")
	if _, ok := a.settings.Shortcuts["close-tab"]; ok || shortcutpage.KeyText(pageHost{a}, "close-tab") != "Ctrl+W" {
		t.Fatalf("shortcuts %v text %q", a.settings.Shortcuts, shortcutpage.KeyText(pageHost{a}, "close-tab"))
	}
	a.shortcutKeys.Start("close-tab")
	a.step(combo(ebiten.KeyW, true, false, false)) // its own default
	if _, ok := a.settings.Shortcuts["close-tab"]; ok {
		t.Fatalf("a binding equal to the default must not be stored: %v", a.settings.Shortcuts)
	}
}

func TestSavedBindingsApplyFromTheStart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s := settings.Default()
	s.Shortcuts = map[string]string{"new-request": "Ctrl+N", "teleport": "Ctrl+J"}
	if err := settingsfile.Save(path, s); err != nil {
		t.Fatal(err)
	}
	a, err := NewApp(withAdapters(Options{SettingsPath: path}))
	if err != nil {
		t.Fatal(err)
	}
	a.w, a.h = winW, winH
	a.step(key(ebiten.KeyN, true, false))
	if len(a.tabs) != 2 {
		t.Fatal("the saved binding must work")
	}
	a.step(key(ebiten.KeyT, true, false))
	if len(a.tabs) != 2 {
		t.Fatal("the replaced default must not")
	}
}

func TestMenusShowTheConfiguredKey(t *testing.T) {
	a := jsonResponse(t, `{"ok":true}`)
	a.shortcutKeys.Start("find")
	a.step(combo(ebiten.KeyG, true, false, false))
	rightClick(a, 1000, 400)
	if got := itemByLabel(a, "Find").shortcut; got != "Ctrl+G" {
		t.Fatalf("the Find item shows %q", got)
	}
}

func TestShortcutSettingsRenderWhileRecording(t *testing.T) {
	a, _ := openSettingsPage(t, "shortcuts")
	a.shortcutKeys.Start("close-tab")
	a.step(combo(ebiten.KeyC, true, false, false)) // refused: it belongs to text editing, so a note shows
	a.step(core.Input{})                           // the menu with a recording row and a note must lay out
	if !a.menu || a.shortcutKeys.Recording() != "close-tab" {
		t.Fatal("the menu stays open while recording")
	}
}

// Both panes start their first row at the same height, so the two halves
// of the window line up across the divider.
func TestRequestAndResponsePanesShareTheirFirstRowBand(t *testing.T) {
	a := jsonResponse(t, `{"ok":true}`)
	clip := &fakeClip{}
	a.ctx.Clipboard = clip
	click(a, bulkBtnX, chipsY) // the request pane's first row
	if a.bulk == nil {
		t.Fatal("the Bulk edit link is in the first row of the request pane")
	}
	click(a, copyBtnX, chipsY+10) // the lower edge of the shared band, below the old toolbar
	if clip.text != `{"ok":true}` {
		t.Fatalf("the Copy button is not in the same band as Bulk edit, clipboard %q", clip.text)
	}
}

// openResponseRowMenu right-clicks down the response Headers table until
// a row menu opens (the exact row height is the table's business).
func openResponseRowMenu(t *testing.T, a *App) {
	t.Helper()
	a.cur().respTab = 1 // Headers
	a.step(core.Input{})
	for y := 150; y < 500; y += 4 {
		rightClick(a, 900, y)
		if a.ctxMenu != nil {
			return
		}
	}
	t.Fatal("right-clicking the response headers opened no menu")
}

func TestResponseHeaderRowsHaveACopyMenu(t *testing.T) {
	a := jsonResponse(t, `{}`)
	clip := &fakeClip{}
	a.ctx.Clipboard = clip
	openResponseRowMenu(t, a)
	labels := menuLabels(a)
	for _, want := range []string{"Copy value", "Copy name", "Copy as Name: Value", "Copy all headers"} {
		found := false
		for _, l := range labels {
			found = found || l == want
		}
		if !found {
			t.Fatalf("menu %v lacks %q", labels, want)
		}
	}
	for _, l := range labels {
		if strings.HasPrefix(l, "Insert") || l == "Delete" {
			t.Fatalf("a read-only table must not offer %q", l)
		}
	}
	h := (*responsePanes[1].rows(a, a.cur()))[0]
	pickMenu(t, a, "Copy as Name: Value")
	if clip.text != h.Name+": "+h.Value {
		t.Fatalf("clipboard %q", clip.text)
	}
}

func TestResponseHeaderMenuCopiesNameValueAndAll(t *testing.T) {
	a := jsonResponse(t, `{}`)
	clip := &fakeClip{}
	a.ctx.Clipboard = clip
	rows := *responsePanes[1].rows(a, a.cur())
	openResponseRowMenu(t, a)
	pickMenu(t, a, "Copy value")
	if clip.text != rows[0].Value {
		t.Fatalf("value: %q", clip.text)
	}
	openResponseRowMenu(t, a)
	pickMenu(t, a, "Copy name")
	if clip.text != rows[0].Name {
		t.Fatalf("name: %q", clip.text)
	}
	openResponseRowMenu(t, a)
	pickMenu(t, a, "Copy all headers")
	if !strings.Contains(clip.text, rows[0].Name+": "+rows[0].Value+"\n") || strings.Count(clip.text, "\n") < len(rows)-1 {
		t.Fatalf("all: %q", clip.text)
	}
}

func TestEditableTableMenuAlsoCopiesCells(t *testing.T) {
	a, _ := newTestApp(t)
	a.cur().Req.Params = []model.KV{{Name: "q", Value: "giou", Enabled: true}}
	clip := &fakeClip{}
	a.ctx.Clipboard = clip
	a.step(core.Input{})
	items := a.rowMenuItems(requestPanes[0], a.cur(), 0)
	var found bool
	for _, it := range items {
		if it.label == "Copy value" {
			it.run()
			found = true
		}
	}
	if !found || clip.text != "giou" {
		t.Fatalf("found %v clipboard %q", found, clip.text)
	}
}

// tipAt hovers (x, y) long enough and draws; it returns the tooltip shown.
func tipAt(a *App, x, y int) string {
	a.step(core.Input{X: x, Y: y})
	img := ebiten.NewImage(winW, winH)
	a.Draw(img) // the pointer comes to rest on the element
	a.ctx.Tick += 100
	a.dirty = true
	a.Draw(img)
	if a.ctx.TipShown {
		return a.ctx.TipText
	}
	return ""
}

func TestToolbarAndTabButtonsHaveTooltipsWithTheirShortcuts(t *testing.T) {
	a := jsonResponse(t, `{}`)
	for _, tt := range []struct {
		name string
		x, y int
		want string
	}{
		{"send", sendX, urlBarY, "Send the request (Ctrl+Enter)"},
		{"copy", copyBtnX, toolbarY, "Copy the response body"},
		{"curl", curlBtnX, toolbarY, "Copy the request as a cURL command"},
		{"menu", menuBtnX, menuBtnY, "Settings (Ctrl+,)"},
	} {
		if got := tipAt(a, tt.x, tt.y); got != tt.want {
			t.Errorf("%s: tooltip %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestDeleteButtonIsATrashCanWithATooltip(t *testing.T) {
	a, _ := newTestApp(t)
	found := false
	for x := 300; x < 900 && !found; x += 2 {
		found = tipAt(a, x, paramsRow0Y) == "Delete row"
	}
	if !found {
		t.Fatal("no Delete row tooltip along the first params row")
	}
}

func TestTooltipKeepsFramesComingWhileItWaits(t *testing.T) {
	a, _ := newTestApp(t)
	a.step(core.Input{X: sendX, Y: urlBarY})
	a.Draw(ebiten.NewImage(winW, winH))
	if !a.ctx.TipWaiting {
		t.Fatal("hovering a button with a tooltip must be waiting")
	}
	a.dirty = false
	a.step(core.Input{X: sendX, Y: urlBarY}) // nothing else changed
	if !a.dirty {
		t.Fatal("frames must keep being drawn until the tooltip shows")
	}
}

func TestManyTabsScrollTheStripAndKeepTheActiveTabInView(t *testing.T) {
	a, _ := newTestApp(t)
	for range 14 {
		a.newRequest()
	}
	a.step(core.Input{})
	if a.tabsScroll.X <= 0 {
		t.Fatalf("the new, active tab is past the right edge: X = %d", a.tabsScroll.X)
	}
	x := a.tabsScroll.X
	a.step(core.Input{X: 600, Y: tabStripY, WheelX: 1}) // two-finger swipe right: back toward the first tabs
	if a.tabsScroll.X >= x {
		t.Fatalf("a sideways swipe must scroll the strip back: %d -> %d", x, a.tabsScroll.X)
	}
	x = a.tabsScroll.X
	a.step(core.Input{X: 600, Y: tabStripY, Wheel: -1}) // the mouse wheel scrolls it too
	if a.tabsScroll.X <= x {
		t.Fatalf("the wheel must scroll the strip forward: %d -> %d", x, a.tabsScroll.X)
	}
	x = a.tabsScroll.X
	click(a, sidebarW+4, tabStripY) // the left arrow, at the strip's left edge
	if a.tabsScroll.X >= x {
		t.Fatalf("clicking the left arrow must scroll back: %d -> %d", x, a.tabsScroll.X)
	}
	if len(a.tabs) != 15 || a.active != 14 {
		t.Fatalf("an arrow click must not select or close a tab: %d tabs, active %d", len(a.tabs), a.active)
	}
}

func TestLeftDragOnAnOverflowingTabStripPansItInsteadOfReorderingTabs(t *testing.T) {
	a, _ := newTestApp(t)
	for range 14 {
		a.newRequest()
	}
	a.step(core.Input{})
	order := append([]*reqTab(nil), a.tabs...)
	x0 := a.tabsScroll.X // the active (last) tab was revealed: the strip is at its right end
	a.step(core.Input{X: 600, Y: tabStripY, Click: true, Held: true})
	a.step(core.Input{X: 700, Y: tabStripY, Held: true}) // drag right: earlier tabs come into view
	a.step(core.Input{X: 800, Y: tabStripY, Held: true})
	if want := x0 - 200; a.tabsScroll.X != want {
		t.Fatalf("X = %d, want %d: the strip follows the pointer", a.tabsScroll.X, want)
	}
	for i := range order {
		if a.tabs[i] != order[i] {
			t.Fatalf("tab %d moved: a left drag on an overflowing strip must not reorder", i)
		}
	}
	a.step(core.Input{X: 800, Y: tabStripY}) // release
	if a.dragTab != -1 {
		t.Fatalf("dragTab = %d", a.dragTab)
	}
}

// A response with nested blocks: one match deep inside one item, one at
// the top, and plenty of siblings that must disappear while searching.
const compactBody = `{"total":3,"items":[` +
	`{"id":1,"name":"alpha","tags":["red","green","blue","yellow","orange","purple","brown","black"],"owner":{"login":"ann","site":"https://a.example.org/very/long/path/to/ann"}},` +
	`{"id":2,"name":"beta","tags":["red","green","blue","yellow","orange","purple","brown","black"],"owner":{"login":"bob","site":"https://b.example.org/very/long/path/to/bob"}},` +
	`{"id":3,"name":"gamma","tags":["cyan","magenta","teal","olive","navy","maroon","lime","aqua"],"owner":{"login":"cy","site":"https://c.example.org/very/long/path/to/cy"}}]}`

func viewText(t *reqTab) string {
	var sb strings.Builder
	for _, i := range t.view.Visible() {
		for _, sp := range t.view.Lines[i].Spans {
			sb.WriteString(sp.Text)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func TestCompactChipShowsJSONCompactlyWithDeepBlocksFolded(t *testing.T) {
	a := jsonResponse(t, compactBody)
	tab := a.cur()
	pretty := len(tab.view.Visible())
	click(a, compactX, toolbarY)
	if !tab.compact || tab.raw {
		t.Fatalf("compact %v raw %v", tab.compact, tab.raw)
	}
	text := viewText(tab)
	if got := len(tab.view.Visible()); got >= pretty {
		t.Fatalf("compact shows %d lines, pretty showed %d", got, pretty)
	}
	// Items are blocks at depth 2: folded, so only their previews show.
	if strings.Contains(text, `"login": "ann"`) && !strings.Contains(text, `…`) {
		t.Fatalf("deep blocks must be folded to previews:\n%s", text)
	}
	if !strings.Contains(text, `"total": 3`) {
		t.Fatalf("top-level values stay:\n%s", text)
	}
	click(a, rawX, toolbarY)
	if tab.compact || !tab.raw {
		t.Fatal("Raw replaces Compact")
	}
	click(a, prettyX, toolbarY)
	if tab.compact || tab.raw {
		t.Fatal("Pretty replaces both")
	}
}

func TestFindInCompactViewKeepsOnlyMatchingLeavesTheirChildrenAndTheirPath(t *testing.T) {
	a := jsonResponse(t, compactBody)
	tab := a.cur()
	click(a, compactX, toolbarY)
	a.openFind()
	a.typeText("cy")
	text := viewText(tab)
	if !strings.Contains(text, `"login": "cy"`) {
		t.Fatalf("the match must be visible and unfolded:\n%s", text)
	}
	for _, gone := range []string{`"alpha"`, `"beta"`, `"ann"`, `"bob"`, `"total"`, `cyan`} {
		if gone == "cyan" {
			continue // "cyan" also contains the query: it is a match, so it stays
		}
		if strings.Contains(text, gone) {
			t.Fatalf("%s is unrelated to the match and must be filtered out:\n%s", gone, text)
		}
	}
	if !strings.Contains(text, `"items": [`) || !strings.HasPrefix(text, "{") {
		t.Fatalf("the path to the match stays:\n%s", text)
	}
	if len(tab.view.Matches()) == 0 || tab.matchIdx != 0 {
		t.Fatalf("matches %d idx %d", len(tab.view.Matches()), tab.matchIdx)
	}
	// A matching block keeps all of its children.
	a.find(tab, "owner")
	text = viewText(tab)
	for _, want := range []string{`"login": "ann"`, `"login": "bob"`, `"login": "cy"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("children of a matching block must stay, missing %s:\n%s", want, text)
		}
	}
	// Clearing the search restores the full compact view with its folds.
	a.find(tab, "")
	if !strings.Contains(viewText(tab), `"total": 3`) {
		t.Fatal("an empty query shows everything again")
	}
	a.closeFind(tab)
	if !strings.Contains(viewText(tab), `"total": 3`) {
		t.Fatal("closing the find bar restores the view")
	}
}

// find replaces the search text, as typing would.
func (a *App) find(t *reqTab, text string) {
	t.find = view.NewTextField(text)
	t.find.Focused = true
	a.runFind(t)
}

func TestNewTabsShowResponsesCompactByDefault(t *testing.T) {
	a := newCompactTestApp(t)
	if !a.cur().compact || a.cur().raw {
		t.Fatalf("compact %v raw %v", a.cur().compact, a.cur().raw)
	}
	a.newRequest()
	if !a.cur().compact {
		t.Fatal("a new tab starts compact too")
	}
}

func TestResponseViewChipsAreCompactPrettyRawInThatOrder(t *testing.T) {
	a := jsonResponse(t, `{"a":1}`)
	tab := a.cur()
	click(a, rawX, toolbarY)
	if !tab.raw {
		t.Fatal("the third chip is Raw")
	}
	click(a, prettyX, toolbarY)
	if tab.raw || tab.compact {
		t.Fatal("the second chip is Pretty")
	}
	click(a, compactX, toolbarY)
	if !tab.compact {
		t.Fatal("the first chip is Compact")
	}
	if !(compactX < prettyX && prettyX < rawX) {
		t.Fatalf("order %d %d %d", compactX, prettyX, rawX)
	}
}

func TestResponseViewSettingAppliesToOpenTabsAndIsSaved(t *testing.T) {
	a, path := openSettingsPage(t, "appearance")
	a.prefs.Search = view.NewTextField("response view")
	clickText(t, a, "Compact", 0)
	if !a.cur().compact || a.settings.ResponseView != "compact" {
		t.Fatalf("compact %v setting %q", a.cur().compact, a.settings.ResponseView)
	}
	if saved, _ := settingsfile.Load(path); saved.ResponseView != "compact" {
		t.Fatalf("saved %+v", saved)
	}
	clickText(t, a, "Pretty", 0)
	if a.cur().compact || a.cur().raw || a.settings.ResponseView != "pretty" {
		t.Fatalf("compact %v raw %v setting %q", a.cur().compact, a.cur().raw, a.settings.ResponseView)
	}
}

func TestFindWorksInEveryResponseView(t *testing.T) {
	for _, mode := range []struct {
		name         string
		raw, compact bool
	}{{"pretty", false, false}, {"raw", true, false}, {"compact", false, true}} {
		a := jsonResponse(t, compactBody)
		tab := a.cur()
		a.setView(tab, mode.raw, mode.compact)
		a.openFind()
		a.typeText("login")
		if len(tab.view.Matches()) != 3 || tab.matchIdx != 0 {
			t.Fatalf("%s: %d matches, current %d", mode.name, len(tab.view.Matches()), tab.matchIdx)
		}
		a.findStep(tab, 1)
		if tab.matchIdx != 1 {
			t.Fatalf("%s: stepping goes to the second match, got %d", mode.name, tab.matchIdx)
		}
		a.closeFind(tab)
		if len(tab.view.Matches()) != 0 {
			t.Fatalf("%s: closing the bar clears the highlights", mode.name)
		}
	}
}
