package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	"github.com/hajimehoshi/ebiten/v2"
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
	if a.ctx.Th.FontSize != 17 {
		t.Fatalf("font size = %v, want 17", a.ctx.Th.FontSize)
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
	if a.ctx.Th.Color(widget.Bg) != widget.Light().Color(widget.Bg) || a.ctx.Th.FontSize != 13 || !a.menu {
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

// Cells of the Params table at 1280x800: request pane starts at x=260,
// the table at x=272, columns: actions 36, name 115, value 165, description 132.
const (
	paramsRow0Y   = 262
	paramsRow3Y   = 346 // the placeholder row below the three params
	nameCellX     = 365
	valueCellX    = 500
	descCellX     = 650
	farAwayX      = 900
	farAwayY      = 600
	headersTabIdx = 2
)

func (a *App) typeText(s string) { a.step(widget.Input{Typed: []rune(s)}) }

func TestEditingAParamValueUpdatesTheURL(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, valueCellX, paramsRow0Y)
	if a.edit == nil || a.edit.row != 0 || a.edit.col != 1 {
		t.Fatalf("edit = %+v", a.edit)
	}
	a.typeText("rust") // the old text is selected, so typing replaces it
	a.step(widget.Input{Enter: true})

	req := a.cur().req
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
	a.step(widget.Input{Enter: true})
	click(a, descCellX, paramsRow0Y)
	a.typeText("what to find")
	a.step(widget.Input{Enter: true})

	p := a.cur().req.Params[0]
	if p.Name != "search" || p.Description != "what to find" || !strings.Contains(a.cur().req.URL, "search=gioui") {
		t.Fatalf("param %+v url %q", p, a.cur().req.URL)
	}
}

func TestEscapeCancelsACellEdit(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, valueCellX, paramsRow0Y)
	a.typeText("zzz")
	a.step(widget.Input{Escape: true})
	if a.edit != nil || a.cur().req.Params[0].Value != "gioui" {
		t.Fatalf("edit %+v value %q", a.edit, a.cur().req.Params[0].Value)
	}
}

func TestClickingElsewhereCommitsACellEdit(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, valueCellX, paramsRow0Y)
	a.typeText("go")
	click(a, farAwayX, farAwayY)
	a.step(widget.Input{})
	if a.edit != nil || a.cur().req.Params[0].Value != "go" {
		t.Fatalf("edit %+v value %q", a.edit, a.cur().req.Params[0].Value)
	}
}

func TestClickingAnotherCellCommitsAndMovesTheEdit(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, valueCellX, paramsRow0Y)
	a.typeText("go")
	click(a, valueCellX, paramsRow0Y+widget.RowH) // the order row
	if a.cur().req.Params[0].Value != "go" || a.edit == nil || a.edit.row != 1 {
		t.Fatalf("value %q edit %+v", a.cur().req.Params[0].Value, a.edit)
	}
}

func TestTabMovesToTheNextCellAndShiftTabBack(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, nameCellX, paramsRow0Y)
	a.typeText("k")
	a.step(widget.Input{Tab: true})
	if a.edit == nil || a.edit.row != 0 || a.edit.col != 1 || a.cur().req.Params[0].Name != "k" {
		t.Fatalf("after Tab: edit %+v name %q", a.edit, a.cur().req.Params[0].Name)
	}
	a.step(widget.Input{Tab: true})
	a.step(widget.Input{Tab: true}) // description, then the next row's name
	if a.edit == nil || a.edit.row != 1 || a.edit.col != 0 {
		t.Fatalf("Tab past the last column must wrap to the next row, edit %+v", a.edit)
	}
	a.step(widget.Input{Tab: true, Shift: true})
	if a.edit == nil || a.edit.row != 0 || a.edit.col != 2 {
		t.Fatalf("Shift+Tab goes back, edit %+v", a.edit)
	}
}

func TestTypingInThePlaceholderRowAddsARow(t *testing.T) {
	a, _ := newTestApp(t)
	click(a, nameCellX, paramsRow3Y)
	a.typeText("page")
	a.step(widget.Input{Tab: true})
	a.typeText("2")
	a.step(widget.Input{Enter: true})

	req := a.cur().req
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
	a.step(widget.Input{Enter: true})
	if len(a.cur().req.Params) != 3 || a.edit != nil {
		t.Fatalf("params %+v edit %+v", a.cur().req.Params, a.edit)
	}
}

