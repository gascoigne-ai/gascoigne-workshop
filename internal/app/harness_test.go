package app

import (
	"context"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"

	"cmp"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/adapter/collectionstore"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history/adapter/historyfile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/adapter/protocol"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/adapter/settingsfile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/adapter/statefile"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/fonts"
	"io"
	"net/http"
	"net/http/httptest"
)

// The helpers shared by the app tests live here; a test file holds only
// what its own tests need.

// ---- the app under test ----------------------------------------------------

// appBuild is what testApp starts an App from.
type appBuild struct {
	opts    Options
	prefs   settings.Settings
	path    string // the settings file; "" is a new file in a temporary folder
	memory  bool   // settings kept in memory only
	written bool   // prefs are saved to the settings file before the start
	dir     string // the workspace folder
}

// appOption changes how testApp starts the App.
type appOption func(*appBuild)

// withSettings adjusts the settings the test app starts with.
func withSettings(adjust func(*settings.Settings)) appOption {
	return func(build *appBuild) { adjust(&build.prefs) }
}

// defaultLayout puts the layout setting back to its default.
func defaultLayout(prefs *settings.Settings) { prefs.Layout = settings.Default().Layout }

// withOptions starts the App with opts; the settings file and the
// workspace folder come from the other options.
func withOptions(opts Options) appOption {
	return func(build *appBuild) { build.opts = opts }
}

// inWorkspace opens the workspace folder dir.
func inWorkspace(dir string) appOption {
	return func(build *appBuild) { build.dir = dir }
}

// settingsAt starts from the settings file at path, as it is.
func settingsAt(path string) appOption {
	return func(build *appBuild) { build.path, build.written = path, false }
}

// untouchedSettings starts from a settings file that does not exist, so
// every setting has its default.
func untouchedSettings() appOption {
	return func(build *appBuild) { build.written = false }
}

// memorySettings keeps the settings in memory, with no file.
func memorySettings() appOption {
	return func(build *appBuild) { build.memory, build.written = true, false }
}

// testApp is a test app, after its first frame. Unless an option says
// otherwise it has the panes side by side, whose geometry the click
// coordinates of most tests assume, and the pretty-printed response,
// which most tests were written against, in a settings file of its own
// (a.SettingsPath()). The defaults have their own tests: defaultLayout
// for the layout, untouchedSettings for the response view.
func testApp(t *testing.T, options ...appOption) *App {
	t.Helper()
	build := appBuild{prefs: settings.Default(), written: true}
	build.prefs.ResponseView = "pretty"
	build.prefs.Layout = "side"
	for _, option := range options {
		option(&build)
	}
	if build.path == "" && !build.memory {
		build.path = filepath.Join(t.TempDir(), "settings.json")
	}
	if build.written {
		if err := settingsDisk.Save(build.path, build.prefs); err != nil {
			t.Fatal(err)
		}
	}
	build.opts.SettingsPath = build.path
	if build.dir != "" {
		build.opts.WorkspaceDir = build.dir
	}
	app, err := NewApp(withAdapters(build.opts))
	if err != nil {
		t.Fatal(err)
	}
	// A save is written in the background and the tabs read as saved at once,
	// so a test can end while the file is still being written, and its
	// temporary folder then cannot be removed ("directory not empty").
	t.Cleanup(app.WaitWrites)
	app.w, app.h = winW, winH
	app.Step(core.Input{}) // first frame: lay everything out once
	return app
}

// ---- pollers ---------------------------------------------------------------

