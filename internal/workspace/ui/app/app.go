package ui

import (
	"context"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/collection"
	cport "github.com/gascoigne-ai/gascoigne-workshop/internal/collection/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/environment/controller/envs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model/assert"
	reqport "github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/ui/calltrace"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/controller/viewer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/model/textfmt"
	rport "github.com/gascoigne-ai/gascoigne-workshop/internal/response/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/ui/sentlog"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/appearance"
	sport "github.com/gascoigne-ai/gascoigne-workshop/internal/settings/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/ui/colorpage"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/ui/settingsui"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/ui/shortcutpage"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/autosave"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/serial"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/toasts"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/port"
	"net/http"
	"net/http/cookiejar"
	"slices"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/controller/editor"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/model/state"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

const (
	version = "v0.1.0"

	// defaultTimeoutSec is used when neither the request nor the settings name one.
	defaultTimeoutSec = 60
)

// Cycled by clicking the protocol and method chips of the URL bar.
var (
	protocolOrder = []string{"rest", "websocket", "mcp", "sse", "grpc"}
	protocolLabel = map[string]string{"rest": "REST", "websocket": "WS", "mcp": "MCP", "sse": "SSE", "grpc": "gRPC"}
	methodOrder   = []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
)

// next returns the entry after cur in order, wrapping; the first entry
// when cur is not listed.
func next(order []string, cur string) string {
	for i, s := range order {
		if s == cur {
			return order[(i+1)%len(order)]
		}
	}
	return order[0]
}

// reqTab is one open request with its editor and response state.
type reqTab struct {
	*tabs.Tab
	url                   view.TextField
	paneTab               int
	respTab               int
	view                  view.CodeView              // response body
	bodyView              view.CodeView              // request body, read-only
	picture               *ebiten.Image              // the response body when it is an image
	pictureInfo           string                     // "PNG 640x480, 12.5KB"
	tests                 view.TextArea              // the assertions editor
	testResults           []assert.Result            // what the assertions found in the shown response
	gqlVars               view.TextArea              // the variables of a GraphQL body
	file                  view.TextField             // the path of a file body
	body                  view.TextArea              // request body editor
	reqScroll, respScroll view.ScrollState           // scroll of the table and note panes
	bodyTypesX            view.ScrollState           // sideways scroll of the body type chips
	reqTabsX, respTabsX   view.ScrollState           // sideways scroll of the two pane tab strips
	timeout               view.TextField             // the request's timeout seconds, as typed
	cancel                context.CancelFunc         // stops the running request
	auth                  map[string]*view.TextField // Auth pane inputs by field id, created on first use
	showSecrets           bool                       // show masked credential fields
	look                  viewer.Model               // the response mode, the find bar and its active match
	compactAll            []view.Line                // the unfiltered compact lines, kept while a search filters the view
	find                  view.TextField             // the search text
	wsCancel              context.CancelFunc         // ends the timeout of the connect in progress
	wsView                view.CodeView              // the message log
	msg                   view.TextField             // the message being composed
	grpcStream            bool                       // a gRPC tab streams instead of making single calls
	schema                *model.Schema              // what a GraphQL introspection answered
}

// makeTab is newTab with the response view the settings ask for.
func (a *App) makeTab(tab *tabs.Tab) *reqTab {
	t := newReqTab(tab)
	t.look = viewer.New(a.settings.ResponseView)
	t.Earlier = a.loadHistory(tab.Req)
	return t
}

// updateTabs applies msg to the tab list, then does what the change means
// for the window: an inline edit in a closed tab ends, closed tabs lose
// their widgets and new ones get theirs.
func (a *App) updateTabs(msg tabs.Msg) {
	next := tabs.Update(a.list, msg)
	for tab, widgets := range a.widgets {
		if slices.Contains(next.Tabs, tab) {
			continue
		}
		if a.edit != nil && a.edit.tab == widgets {
			a.cancelEdit()
		}
		delete(a.widgets, tab)
	}
	for _, tab := range next.Tabs {
		if a.widgets[tab] == nil {
			a.widgets[tab] = a.makeTab(tab)
		}
	}
	a.list = next
}

// openTabs are the widget states of the open tabs, in the order of the strip.
func (a *App) openTabs() []*reqTab {
	open := make([]*reqTab, len(a.list.Tabs))
	for i, tab := range a.list.Tabs {
		open[i] = a.widgets[tab]
	}
	return open
}

func newTab(req *model.Request) *reqTab { return newReqTab(tabs.New(req)) }

func newReqTab(tab *tabs.Tab) *reqTab {
	req := tab.Req
	t := &reqTab{Tab: tab, url: view.NewTextField(req.URL)}
	t.look.Match = -1
	t.wsView.Follow = true
	t.wsCancel = func() {}
	t.timeout = view.NewTextField(timeoutText(req.TimeoutSec))
	t.body = view.NewTextArea(req.Body)
	t.tests = view.NewTextArea(req.Tests)
	t.file = view.NewTextField(req.Body)
	t.gqlVars = view.NewTextArea(req.GraphQLVars)
	return t
}

type sendResult struct {
	tab  *reqTab
	seq  int // which send of the tab this answers
	resp model.Response
}

// App is the whole window. It implements ebiten.Game.
type App struct {
	ctx         *core.Ctx
	reg         reqport.Registry
	protocols   func(reqport.ProtocolConfig) reqport.Registry // builds the registry; nil has no protocols
	grpcMethods reqport.GRPCMethods
	curl        func(model.Request) string
	col         model.Collection
	expanded    map[*model.Folder]bool
	list        tabs.List             // the open tabs, the active one and the one being dragged
	widgets     map[*tabs.Tab]*reqTab // the widget state of each open tab
	bus         *bus.Bus              // what goroutines report back, drained once per frame
	status      string
	w, h        int
	shot        screenshot

	settings           settings.Settings
	settingsPath       string        // "" keeps settings in memory only
	menu               bool          // settings menu is open
	edit               *cellEdit     // the table cell being edited, if any
	dragRow            *rowSelection // the table row being dragged by its grip, nil when none
	ctxMenu            *contextMenu  // the open context menu, if any
	bulk               *bulkEdit     // the table in bulk-edit mode, if any
	actionIDs          []string      // the ids of the shortcut actions, for completing settings.json
	store              cport.Store
	states             port.StateStore
	files              port.FileDialog               // the desktop file chooser, replaced in tests
	cancelSettingsPick func()                        // takes back a file question the settings dialog asked
	settingsUnsaved    bool                          // settings were changed but not written (auto-save is off)
	prefs              settingsui.Dialog             // the settings dialog
	shortcutKeys       shortcutpage.Page             // the Shortcuts page of the settings dialog
	tabsScroll         view.ScrollState              // sideways scroll of the request tab strip
	dirty              bool                          // something changed since the last draw
	tps                int                           // the update rate last set
	lastIn             core.Input                    // the previous frame's input
	pacer              engine.Pacer                  // update rate from idle time
	colors             colorpage.Page                // the Colors page of the settings dialog
	colWidths          map[string]*view.ColumnWidths // the column sizes of each table, by name
	rowSel             *rowSelection                 // the table row the keyboard acts on
	selected           any                           // the tree node last clicked or moved to: a *Folder or *Request
	treeFocus          bool                          // the tree has the keyboard
	filter             view.TextField                // the sidebar filter
	sent               sentlog.Log                   // requests sent in this workspace, newest first
	calls              calltrace.Log                 // the call trace behind the Log button
	showSent           bool                          // the sidebar lists sent requests instead of the tree
	dragNode           any                           // the tree node being dragged, nil when none
	pressedFolder      *model.Folder                 // the folder row the button went down on, toggled on release there
	hoverFolder        *model.Folder                 // the closed folder a drag has been over
	hoverTicks         int                           // for how many frames
	dropTarget         any                           // the node the pointer is over while dragging
	dialog             *dialog                       // the open dialog, if any
	toasts             toasts.Stack
	now                func() time.Time               // the clock, replaced in tests
	histories          func(dir string) rport.History // nil keeps no history
	fonts              sport.FontStore
	settingFiles       sport.SettingsStore
	previews           rport.BodyPreview // shows a response body in the browser, replaced in tests
	sampler            core.Sampler
	treeScroll         view.ScrollState
	sidebarSplit       view.SplitState   // width of the collections sidebar
	paneSplit          view.SplitState   // request pane share of the request/response area
	statePath          string            // where the UI state is remembered
	varsFolder         *model.Folder     // the folder whose variables the Vars pane edits, nil for the environment's or the collection's
	varsCollection     bool              // the Vars pane edits the collection's variables although an environment is active
	title              string            // the window title last set
	jar                http.CookieJar    // cookies the server set, kept until the app closes
	env                envs.Selection    // the active environment of col
	workspace          string            // the folder the collection is saved to; "" for none yet
	defaultWorkspace   string            // where the first save goes when there is no workspace
	colSave            autosave.Debounce // when the collection was last written
	writes             serial.Queue      // the background writes in flight and waiting, by file
	keys               changeKeys        // the comparable forms of the collection and the state, recomputed only on frames where something could have changed
	stateSave          autosave.Debounce // when the remembered state was last written
	initial            state.Window      // window geometry of the last run
	winX, winY         int               // window position, read each frame
}

// Options are the startup switches of the command line.
type Options struct {
	ScreenshotPath   string                                        // save a PNG of the first settled frame and exit
	ScreenshotSet    []string                                      // key=value settings applied live before the screenshot
	SendOnStart      bool                                          // send the first request at startup
	SettingsPath     string                                        // settings file; "" keeps settings in memory
	StatePath        string                                        // remembered UI state; "" keeps it in memory
	ThemesDir        string                                        // folder of user theme files (*.json); "" for none
	ScanFonts        bool                                          // look for installed fonts in the background, to apply the saved font families
	Protocols        func(reqport.ProtocolConfig) reqport.Registry // builds the protocols; nil has none
	GRPCMethods      reqport.GRPCMethods                           // lists gRPC methods by reflection; nil cannot
	Curl             func(model.Request) string                    // the cURL command of a request; nil copies nothing
	Store            cport.Store                                   // where the collection is saved; nil saves nothing
	States           port.StateStore                               // where the UI state is kept; nil keeps none
	Files            port.FileDialog                               // the desktop file chooser; nil asks for paths as text
	History          func(dir string) rport.History                // the history in a folder; nil keeps none
	Fonts            sport.FontStore                               // installed fonts; nil has none
	SettingsFiles    sport.SettingsStore                           // the settings file; nil reads the defaults and cannot save
	Previews         rport.BodyPreview                             // shows a response body in the browser; nil has none
	Clock            port.Clock                                    // the time source; nil uses the wall clock
	OpenMenu         bool                                          // start with the settings menu open
	WorkspaceDir     string                                        // collection folder; "" uses the remembered one, else the built-in sample
	DefaultWorkspace string                                        // where saving goes when there is no workspace yet
}

// NewApp builds the window state around the sample collection.
func NewApp(opts Options) (*App, error) {
	settingFiles := opts.SettingsFiles
	if settingFiles == nil {
		settingFiles = noSettingFiles{}
	}
	st, loadErr := settings.Default(), error(nil)
	if opts.SettingsPath != "" {
		st, loadErr = settingFiles.Load(opts.SettingsPath)
	}
	themes, themeProblems := appearance.LoadUserThemes(opts.ThemesDir)
	appearance.SetUserThemes(themes)
	th := appearance.ThemeWith(st)
	ctx, err := core.NewCtx(th)
	if err != nil {
		return nil, err
	}
	a := &App{
		ctx:              ctx,
		col:              collection.Sample(),
		defaultWorkspace: opts.DefaultWorkspace,
		expanded:         map[*model.Folder]bool{},
		widgets:          map[*tabs.Tab]*reqTab{},
		bus:              bus.New(),
		status:           "Ready",
		shot:             screenshot{path: opts.ScreenshotPath, sendOnStart: opts.SendOnStart, set: opts.ScreenshotSet},

		settings:     st,
		settingFiles: settingFiles,
		colSave:      autosave.New(saveDebounce, ""),
		settingsPath: opts.SettingsPath,
		statePath:    opts.StatePath,
		menu:         opts.OpenMenu,
	}
	if loadErr != nil {
		a.status = loadErr.Error()
	}
	a.prefs.Stamp(a, a.settingsPath)
	a.jar, _ = cookiejar.New(nil)
	a.protocols, a.grpcMethods, a.curl = opts.Protocols, opts.GRPCMethods, opts.Curl
	a.reg = a.newRegistry()
	ctx.Clipboard = statusClipboard{inner: ctx.Clipboard, report: a.clipboardNote}
	a.list.Drag = -1
	a.dirty = true
	a.now = time.Now // the fallback when no Clock port is given
	if opts.Clock != nil {
		a.now = opts.Clock.Now
	}
	ctx.Now = func() time.Time { return a.now() } // the caret blinks by the app clock
	a.store, a.states = opts.Store, opts.States
	if a.states == nil {
		a.states = noStates{}
	}
	if a.store == nil {
		a.store = noStore{}
	}
	a.files, a.previews, a.histories, a.fonts = opts.Files, opts.Previews, opts.History, opts.Fonts
	if a.fonts == nil {
		a.fonts = noFonts{}
	}
	if a.files == nil {
		a.files = noFileDialog{}
	}
	if a.previews == nil {
		a.previews = noBrowser{}
	}
	for _, s := range shortcuts {
		a.actionIDs = append(a.actionIDs, s.id)
	}
	if opts.ScanFonts {
		a.prefs.ScanFonts(a)
	}
	a.openWorkspace(opts.WorkspaceDir)
	stored, stateErr := a.states.Load(opts.StatePath)
	a.initial = stored.Window
	a.restore(stored)
	a.sent.Load(a)
	a.status = stateNote(a.status, stateErr)
	if len(themeProblems) > 0 {
		a.status = stateNote(a.status, fmt.Errorf("%s", themeProblems[0]))
	}
	a.stateSave = autosave.New(stateDebounce, stateKey(a.snapshot()))
	return a, nil
}

func (a *App) cur() *reqTab { return a.widgets[a.list.Tabs[a.list.Active]] }

// openRequest focuses the tab of req, opening one if needed.
func (a *App) openRequest(req *model.Request) {
	a.selected = req
	a.updateTabs(tabs.Open{Req: req})
}

// send runs t's request off the UI goroutine; the result comes back
// through the bus and is applied in drain.
func (a *App) send(t *reqTab) {
	if isSocket(t) {
		a.toggleConnection(t)
		return
	}
	a.dispatch(t, t.Req)
}

// dispatch sends req on behalf of tab t: the request of the tab, or a
// helper one such as a GraphQL schema query that uses the tab's URL and
// credentials.
func (a *App) dispatch(t *reqTab, source *model.Request) {
	seq, started := t.BeginSend()
	if !started {
		return
	}
	a.status = "Sending " + source.URL
	req := source.Expanded(a.varsFor(t.Req))
	a.calls.Sent(a.now(), req)
	ctx, cancel := context.WithTimeout(context.Background(), a.timeoutFor(req))
	t.cancel = cancel
	reg := a.reg // the command must not read App fields the UI goroutine replaces
	a.bus.Run(ctx, func(ctx context.Context) bus.Msg {
		defer cancel()
		resp, err := reg.Send(ctx, req)
		if err != nil {
			resp = model.Response{Err: err.Error()}
		}
		return sendResult{tab: t, seq: seq, resp: resp}
	})
}

// cancelSend stops the running request of t. Its late result, which
// arrives after this, belongs to an older send and is dropped.
func (a *App) cancelSend(t *reqTab) {
	if !t.Cancel() {
		return
	}
	t.cancel()
	t.view.SetLines(nil)
	a.status = model.ErrCancelled
}

// timeoutFor is how long a request may run: its own setting, else the
// default from the settings, else a minute.
func (a *App) timeoutFor(req model.Request) time.Duration {
	sec := req.TimeoutSec
	if sec <= 0 {
		sec = a.settings.TimeoutSec
	}
	if sec <= 0 {
		sec = defaultTimeoutSec
	}
	return time.Duration(sec) * time.Second
}

// timeoutText is a timeout as shown in its field; 0 (the default) is blank.
func timeoutText(sec int) string {
	if sec <= 0 {
		return ""
	}
	return strconv.Itoa(sec)
}

// drain applies what other goroutines reported and reports whether there was
// anything.
func (a *App) drain() bool { return a.bus.Drain(a.handle) }

// handle applies one message from the bus.
func (a *App) handle(msg bus.Msg) {
	switch msg := msg.(type) {
	case sendResult:
		a.apply(msg)
	case wsEvent:
		a.applyWS(msg)
	case func(): // something a goroutine wants done on the UI goroutine, such as a chosen file
		msg()
	}
}

func (a *App) apply(r sendResult) {
	t := r.tab
	if !t.Accept(r.seq) {
		return // the answer to a send that was cancelled or replaced
	}
	t.Remember(t.Resp, a.now())
	t.Resp = &r.resp
	a.saveHistory(t)
	if path := a.requestPath(t.Req); path != "" {
		a.sent.Record(a, a.now(), path, t.Req, r.resp.Status)
	}
	a.calls.Answer(a.now(), t.Req, &r.resp)
	a.rememberSchema(t, &r.resp)
	a.refreshView(t)
	if r.resp.Err != "" {
		a.status = "Request failed"
		a.runTests(t)
		return
	}
	a.status = fmt.Sprintf("%s %s: %d %s in %s", t.Req.Method, t.Req.URL, r.resp.Status, r.resp.StatusText, textfmt.Duration(r.resp.Duration))
	a.runTests(t)
}

// cursorShapes maps the cursor widgets ask for to the window system's.
var cursorShapes = map[core.Cursor]ebiten.CursorShapeType{
	core.CursorDefault:    ebiten.CursorShapeDefault,
	core.CursorResizeH:    ebiten.CursorShapeEWResize,
	core.CursorResizeV:    ebiten.CursorShapeNSResize,
	core.CursorMove:       ebiten.CursorShapeMove,
	core.CursorResizeNWSE: ebiten.CursorShapeNWSEResize,
	core.CursorResizeNESW: ebiten.CursorShapeNESWResize,
}

// Layout implements ebiten.Game. The UI is laid out in logical pixels
// (w by h); the screen is that size times the monitor's scale factor so
// text and lines stay sharp on HiDPI displays.
func (a *App) Layout(w, h int) (int, int) {
	lw, lh, scale, sw, sh := frameSize(w, h, ebiten.Monitor().DeviceScaleFactor(), a.settings.UIScale())
	a.setSize(lw, lh, scale)
	return sw, sh
}

// Update implements ebiten.Game: it samples input and runs one frame.
func (a *App) Update() error {
	a.winX, a.winY = ebiten.WindowPosition()
	a.step(a.sampler.Read().Scaled(a.ctx.Scale))
	if title := a.Title(); title != a.title {
		a.title = title
		ebiten.SetWindowTitle(title)
	}
	ebiten.SetCursorShape(cursorShapes[a.ctx.Cursor])
	if tps := a.pacer.TPS(); tps != a.tps {
		ebiten.SetTPS(tps)
		a.tps = tps
	}
	return a.shotUpdate()
}

// step runs one frame with the given input: it applies finished sends,
// builds the view, lays it out and lets it react. Handlers in the view
// change App state. Tests drive the app through step without a window.
func (a *App) step(in core.Input) {
	a.ctx.Tick++
	a.ctx.In = in
	a.ctx.Cursor = core.CursorDefault
	if !in.Held && !in.MiddleHeld {
		a.updateTabs(tabs.DragEnd{})
		a.dragRow = nil
	}
	drained := a.drain()
	pruned := a.pruneToasts()
	active := engine.InputActive(a.lastIn, in)
	a.noteGeometry()
	if active || drained || pruned {
		a.markStale()
	}
	if restartsCaret(in) {
		a.ctx.RestartCaret()
	}
	a.lastIn = in
	if in.Held {
		a.dropTarget = nil // set again by the row under the pointer
	} else {
		a.finishDrag()
	}
	hadOverlay := a.overlayOpen()
	wasRecording := a.shortcutKeys.Recording() != ""
	menuWasOpen := a.ctxMenu != nil
	a.overlayKeys(in)
	if menuWasOpen {
		a.swallowMenuKeys()
	}
	if !hadOverlay && !a.overlayOpen() && !wasRecording {
		a.treeKeys(in)
	}
	if wasRecording {
		a.shortcutKeys.RecordKey(a, in)
	} else if !a.overlayOpen() {
		a.runShortcuts(in)
	}
	if in.Escape && !wasRecording { // one step per press: overlays, the settings menu, a cell edit, find, a running request
		a.escape()
	}
	if (in.Undo || in.Redo) && !a.overlayOpen() && !a.textFocused() {
		if p, t, ok := a.visibleTable(); ok {
			if in.Undo {
				a.undoRows(p, t)
			} else {
				a.redoRows(p, t)
			}
		}
	}
	root := a.view()
	view.Layout(a.ctx, root, a.w, a.h)
	view.Update(a.ctx, root)
	if a.edit != nil && !a.edit.field.Focused { // a click elsewhere ends the edit
		a.commitEdit()
	}
	if a.ctx.Tick%settingsui.PollTicks == 0 {
		a.prefs.Poll(a)
	}
	a.dragHover()
	a.showGrab()
	a.settleBulk()
	a.stateTick()
	a.workspaceTick()
	a.markFrame(active || drained || pruned || a.ctx.TipWaiting)
	a.pacer.NoteIdle(active || drained || pruned || a.ctx.TipWaiting)
}

// Draw implements ebiten.Game. The view is rebuilt so it shows the
// state the handlers just changed.
func (a *App) Draw(screen *ebiten.Image) {
	if !a.dirty && !a.alwaysDraw() {
		return // nothing changed: the screen keeps what it has and the GPU stays idle
	}
	root := a.view()
	view.Layout(a.ctx, root, a.w, a.h)
	view.Draw(a.ctx, root, screen)
	a.dirty = false
	a.shotDraw(screen)
}

func (a *App) clickTree(row collection.Row) {
	a.selectNode(row.Node())
	if row.Folder {
		a.expanded[row.Dir] = !a.expanded[row.Dir]
		return
	}
	a.openRequest(row.Req)
}

func (a *App) newRequest() { a.newRequestIn(a.selectedFolder()) }

func (a *App) cycleProtocol(t *reqTab) {
	a.disconnect(t) // a connection belongs to the protocol that opened it
	editRequest(t, editor.ProtocolChosen{Protocol: next(protocolOrder, t.Req.Protocol)})
}

func (a *App) cycleMethod(t *reqTab) {
	editRequest(t, editor.MethodChosen{Method: next(methodOrder, t.Req.Method)})
}

// urlEdited keeps the request and its Params table in step with the URL field.
func (a *App) urlEdited(t *reqTab) { editRequest(t, editor.URLTyped{Text: t.url.Text}) }

// editRequest applies an edit to the request of t, which is the collection's
// own, so the sidebar and the autosave see it at once.
func editRequest(t *reqTab, msg editor.Msg) { *t.Req = editor.Update(*t.Req, msg) }

// dirtyMark is shown on tabs whose request changed since it was opened.
const dirtyMark = "\u2022"

// dirty reports whether the request differs from its state when opened.
func dirty(t *reqTab) bool { return t.Dirty() }