func TestHeadersTableIsEditableToo(t *testing.T) {
	a, _ := newTestApp(t)
	a.cur().paneTab = headersTabIdx
	a.step(widget.Input{})
	click(a, nameCellX, paramsRow0Y) // headers have no rows: the placeholder is the first row
	a.typeText("X-Test")
	a.step(widget.Input{Tab: true})
	a.typeText("1")
	a.step(widget.Input{Enter: true})
	h := a.cur().req.Headers
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
	a.step(widget.Input{})
	click(a, 1000, paramsRow0Y)
	if a.edit != nil {
		t.Fatalf("response headers must stay read-only, edit %+v", a.edit)
	}
}

func key(k ebiten.Key, ctrl, shift bool) widget.Input {
	return widget.Input{Keys: []ebiten.Key{k}, Ctrl: ctrl, Shift: shift}
}

func TestCtrlEnterSendsTheRequest(t *testing.T) {
	srv := jsonServer(t)
	a, _ := newTestApp(t)
	tab := a.cur()
	pointTabAt(tab, srv.URL)
	a.step(key(ebiten.KeyEnter, true, false))
	waitForResponse(t, a, tab)
	if tab.resp.Status != 200 {
		t.Fatalf("resp = %+v", tab.resp)
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
	if len(a.tabs) != 1 || a.active != 0 || a.cur().req.Name != "Search Repos" {
		t.Fatalf("Ctrl+W: %d tabs, active %d (%s)", len(a.tabs), a.active, a.cur().req.Name)
	}
	a.step(key(ebiten.KeyW, true, false))
	if len(a.tabs) != 1 || a.cur().req.Name != "New Request" {
		t.Fatalf("closing the last tab must leave a blank request, got %d tabs (%s)", len(a.tabs), a.cur().req.Name)
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
	if len(a.tabs) != 1 || a.menu || a.cur().sending {
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
	a.h = 200 // the tree area is about 128px: five rows
	for _, f := range []string{"User", "Realtime"} {
		a.expanded[a.col.Name+"/"+f] = true
	}
	a.step(widget.Input{})
	rowsN := len(treeRows(a.col.Folder, a.expanded))
	if rowsN*widget.RowH <= a.h-headerH-statusH {
		t.Fatalf("test setup: %d rows fit, nothing to scroll", rowsN)
	}

	a.step(widget.Input{X: 100, Y: 100, Wheel: -5})
	if a.treeScroll.Y <= 0 {
		t.Fatalf("tree scroll = %d after wheeling over it", a.treeScroll.Y)
	}

	// scroll to the end, then click the last row: it must open that request
	a.step(widget.Input{X: 100, Y: 100, Wheel: -50})
	last := treeRows(a.col.Folder, a.expanded)[rowsN-1]
	click(a, 100, a.h-statusH-widget.RowH/2)
	if a.cur().req != last.Req {
		t.Fatalf("clicking the bottom row opened %q, want %q", a.cur().req.Name, last.Name)
	}
}

// Body pane at 1280x800: type chips at y=206 (None, JSON, Text, Form,
// Multipart from x=272), the Format button at the right end of that
// row, the editor below y=228, a form table's placeholder row at y=270.
const (
	bodyTabIdx = 1
	chipsY     = 206
	noneChipX  = 300
	jsonChipX  = 374
	textChipX  = 444
	formChipX  = 514
	multiChipX = 600
	formatX    = 722
	editorX    = 500
	editorY    = 320
	formRow0Y  = 270
)

func openBodyPane(t *testing.T) *App {
	a, _ := newTestApp(t)
	a.cur().paneTab = bodyTabIdx
	a.step(widget.Input{})
	return a
}

func TestBodyTypeChipsSwitchTheTypeAndKeepTheText(t *testing.T) {
	a := openBodyPane(t)
	req := a.cur().req
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
	a.step(widget.Input{Enter: true})
	a.typeText("x")
	if got := a.cur().req.Body; got != "{\"a\":1}\nx" {
		t.Fatalf("Body = %q", got)
	}
}

func TestFormatButtonIndentsValidJSONOnly(t *testing.T) {
	a := openBodyPane(t)
	req := a.cur().req
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
	req := a.cur().req
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
	req := a.cur().req
	click(a, formChipX, chipsY)
	click(a, nameCellX, formRow0Y)
	a.typeText("user")
	a.step(widget.Input{Tab: true})
	a.typeText("ann")
	a.step(widget.Input{Enter: true})
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
	tab.req.Method, tab.req.BodyType = "POST", model.BodyForm
	tab.req.Form = []model.KV{{Name: "q", Value: "go lang", Enabled: true}}
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
	a.step(widget.Input{})
	click(a, textChipX, chipsY)
	click(a, editorX, editorY)
	a.typeText("second")
	a.switchTab(-1)
	if a.cur().req.Body != "first" || second.req.Body != "second" {
		t.Fatalf("bodies %q and %q", a.cur().req.Body, second.req.Body)
	}
}

// tabX is the left edge and width of tab i in the request-tab strip,
// from the layout the view uses: 14px padding, a fixed badge slot, 8px
// gaps, the name, a fixed slot for the unsaved dot and a 24px close
// button. Nothing but the name changes a tab's width.
func (a *App) tabX(i int) (left, width int) {
	x := 260
	for j, t := range a.tabs {
		wd := 14 + tabBadgeW + 8 + a.ctx.Width(a.ctx.F.UI, t.req.Name) + 8 + tabDirtyW + 24
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
	if len(a.tabs) != 1 || a.cur().req.Name != "Repository Info" {
		t.Fatalf("%d tabs, current %q", len(a.tabs), a.cur().req.Name)
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

func TestMiddleClickClosesATab(t *testing.T) {
	a := twoTabs(t)
	left, width := a.tabX(1)
	a.step(widget.Input{X: left + width/2, Y: tabStripY, MiddleClick: true})
	if len(a.tabs) != 1 || a.cur().req.Name != "Search Repos" {
		t.Fatalf("%d tabs, current %q", len(a.tabs), a.cur().req.Name)
	}
}

func TestClosingATabBeforeTheActiveOneKeepsTheSameTabActive(t *testing.T) {
	a := twoTabs(t)
	a.closeTab(0)
	if a.active != 0 || a.cur().req.Name != "Repository Info" {
		t.Fatalf("active %d current %q", a.active, a.cur().req.Name)
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
	a.step(widget.Input{Edits: []widget.Edit{widget.EditBackspace}})
	if dirty(tab) {
		t.Fatal("restoring the original text makes it clean again")
	}
	click(a, 739, 262) // delete a param
	if !dirty(tab) {
		t.Fatal("deleting a param is a change")
	}
}

func TestTabWidthDoesNotChangeWhenTheRequestChanges(t *testing.T) {
	a := twoTabs(t)
	left, width := a.tabX(0)
	req := a.tabs[0].req
	req.Method = "DELETE"                                                 // a longer badge
	req.Headers = append(req.Headers, model.KV{Name: "X", Enabled: true}) // and an unsaved dot
	a.step(widget.Input{})
	if !dirty(a.tabs[0]) {
		t.Fatal("test setup: the tab must be dirty")
	}

	// The × sits where it did before the edits; if the tab had grown it
	// would be somewhere else and the click would only select the tab.
	click(a, left+width-12, tabStripY)
	if len(a.tabs) != 1 || a.cur().req.Name != "Repository Info" {
		t.Fatalf("%d tabs: the tab changed size when its request changed", len(a.tabs))
	}
}

func TestTheTabSlotsFitTheirContentAtEveryTextSize(t *testing.T) {
	for _, size := range textSizeChoices {
		ctx, err := widget.NewCtx(ThemeFor("dark", size.id))
		if err != nil {
			t.Fatal(err)
		}
		for _, badge := range []string{"GET", "POST", "PUT", "PATCH", "DEL", "WS", "MCP"} {
			if got := ctx.Width(ctx.F.Small, badge); got > tabBadgeW {
				t.Errorf("%s: %q is %dpx wide, the badge slot is %dpx", size.id, badge, got, tabBadgeW)
			}
		}
		if got := ctx.Width(ctx.F.UI, dirtyMark); got > tabDirtyW {
			t.Errorf("%s: the dirty mark is %dpx wide, the slot is %dpx", size.id, got, tabDirtyW)
		}
	}
}

func TestDraggingATabReordersIt(t *testing.T) {
	a := twoTabs(t)
	first := a.tabs[0]
	l0, w0 := a.tabX(0)
	l1, w1 := a.tabX(1)

	a.step(widget.Input{X: l0 + 20, Y: tabStripY, Click: true, Held: true}) // press the first tab
	a.step(widget.Input{X: l1 + w1/2, Y: tabStripY, Held: true})            // drag over the second
	if a.tabs[1] != first {
		t.Fatalf("the dragged tab must move to the second place")
	}
	a.step(widget.Input{X: l1 + w1/2, Y: tabStripY}) // release
	a.step(widget.Input{X: l0 + w0/2, Y: tabStripY, Held: true})
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
	a.step(widget.Input{X: l0 + 20, Y: tabStripY, Click: true, Held: true})
	a.step(widget.Input{X: l1 + w1/2, Y: tabStripY, Held: true})
	if a.cur() == active {
		t.Fatal("pressing the first tab activates it, so it is the active one")
	}
	if a.cur() != a.tabs[1] {
		t.Fatal("the active tab must travel with the drag")
	}
}