// until steps the app until cond holds.
func until(t *testing.T, app *App, what string, cond func() bool) {
	t.Helper()
	for range 400 {
		app.Step(core.Input{})
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// statusSoon steps the app until its status holds want.
func statusSoon(t *testing.T, app *App, want string) {
	t.Helper()
	until(t, app, fmt.Sprintf("a status holding %q", want), func() bool {
		return strings.Contains(app.notices.Status(), want)
	})
}

// waitForResponse steps the app until tab has a response.
func waitForResponse(t *testing.T, app *App, tab *reqTab) {
	t.Helper()
	until(t, app, "a response", func() bool { return tab.Resp != nil })
}

// ---- dumps and the element tree -------------------------------------------

// assertDumpPinned lays root out in a window of width by height and
// compares its dump with testdata/name.dump.
func assertDumpPinned(t *testing.T, app *App, root Node, name string, width, height int) {
	t.Helper()
	Layout(app.ctx, root, width, height)
	want, err := os.ReadFile(filepath.Join("testdata", name+".dump"))
	if err != nil {
		t.Fatal(err)
	}
	// The settings page's tooltip names the settings file, in a temp dir that
	// differs every run.
	got := strings.ReplaceAll(Dump(root), app.SettingsPath(), "/config/settings.json")
	if got != string(want) {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// findWidgets is every node under root whose behavior has the Go type
// kind, in tree order.
func findWidgets(root Node, kind string) []Node {
	var found []Node
	if fmt.Sprintf("%T", root.Behavior()) == kind {
		found = append(found, root)
	}
	for _, child := range root.Children() {
		found = append(found, findWidgets(child, kind)...)
	}
	return found
}

// findWidget is the first node findWidgets finds, nil when there is none.
func findWidget(root Node, kind string) Node {
	if found := findWidgets(root, kind); len(found) > 0 {
		return found[0]
	}
	return nil
}

// behaviorStates prints the behavior of every node under root whose
// behavior has the Go type kind, in order.
func behaviorStates(root Node, kind string) []string {
	var states []string
	for _, node := range findWidgets(root, kind) {
		states = append(states, fmt.Sprintf("%+v", node.Behavior()))
	}
	return states
}

// lineCenter is the center of the rect on a layout dump line.
func lineCenter(t *testing.T, line string) (x, y int) {
	t.Helper()
	match := rectPattern.FindStringSubmatch(line)
	if match == nil {
		t.Fatalf("no rectangle on %q", line)
	}
	var left, top, width, height int
	fmt.Sscan(match[1], &left)
	fmt.Sscan(match[2], &top)
	fmt.Sscan(match[3], &width)
	fmt.Sscan(match[4], &height)
	return left + width/2, top + height/2
}

// dumpedLines are the lines of the window's layout dump that contain needle.
func dumpedLines(app *App, needle string) []string {
	var lines []string
	for _, line := range strings.Split(layoutDump(app), "\n") {
		if strings.Contains(line, needle) {
			lines = append(lines, line)
		}
	}
	return lines
}

// textCenter is the middle of the first element of the window that shows
// text exactly.
func textCenter(t *testing.T, app *App, text string) (x, y int) {
	t.Helper()
	lines := dumpedLines(app, fmt.Sprintf("text=%q", text))
	if len(lines) == 0 {
		t.Fatalf("no element with text %q", text)
	}
	return lineCenter(t, lines[0])
}

// clickDumped clicks the middle of the nth layout dump line that contains needle.
func clickDumped(t *testing.T, app *App, needle string, nth int) {
	t.Helper()
	app.Step(core.Input{X: 5000, Y: 5000})
	lines := dumpedLines(app, needle)
	if nth >= len(lines) {
		t.Fatalf("no element with %s (#%d) in:\n%s", needle, nth, layoutDump(app))
	}
	x, y := lineCenter(t, lines[nth])
	click(app, x, y)
}

// ---- toasts and menus ------------------------------------------------------

// toastsWith are the toasts on the screen whose text contains part.
func toastsWith(app *App, part string) []widget.Toast {
	var found []widget.Toast
	for _, toast := range app.over.Toasts.Stack() {
		if strings.Contains(toast.Text, part) {
			found = append(found, toast)
		}
	}
	return found
}

// toastSays is true when a toast on the screen contains text.
func toastSays(app *App, text string) bool { return len(toastsWith(app, text)) > 0 }

// toastWith is true when a toast of the kind isErr contains text.
func toastWith(app *App, text string, isErr bool) bool {
	for _, toast := range toastsWith(app, text) {
		if (toast.Kind == widget.ToastError) == isErr {
			return true
		}
	}
	return false
}

// errorToasts are the texts of the error toasts that contain part.
func errorToasts(app *App, part string) []string {
	var found []string
	for _, toast := range toastsWith(app, part) {
		if toast.Kind == widget.ToastError {
			found = append(found, toast.Text)
		}
	}
	return found
}

// toastTexts is the text of every toast on the screen, one per line.
func toastTexts(app *App) string {
	var texts []string
	for _, toast := range app.over.Toasts.Stack() {
		texts = append(texts, toast.Text)
	}
	return strings.Join(texts, "\n")
}

// itemByLabel is the open menu's item called label.
func itemByLabel(t *testing.T, app *App, label string) *widget.MenuItem {
	t.Helper()
	items := widget.PopupMenu(app.ctx).Items
	for i := range items {
		if items[i].Label == label {
			return &items[i]
		}
	}
	t.Fatalf("no menu item %q in %v", label, menuLabels(app))
	return nil
}

// ---- input and files -------------------------------------------------------

// combo is a key press with the modifiers held.
func combo(key ebiten.Key, ctrl, shift, alt bool) core.Input {
	return core.Input{Keys: []ebiten.Key{key}, Ctrl: ctrl, Shift: shift, Alt: alt}
}

// fileText is the text of the file at path, empty when it cannot be read.
func fileText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// ---- sessions --------------------------------------------------------------

// fakeSession is an open session. It records what it is asked to send
// and to request, answers a request with result and err, and, when
// release is set, holds every send until release is closed.
type fakeSession struct {
	mu      sync.Mutex
	sent    []string
	asked   []string
	result  string
	err     error
	release chan struct{}
	closed  bool
}

func (session *fakeSession) Events() <-chan request.Event { return nil }

func (session *fakeSession) Close(context.Context) error {
	session.closed = true
	return nil
}

func (session *fakeSession) Ping(ctx context.Context) error { return session.Send(ctx, "ping") }

func (session *fakeSession) Send(_ context.Context, text string) error {
	if session.release != nil {
		<-session.release
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	session.sent = append(session.sent, text)
	return nil
}

func (session *fakeSession) Request(_ context.Context, text string) (string, error) {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.asked = append(session.asked, text)
	return session.result, session.err
}

// log is what was sent, in order.
func (session *fakeSession) log() []string {
	session.mu.Lock()
	defer session.mu.Unlock()
	return append([]string(nil), session.sent...)
}

// These tests drive the whole window without a GPU: step() builds the
// view, lays it out and delivers synthetic input exactly as Update does.
// Coordinates follow the layout of a 1280x800 window: sidebar 260 wide,
// header 44, request tabs 42, URL bar 56 (centre y 112), tree rows 28.
const (
	winW, winH = 1280, 800
	urlBarY    = headerH + tabsH + urlH/2
	sendX      = winW - 12 - 40
	menuBtnX   = winW - 12 - headerH/2
	menuBtnY   = headerH / 2
)

func treeRowY(index int) int { return headerH + index*core.RowH + core.RowH/2 }

// withAdapters gives opts the real history and font adapters, as main does.
func withAdapters(opts Options) Options {
	opts.HostOS = cmp.Or(opts.HostOS, "linux") // the same shortcut text on every machine
	opts.RetryDelay = cmp.Or(opts.RetryDelay, 10*time.Millisecond)
	opts.History = func(dir string) history.History { return historyfile.Dir(dir) }
	opts.Fonts = fonts.System{}
	opts.Protocols = protocol.Configured
	opts.GRPCMethods = protocol.ListGRPCMethods
	opts.GRPCSchema = protocol.GRPCInputSchema
	opts.Curl = protocol.Curl
	opts.Store = cmp.Or[collection.Store](opts.Store, collectionstore.Files{})
	opts.States = statefile.Files{}
	opts.SettingsFiles = settingsfile.Files{}
	return opts
}

// click is a press followed by a release at the same place.
func click(app *App, posX, posY int) {
	app.Step(core.Input{X: posX, Y: posY, Click: true, Held: true})
	app.Step(core.Input{X: posX, Y: posY})
}

// jsonServer answers every request with a small JSON body.
func jsonServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
		io.WriteString(writer, `{"ok":true}`)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// pointTabAt makes the open request target url, as if typed into the field.
func (a *App) pointTabAt(tab *reqTab, url string) {
	tab.Req.URL = url
	*a.urlbar.Field(tab.Tab) = widget.NewTextField(url)
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

func (a *App) typeText(text string) { a.Step(core.Input{Typed: []rune(text)}) }

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
	a := testApp(t)
	a.cur().paneTab = bodyTabIdx
	a.Step(core.Input{})
	return a
}

// tabX is the left edge and width of tab i in the request-tab strip,
// from the layout the view uses: 14px padding, a fixed badge slot, 8px
// gaps, the name, a fixed slot for the unsaved dot and a 24px close
// button. Nothing but the name changes a tab's width.
func (a *App) tabX(index int) (left, width int) {
	x := 260
	for j, t := range a.openTabs() {
		wd := 14 + sizes.TabBadgeW + 8 + a.ctx.Width(a.ctx.F.UI, t.Req.Name) + 8 + sizes.TabDirtyW + 24
		if j == index {
			return x, wd
		}
		x += wd
	}
	return x, 0
}

const tabStripY = headerH + tabsH/2

func twoTabs(t *testing.T) *App {
	a := testApp(t)
	click(a, 100, treeRowY(4)) // opens "Repository Info" as the second tab
	if len(a.tabs.Tabs()) != 2 {
		t.Fatal("test setup: want two tabs")
	}
	return a
}

// pathServer serves /fast at once and /slow only when released (or when
// the client gives up), so tests control which request finishes when.
func pathServer(t *testing.T) (srv *httptest.Server, release func()) {
	t.Helper()
	gate := make(chan struct{})
	var once sync.Once
	srv = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) {
		switch call.URL.Path {
		case "/slow":
			select {
			case <-gate:
				io.WriteString(writer, "slow")
			case <-call.Context().Done():
			}
		case "/redirect":
			http.Redirect(writer, call, "/fast", http.StatusFound)
		default:
			io.WriteString(writer, "fast")
		}
	}))
	release = func() { once.Do(func() { close(gate) }) }
	t.Cleanup(func() { release(); srv.Close() })
	return srv, release
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
	a := testApp(t)
	a.cur().paneTab = settingsTabIdx
	a.Step(core.Input{})
	return a
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
	a := testApp(t)
	a.cur().paneTab = authTabIdx
	a.Step(core.Input{})
	return a
}

func sendAndWait(t *testing.T, app *App, url string) {
	t.Helper()
	tab := app.cur()
	app.pointTabAt(tab, url)
	tab.Resp = nil
	app.send(tab)
	waitForResponse(t, app, tab)
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
	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, call *http.Request) { io.WriteString(writer, body) }))
	t.Cleanup(srv.Close)
	a := testApp(t)
	sendAndWait(t, a, srv.URL)
	return a
}

