package ui

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/proto"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/state"
	"github.com/gascoigne-ai/gascoigne-workshop/ui/engine"
	"github.com/gascoigne-ai/gascoigne-workshop/ui/vml"
	"github.com/gascoigne-ai/gascoigne-workshop/ui/widget"
)

const (
	version = "v0.1.0"

	// defaultTimeoutSec is used when neither the request nor the settings name one.
	defaultTimeoutSec = 60
)

// Cycled by clicking the protocol and method chips of the URL bar.
var (
	protocolOrder = []string{"rest", "websocket", "mcp"}
	protocolLabel = map[string]string{"rest": "REST", "websocket": "WS", "mcp": "MCP"}
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
	req                   *model.Request
	url                   widget.TextField
	paneTab               int
	respTab               int
	resp                  *model.Response
	view                  widget.CodeView  // response body
	bodyView              widget.CodeView  // request body, read-only
	body                  *widget.TextArea // request body editor
	reqScroll, respScroll vml.ScrollState  // scroll of the table and note panes
	reqTabsX, respTabsX   vml.ScrollState  // sideways scroll of the two pane tab strips
	sending               bool
	hist                  map[string]*rowHistory       // undo history of each table
	saved                 model.Request                // the request as it was when opened or last saved
	timeout               widget.TextField             // the request's timeout seconds, as typed
	cancel                context.CancelFunc           // stops the running request
	auth                  map[string]*widget.TextField // Auth pane inputs by field id, created on first use
	showSecrets           bool                         // show masked credential fields
	raw                   bool                         // show the response body as received, not pretty-printed
	compact               bool                         // show JSON compactly, as a browser's viewer does
	compactAll            []widget.Line                // the unfiltered compact lines, kept while a search filters the view
	findOpen              bool                         // the find bar is showing
	find                  widget.TextField             // the search text
	matchIdx              int                          // the active match, -1 for none
	seq                   int                          // numbers the sends, so a late result of an old one is ignored
}

// responseViews are the values of the response view setting.
var responseViews = []struct {
	id, label    string
	raw, compact bool
}{
	{"compact", "Compact", false, true},
	{"pretty", "Pretty", false, false},
	{"raw", "Raw", true, false},
}

// viewFlags are the raw and compact flags of a response view setting;
// anything unknown is compact, the default.
func viewFlags(id string) (raw, compact bool) {
	for _, v := range responseViews {
		if v.id == id {
			return v.raw, v.compact
		}
	}
	return false, true
}

// makeTab is newTab with the response view the settings ask for.
func (a *App) makeTab(req *model.Request) *reqTab {
	t := newTab(req)
	t.raw, t.compact = viewFlags(a.settings.ResponseView)
	return t
}

func newTab(req *model.Request) *reqTab {
	t := &reqTab{req: req, url: widget.NewTextField(req.URL)}
	if req.BodyType == "" { // requests from before body types: infer one
		req.BodyType = model.BodyNone
		if req.Body != "" {
			req.BodyType = model.BodyText
		}
	}
	t.saved = req.Clone()
	t.matchIdx = -1
	t.timeout = widget.NewTextField(timeoutText(req.TimeoutSec))
	t.body = widget.NewTextArea(req.Body)
	return t
}

type sendResult struct {
	tab  *reqTab
	seq  int // which send of the tab this answers
	resp model.Response
}

// App is the whole window. It implements ebiten.Game.
type App struct {
	ctx      *widget.Ctx
	reg      proto.Registry
	col      model.Collection
	expanded map[*model.Folder]bool
	tabs     []*reqTab
	active   int
	results  chan sendResult
	status   string
	w, h     int
	shot     screenshot

	settings      settings.Settings
	settingsPath  string                             // "" keeps settings in memory only
	menu          bool                               // settings menu is open
	edit          *cellEdit                          // the table cell being edited, if any
	dragTab       int                                // index of the tab being dragged, -1 when none
	ctxMenu       *contextMenu                       // the open context menu, if any
	bulk          *bulkEdit                          // the table in bulk-edit mode, if any
	menuScroll    vml.ScrollState                    // scroll of the settings dialog's page
	settingsCat   string                             // the settings page shown
	tabsScroll    vml.ScrollState                    // sideways scroll of the request tab strip
	scaleSlider   vml.SliderState                    // drag of the interface scale slider
	dirty         bool                               // something changed since the last draw
	tps           int                                // the update rate last set
	lastIn        widget.Input                       // the previous frame's input
	pacer         engine.Pacer                       // update rate from idle time
	recording     string                             // the action whose key is being recorded, "" when none
	recordNotes   map[string]string                  // why a recorded key was refused, by action id
	colorInputs   map[widget.Token]*widget.TextField // hex fields of the settings menu
	colorNotes    map[widget.Token]string            // why a typed color was refused
	colWidths     map[string]*vml.ColumnWidths       // the column sizes of each table, by name
	rowSel        *rowSelection                      // the table row the keyboard acts on
	selected      any                                // the tree node last clicked or moved to: a *Folder or *Request
	treeFocus     bool                               // the tree has the keyboard
	filter        widget.TextField                   // the sidebar filter
	dragNode      any                                // the tree node being dragged, nil when none
	pressedFolder *model.Folder                      // the folder row the button went down on, toggled on release there
	dropTarget    any                                // the node the pointer is over while dragging
	dialog        *dialog                            // the open dialog, if any
	toasts        []toast
	now           func() time.Time // the clock, replaced in tests
	sampler       widget.Sampler
	treeScroll    vml.ScrollState
	sidebarSplit  vml.SplitState // width of the collections sidebar
	paneSplit     vml.SplitState // request pane share of the request/response area
	statePath     string         // where the UI state is remembered
	stateSeen     string         // the last snapshot seen, to notice changes
	stateSaved    string         // the last snapshot written
	stateAt       time.Time      // when the snapshot last changed
	initial       state.Window   // window geometry of the last run
	winX, winY    int            // window position, read each frame
}

