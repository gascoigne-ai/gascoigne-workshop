package ui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/widget"
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

func treeRowY(i int) int { return headerH + i*widget.RowH + widget.RowH/2 }

func newTestApp(t *testing.T) (*App, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.json")
	a, err := NewApp(Options{SettingsPath: path})
	if err != nil {
		t.Fatal(err)
	}
	a.w, a.h = winW, winH
	a.step(widget.Input{}) // first frame: lay everything out once
	return a, path
}

func click(a *App, x, y int) { a.step(widget.Input{X: x, Y: y, Click: true, Held: true}) }

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
	t.req.URL = url
	t.url = widget.NewTextField(url)
}

func waitForResponse(t *testing.T, a *App, tab *reqTab) {
	t.Helper()
	for range 400 {
		a.step(widget.Input{})
		if tab.resp != nil {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("no response arrived")
}

func TestClickingARequestInTheTreeOpensItsTab(t *testing.T) {
	a, _ := newTestApp(t)
	if len(a.tabs) != 1 || a.cur().req.Name != "Search Repos" {
		t.Fatalf("start: %d tabs, current %q", len(a.tabs), a.cur().req.Name)
	}

	click(a, 100, treeRowY(4)) // Repository Info
	if len(a.tabs) != 2 || a.cur().req.Name != "Repository Info" {
		t.Fatalf("after click: %d tabs, current %q", len(a.tabs), a.cur().req.Name)
	}

	click(a, 100, treeRowY(4)) // the same request again focuses its tab
	if len(a.tabs) != 2 {
		t.Fatalf("reopening must not add a tab, have %d", len(a.tabs))
	}
}

func TestClickingAFolderTogglesIt(t *testing.T) {
	a, _ := newTestApp(t)
	before := len(treeRows(a.col.Folder, a.expanded))
	click(a, 100, treeRowY(1)) // User folder, collapsed at start
	if got := len(treeRows(a.col.Folder, a.expanded)); got != before+1 {
		t.Fatalf("rows %d -> %d, want one more (Get User)", before, got)
	}
	click(a, 100, treeRowY(1))
	if got := len(treeRows(a.col.Folder, a.expanded)); got != before {
		t.Fatalf("second click must collapse again, rows = %d", got)
	}
}

func TestNewRequestAndSwitchingTabs(t *testing.T) {
	a, _ := newTestApp(t)
	a.newRequest()
	if len(a.tabs) != 2 || a.active != 1 || a.cur().req.Name != "New Request" {
		t.Fatalf("tabs %d active %d", len(a.tabs), a.active)
	}
	click(a, 280, headerH+tabsH/2) // the first tab sits at the left edge of the strip
	if a.active != 0 {
		t.Fatalf("active = %d, want 0", a.active)
	}
}

func TestProtocolAndMethodChipsCycle(t *testing.T) {
	a, _ := newTestApp(t)
	req := a.cur().req

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
	if req.Protocol != "rest" || req.Method != "POST" {
		t.Fatalf("after a full cycle: %q %q", req.Protocol, req.Method)
	}
}

func TestSwitchingToRESTRestoresAMethod(t *testing.T) {
	a, _ := newTestApp(t)
	req := a.cur().req
	req.Protocol, req.Method = "mcp", ""
	a.cycleProtocol(a.cur())
	if req.Protocol != "rest" || req.Method != "GET" {
		t.Fatalf("got %q %q, want rest GET", req.Protocol, req.Method)
	}
}

func TestTypingInTheURLFieldUpdatesParams(t *testing.T) {
	a, _ := newTestApp(t)
	req := a.cur().req
	click(a, 1100, urlBarY) // inside the field, past the end of the text: cursor at the end
	a.step(widget.Input{Typed: []rune("&page=2")})

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
	req := a.cur().req
	click(a, 290, 262) // checkbox of the first Params row (q)
	if req.Params[0].Enabled || strings.Contains(req.URL, "q=") {
		t.Fatalf("q must be disabled and gone from %q", req.URL)
	}
	if !strings.Contains(req.URL, "order=desc") {
		t.Fatalf("other params must stay in %q", req.URL)
	}
	click(a, 290, 262)
	if !req.Params[0].Enabled || !strings.Contains(req.URL, "q=gioui") {
		t.Fatalf("toggling back must restore q in %q", req.URL)
	}
}

func TestDeletingAParamRewritesTheURL(t *testing.T) {
	a, _ := newTestApp(t)
	req := a.cur().req
	click(a, 739, 262) // delete button of the first Params row
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
	if !tab.sending {
		t.Fatal("a click on Send must start a request")
	}
	waitForResponse(t, a, tab)

	if tab.sending || tab.resp.Status != 200 || tab.resp.Err != "" {
		t.Fatalf("sending %v resp %+v", tab.sending, tab.resp)
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
	a.step(widget.Input{Enter: true})
	waitForResponse(t, a, tab)
	if tab.resp.Status != 200 {
		t.Fatalf("resp = %+v", tab.resp)
	}
}

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
	click(a, sendX, urlBarY)
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
	if tab.resp.Err == "" || a.status != "Request failed" {
		t.Fatalf("err %q status %q", tab.resp.Err, a.status)
	}
}

func TestOtherProtocolsAnswerThroughTheSameUI(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	click(a, 300, urlBarY) // rest -> websocket
	click(a, sendX, urlBarY)
	waitForResponse(t, a, tab)
	if !strings.Contains(tab.resp.Err, "not implemented") {
		t.Fatalf("resp = %+v", tab.resp)
	}
}

func TestSettingsMenuOpensChangesThemeAndSaves(t *testing.T) {
	a, path := newTestApp(t)
	click(a, menuBtnX, menuBtnY)
	if !a.menu {
		t.Fatal("the three-dot button must open the menu")
	}

	click(a, 1200, 126) // the Light chip
	if a.settings.Theme != "light" || !a.menu {
		t.Fatalf("theme %q, menu open %v (choosing must keep it open)", a.settings.Theme, a.menu)
	}
	if a.ctx.Th.Color(widget.Bg) != widget.Light().Color(widget.Bg) {
		t.Fatal("the live theme did not change")
	}
	saved, err := settings.Load(path)
	if err != nil || saved.Theme != "light" {
		t.Fatalf("saved %+v err %v", saved, err)
	}

	click(a, 1240, 188) // the Large text size chip
	if a.ctx.Th.FontSize != 16 {
		t.Fatalf("font size = %v, want 16", a.ctx.Th.FontSize)
	}
	if saved, _ := settings.Load(path); saved.TextSize != "large" || saved.Theme != "light" {
		t.Fatalf("saved %+v", saved)
	}
}

func TestMenuClosesOnEscapeAndOnOutsideClick(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, menuBtnX, menuBtnY)
	a.step(widget.Input{Escape: true})
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
	a, err := NewApp(Options{SettingsPath: filepath.Join(blocker, "settings.json")}) // a file where a directory is needed
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
	a, err := NewApp(Options{SettingsPath: path})
	if err != nil {
		t.Fatal(err)
	}
	if a.settings != settings.Default() || !strings.Contains(a.status, "parse settings") {
		t.Fatalf("settings %+v status %q", a.settings, a.status)
	}
}

func TestSettingsStartFromTheSavedChoice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := (settings.Settings{Theme: "light", TextSize: "small"}).Save(path); err != nil {
		t.Fatal(err)
	}
	a, err := NewApp(Options{SettingsPath: path, OpenMenu: true})
	if err != nil {
		t.Fatal(err)
	}
	if a.ctx.Th.Color(widget.Bg) != widget.Light().Color(widget.Bg) || a.ctx.Th.FontSize != 12 || !a.menu {
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
		"no response": func() { tab.resp, tab.sending = nil, false },
		"sending":     func() { tab.resp, tab.sending = nil, true },
		"error":       func() { tab.resp, tab.sending = &respErr, false },
		"ok": func() {
			tab.sending = false
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
				a.step(widget.Input{}) // panics on a bad pane
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
	tab.req.Params[0].Enabled = false
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
		{"sending", reqTab{sending: true}, "Sending...", false},
		{"not sent", reqTab{}, "Send the request to see the response", false},
		{"failed", reqTab{resp: &respErr}, "boom", true},
		{"ok", reqTab{resp: &respOK}, "", false},
	}
	for _, tt := range tests {
		msg, isErr := noResponse(&tt.tab)
		if msg != tt.wantMsg || isErr != tt.wantErr {
			t.Errorf("%s: got %q %v", tt.name, msg, isErr)
		}
	}
}