func drag(app *App, fromX, toX, posY int) {
	app.Step(core.Input{X: fromX, Y: posY, Click: true, Held: true})
	app.Step(core.Input{X: toX, Y: posY, Held: true})
	app.Step(core.Input{X: toX, Y: posY}) // release
}

type ran struct{ log []string }

func (r *ran) item(label string) widget.MenuItem {
	return widget.MenuItem{Label: label, Run: func() { r.log = append(r.log, label) }}
}

func rightClick(app *App, posX, posY int) { app.Step(core.Input{X: posX, Y: posY, RightClick: true}) }

// Sample tree rows: 0 root, 1 User, 2 Repository, 3 Search Repos,
// 4 Repository Info, 5 Repository Tags, 6 Realtime.
const (
	rowRoot, rowUser, rowRepo, rowSearch, rowInfo, rowTags, rowRealtime = 0, 1, 2, 3, 4, 5, 6
)

func menuLabels(app *App) []string {
	var out []string
	for _, it := range widget.PopupMenu(app.ctx).Items {
		out = append(out, it.Label)
	}
	return out
}

// pickMenu clicks the menu item with the given label.
func pickMenu(t *testing.T, app *App, label string) {
	t.Helper()
	m := widget.PopupMenu(app.ctx)
	if m == nil {
		t.Fatal("no menu is open")
	}
	y := m.Y + widget.MenuPad
	for _, it := range m.Items {
		h := widget.MenuRowHeight(app.ctx)
		if it.Separator {
			h = widget.MenuSepH
		}
		if it.Label == label {
			click(app, m.X+100, y+h/2)
			return
		}
		y += h
	}
	t.Fatalf("no menu item %q in %v", label, menuLabels(app))
}