// Options are the startup switches of the command line.
type Options struct {
	ScreenshotPath string // save a PNG of the first settled frame and exit
	SendOnStart    bool   // send the first request at startup
	SettingsPath   string // settings file; "" keeps settings in memory
	StatePath      string // remembered UI state; "" keeps it in memory
	ThemesDir      string // folder of user theme files (*.json); "" for none
	OpenMenu       bool   // start with the settings menu open
}

// NewApp builds the window state around the sample collection.
func NewApp(opts Options) (*App, error) {
	st, loadErr := settings.Default(), error(nil)
	if opts.SettingsPath != "" {
		st, loadErr = settings.Load(opts.SettingsPath)
	}
	themes, themeProblems := loadUserThemes(opts.ThemesDir)
	userThemes = themes
	th := ThemeWith(st)
	ctx, err := widget.NewCtx(th)
	if err != nil {
		return nil, err
	}
	a := &App{
		ctx:      ctx,
		reg:      proto.NewRegistry(),
		col:      SampleCollection(),
		expanded: map[*model.Folder]bool{},
		results:  make(chan sendResult, 8),
		status:   "Ready",
		shot:     screenshot{path: opts.ScreenshotPath, sendOnStart: opts.SendOnStart},

		settings:     st,
		settingsPath: opts.SettingsPath,
		statePath:    opts.StatePath,
		menu:         opts.OpenMenu,
	}
	if loadErr != nil {
		a.status = loadErr.Error()
	}
	ctx.Clipboard = statusClipboard{inner: ctx.Clipboard, report: a.clipboardNote}
	a.dragTab = -1
	a.dirty = true
	a.now = time.Now
	a.expanded[&a.col.Folder] = true
	a.expanded[a.col.Folders[1]] = true
	a.openRequest(a.col.Folders[1].Requests[0])
	stored, stateErr := state.Load(opts.StatePath)
	a.initial = stored.Window
	a.restore(stored)
	a.status = stateNote(a.status, stateErr)
	if len(themeProblems) > 0 {
		a.status = stateNote(a.status, fmt.Errorf("%s", themeProblems[0]))
	}
	a.stateSeen = stateKey(a.snapshot())
	a.stateSaved = a.stateSeen
	return a, nil
}

func (a *App) cur() *reqTab { return a.tabs[a.active] }

// openRequest focuses the tab of req, opening one if needed.
func (a *App) openRequest(req *model.Request) {
	a.selected = req
	for i, t := range a.tabs {
		if t.req == req {
			a.active = i
			return
		}
	}
	a.tabs = append(a.tabs, a.makeTab(req))
	a.active = len(a.tabs) - 1
}

// send runs t's request off the UI goroutine; the result comes back
// through a.results and is applied in Update.
func (a *App) send(t *reqTab) {
	if t.sending {
		return
	}
	t.sending = true
	t.seq++
	seq := t.seq
	a.status = "Sending " + t.req.URL
	req := t.req.Expanded(a.col.Vars)
	ctx, cancel := context.WithTimeout(context.Background(), a.timeoutFor(req))
	t.cancel = cancel
	go func() {
		defer cancel()
		resp, err := a.reg.Send(ctx, req)
		if err != nil {
			resp = model.Response{Err: err.Error()}
		}
		a.results <- sendResult{tab: t, seq: seq, resp: resp}
	}()
}

