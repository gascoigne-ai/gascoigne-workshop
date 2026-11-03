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
	"github.com/gascoigne-ai/gascoigne-workshop/vml"
	"github.com/gascoigne-ai/gascoigne-workshop/widget"
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
	sending               bool
	saved                 model.Request                // the request as it was when opened or last saved
	timeout               widget.TextField             // the request's timeout seconds, as typed
	cancel                context.CancelFunc           // stops the running request
	auth                  map[string]*widget.TextField // Auth pane inputs by field id, created on first use
	showSecrets           bool                         // show masked credential fields
	raw                   bool                         // show the response body as received, not pretty-printed
	findOpen              bool                         // the find bar is showing
	find                  widget.TextField             // the search text
	matchIdx              int                          // the active match, -1 for none
	seq                   int                          // numbers the sends, so a late result of an old one is ignored
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
	expanded map[string]bool
	tabs     []*reqTab
	active   int
	results  chan sendResult
	status   string
	w, h     int
	shot     screenshot

	settings     settings.Settings
	settingsPath string    // "" keeps settings in memory only
	menu         bool      // settings menu is open
	edit         *cellEdit // the table cell being edited, if any
	dragTab      int       // index of the tab being dragged, -1 when none
	sampler      widget.Sampler
	treeScroll   vml.ScrollState
	sidebarSplit vml.SplitState // width of the collections sidebar
	paneSplit    vml.SplitState // request pane share of the request/response area
}

// Options are the startup switches of the command line.
type Options struct {
	ScreenshotPath string // save a PNG of the first settled frame and exit
	SendOnStart    bool   // send the first request at startup
	SettingsPath   string // settings file; "" keeps settings in memory
	OpenMenu       bool   // start with the settings menu open
}

// NewApp builds the window state around the sample collection.
func NewApp(opts Options) (*App, error) {
	st, loadErr := settings.Default(), error(nil)
	if opts.SettingsPath != "" {
		st, loadErr = settings.Load(opts.SettingsPath)
	}
	th := ThemeFor(st.Theme, st.TextSize)
	ctx, err := widget.NewCtx(th)
	if err != nil {
		return nil, err
	}
	a := &App{
		ctx:      ctx,
		reg:      proto.NewRegistry(),
		col:      SampleCollection(),
		expanded: map[string]bool{},
		results:  make(chan sendResult, 8),
		status:   "Ready",
		shot:     screenshot{path: opts.ScreenshotPath, sendOnStart: opts.SendOnStart},

		settings:     st,
		settingsPath: opts.SettingsPath,
		menu:         opts.OpenMenu,
	}
	if loadErr != nil {
		a.status = loadErr.Error()
	}
	ctx.Clipboard = statusClipboard{inner: ctx.Clipboard, report: func(msg string) { a.status = msg }}
	a.dragTab = -1
	a.expanded[a.col.Name] = true
	a.expanded[a.col.Name+"/Repository"] = true
	a.openRequest(&a.col.Folders[1].Requests[0])
	return a, nil
}

func (a *App) cur() *reqTab { return a.tabs[a.active] }

// openRequest focuses the tab of req, opening one if needed.
func (a *App) openRequest(req *model.Request) {
	for i, t := range a.tabs {
		if t.req == req {
			a.active = i
			return
		}
	}
	a.tabs = append(a.tabs, newTab(req))
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

func (a *App) drain() {
	for {
		select {
		case r := <-a.results:
			a.apply(r)
		default:
			return
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
	a.w, a.h = w, h
	scale := ebiten.Monitor().DeviceScaleFactor()
	if err := a.ctx.SetScale(scale); err != nil {
		a.shot.err = err
	}
	return int(float64(w)*a.ctx.Scale + 0.5), int(float64(h)*a.ctx.Scale + 0.5)
}

// Update implements ebiten.Game: it samples input and runs one frame.
func (a *App) Update() error {
	a.step(a.sampler.Read().Scaled(a.ctx.Scale))
	ebiten.SetCursorShape(cursorShapes[a.ctx.Cursor])
	return a.shotUpdate()
}

// step runs one frame with the given input: it applies finished sends,
// builds the view, lays it out and lets it react. Handlers in the view
// change App state. Tests drive the app through step without a window.
func (a *App) step(in widget.Input) {
	a.ctx.Tick++
	a.ctx.In = in
	a.ctx.Cursor = widget.CursorDefault
	if !in.Held {
		a.dragTab = -1
	}
	a.drain()
	a.runShortcuts(in)
	if in.Escape { // one step per press: the menu, then a cell edit, then a running request
		switch {
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
	root := a.view()
	vml.Layout(a.ctx, root, a.w, a.h)
	vml.Update(a.ctx, root)
	if a.edit != nil && !a.edit.field.Focused { // a click elsewhere ends the edit
		a.commitEdit()
	}
}

// Draw implements ebiten.Game. The view is rebuilt so it shows the
// state the handlers just changed.
func (a *App) Draw(screen *ebiten.Image) {
	root := a.view()
	vml.Layout(a.ctx, root, a.w, a.h)
	vml.Draw(a.ctx, root, screen)
	a.shotDraw(screen)
}

func (a *App) clickTree(row treeRow) {
	if row.Folder {
		a.expanded[row.Path] = !a.expanded[row.Path]
		return
	}
	a.openRequest(row.Req)
}

func (a *App) newRequest() {
	a.tabs = append(a.tabs, newTab(&model.Request{Name: "New Request", Protocol: "rest", Method: "GET"}))
	a.active = len(a.tabs) - 1
}

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