func findReq(app *App, name string) *request.Request {
	for _, r := range collection.Rows(&app.col().Folder, allOpen(app)) {
		if r.Req != nil && r.Req.Name == name {
			return r.Req
		}
	}
	return nil
}

func allOpen(app *App) map[*request.Folder]bool {
	open := map[*request.Folder]bool{}
	var walk func(*request.Folder)
	walk = func(folder *request.Folder) {
		open[folder] = true
		for _, s := range folder.Folders {
			walk(s)
		}
	}
	walk(&app.col().Folder)
	return open
}

// Params table at 1280x800: header cells at y=244 (the action cell at
// x=310), rows at y=272, 300, 328 and the placeholder at 356.
const (
	paramsHeaderY = 236
	checkAllX     = 310
	row1Y         = 292
	row2Y         = 320
	bulkBtnX      = 715
)

// The settings dialog at 1280x800: 760x560 at (260, 120), a 44px title
// bar, a 170px list of pages on the left and the page's content from
// x=446, y=180. Appearance sections are 62px apart: Theme, Text size,
// Interface scale, Corners, Gradients, Request / response, Contrast.
const ()

// openSettingsPage opens the settings dialog on the page with the given id.
func openSettingsPage(t *testing.T, id string) (*App, string) {
	a, path := openSettingsMenu(t)
	a.over.Settings.Shell.Cat = id
	a.Step(core.Input{})
	return a, path
}