// cancelSend stops the running request of t. Its late result, which
// arrives after this, belongs to an older send and is dropped.
func (a *App) cancelSend(t *reqTab) {
	if !t.sending {
		return
	}
	t.cancel()
	t.sending = false
	t.seq++
	t.resp = &model.Response{Err: proto.ErrCancelled}
	t.view.SetLines(nil)
	a.status = proto.ErrCancelled
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

// drain applies the results of finished sends and reports whether there were any.
func (a *App) drain() (any bool) {
	for {
		select {
		case r := <-a.results:
			a.apply(r)
			any = true
		default:
			return any
		}
	}
}

func (a *App) apply(r sendResult) {
	t := r.tab
	if r.seq != t.seq {
		return // the answer to a send that was cancelled or replaced
	}
	t.sending = false
	t.resp = &r.resp
	a.refreshView(t)
	if r.resp.Err != "" {
		a.status = "Request failed"
		return
	}
	a.status = fmt.Sprintf("%s %s: %d %s in %s", t.req.Method, t.req.URL, r.resp.Status, r.resp.StatusText, formatDuration(r.resp.Duration))
}

// cursorShapes maps the cursor widgets ask for to the window system's.
var cursorShapes = map[widget.Cursor]ebiten.CursorShapeType{
	widget.CursorDefault: ebiten.CursorShapeDefault,
	widget.CursorResizeH: ebiten.CursorShapeEWResize,
	widget.CursorResizeV: ebiten.CursorShapeNSResize,
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
func (a *App) step(in widget.Input) {
	a.ctx.Tick++
	a.ctx.In = in
	a.ctx.Cursor = widget.CursorDefault
	if !in.Held && !in.MiddleHeld {
		a.dragTab = -1
	}
	drained := a.drain()
	pruned := a.pruneToasts()
	active := engine.InputActive(a.lastIn, in)
	a.lastIn = in
	if in.Held {
		a.dropTarget = nil // set again by the row under the pointer
	} else {
		a.finishDrag()
	}
	hadOverlay := a.overlayOpen()
	wasRecording := a.recording != ""
	a.overlayKeys(in)
	if !hadOverlay && !a.overlayOpen() && !wasRecording {
		a.treeKeys(in)
	}
	if wasRecording {
		a.recordKey(in)
	} else if !a.overlayOpen() {
		a.runShortcuts(in)
	}
	if in.Escape && !wasRecording { // one step per press: overlays, the settings menu, a cell edit, find, a running request
		switch {
		case a.ctxMenu != nil:
			a.closeMenu()
		case a.dialog != nil:
			a.closeDialog()
		case a.menu:
			a.menu = false
		case a.edit != nil:
			a.cancelEdit()
		case a.cur().findOpen:
			a.closeFind(a.cur())
		default:
			a.cancelSend(a.cur())
		}
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
	vml.Layout(a.ctx, root, a.w, a.h)
	vml.Update(a.ctx, root)
	if a.edit != nil && !a.edit.field.Focused { // a click elsewhere ends the edit
		a.commitEdit()
	}
	a.settleBulk()
	a.stateTick()
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
	vml.Layout(a.ctx, root, a.w, a.h)
	vml.Draw(a.ctx, root, screen)
	a.dirty = false
	a.shotDraw(screen)
}

func (a *App) clickTree(row treeRow) {
	a.selectNode(row.node())
	if row.Folder {
		a.expanded[row.Dir] = !a.expanded[row.Dir]
		return
	}
	a.openRequest(row.Req)
}

func (a *App) newRequest() { a.newRequestIn(a.selectedFolder()) }

func (a *App) cycleProtocol(t *reqTab) {
	t.req.Protocol = next(protocolOrder, t.req.Protocol)
	if t.req.Protocol == "rest" && t.req.Method == "" {
		t.req.Method = "GET"
	}
}

func (a *App) cycleMethod(t *reqTab) { t.req.Method = next(methodOrder, t.req.Method) }

// urlEdited keeps the request and its Params table in step with the URL field.
func (a *App) urlEdited(t *reqTab) {
	t.req.URL = t.url.Text
	t.req.Params = model.MergeParams(t.req.Params, t.req.URL)
}

// dirtyMark is shown on tabs whose request changed since it was opened.
const dirtyMark = "\u2022"

// dirty reports whether the request differs from its state when opened.
func dirty(t *reqTab) bool { return !reflect.DeepEqual(t.saved, t.req.Clone()) }

// dragOver moves the tab being dragged to where the pointer is over tab i.
func (a *App) dragOver(i int) {
	if a.dragTab >= 0 && a.dragTab != i {
		a.moveTab(a.dragTab, i)
	}
}

// moveTab moves a tab to a new place; the active tab stays the same tab.
func (a *App) moveTab(from, to int) {
	active := a.cur()
	tab := a.tabs[from]
	a.tabs = append(a.tabs[:from], a.tabs[from+1:]...)
	a.tabs = append(a.tabs[:to], append([]*reqTab{tab}, a.tabs[to:]...)...)
	for i, t := range a.tabs {
		if t == active {
			a.active = i
		}
	}
	a.dragTab = to
}