func openSettingsMenu(t *testing.T) (*App, string) {
	a := testApp(t, withSettings(defaultLayout))
	path := a.SettingsPath()
	click(a, menuBtnX, menuBtnY)
	if !a.over.Settings.Open() {
		t.Fatal("test setup")
	}
	return a, path
}

// tipAt hovers (x, y) long enough and draws; it returns the tooltip shown.
func tipAt(app *App, posX, posY int) string {
	app.Step(core.Input{X: posX, Y: posY})
	img := ebiten.NewImage(winW, winH)
	app.Draw(img) // the pointer comes to rest on the element
	app.ctx.Tick += 100
	app.dirty = true
	app.Draw(img)
	if app.tips.Shown() {
		return app.ctx.TipText
	}
	return ""
}

// A response with nested blocks: one match deep inside one item, one at
// the top, and plenty of siblings that must disappear while searching.
const compactBody = `{"total":3,"items":[` +
	`{"id":1,"name":"alpha","tags":["red","green","blue","yellow","orange","purple","brown","black"],"owner":{"login":"ann","site":"https://a.example.org/very/long/path/to/ann"}},` +
	`{"id":2,"name":"beta","tags":["red","green","blue","yellow","orange","purple","brown","black"],"owner":{"login":"bob","site":"https://b.example.org/very/long/path/to/bob"}},` +
	`{"id":3,"name":"gamma","tags":["cyan","magenta","teal","olive","navy","maroon","lime","aqua"],"owner":{"login":"cy","site":"https://c.example.org/very/long/path/to/cy"}}]}`

func viewText(a *App, tab *reqTab) string {
	view := a.resp.Code(tab.Tab)
	var sb strings.Builder
	for _, i := range view.Visible() {
		for _, sp := range view.Lines[i].Spans {
			sb.WriteString(sp.Text)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// find replaces the search text, as typing would.
func (a *App) find(tab *reqTab, text string) {
	a.resp.Field(tab.Tab).Focused = true
	a.resp.Find(tab.Tab, text)
}

// openSession puts tab in the open state with session, as a finished
// connect would; an open tab is closed first.
func openSession(tab *reqTab, session request.Session) {
	if tab.Session() != nil {
		tab.Disconnect()
		tab.Closed()
	}
	tab.Begin()
	tab.Opened(session)
}

// setSending puts tab in the sending or the idle state, as a send or its
// answer would.
func setSending(app *App, tab *reqTab, sending bool) {
	switch {
	case sending && !app.sends.Sending(tab.Req):
		app.sends.Begin(tab.Req, func() {})
	case !sending && app.sends.Cancel(tab.Req):
	}
}

// saveBodyAs saves the response body of tab to path the way the menu and the
// file chooser do: the Save body item opens the chooser and its answer is the
// path.
func saveBodyAs(t *testing.T, a *App, tab *reqTab, path string) {
	t.Helper()
	a.resp.Refresh(tab.Tab)
	a.Step(core.Input{})
	asked, release := make(chan struct{}), make(chan struct{})
	a.files = fakeDialog{choose: func(context.Context, string, filedialog.Mode, string) (string, error) {
		close(asked)
		<-release // answers once the frames of the click are over, so the answer is drained below
		return path, nil
	}}
	rightClick(a, findFieldX, findBarY)
	pickMenu(t, a, "Save body...")
	select {
	case <-asked:
	case <-time.After(2 * time.Second):
		t.Fatal("the Save body item opened no file chooser")
	}
	close(release)
	for range 400 { // the answer is applied on this goroutine, as the frame loop does
		if a.drain() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("the file chooser's answer never arrived")
}

// openBodyInBrowser opens the response body of tab in the browser the way the
// menu does.
func openBodyInBrowser(t *testing.T, a *App, tab *reqTab) {
	t.Helper()
	a.resp.Refresh(tab.Tab)
	a.Step(core.Input{})
	rightClick(a, findFieldX, findBarY)
	pickMenu(t, a, "Open in browser")
}
