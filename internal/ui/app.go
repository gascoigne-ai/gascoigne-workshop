package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/proto"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
)

const (
	version        = "v0.1.0"
	requestTimeout = 60 * time.Second
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
	req      *model.Request
	url      TextField
	paneTab  int
	respTab  int
	resp     *model.Response
	view     CodeView // response body
	bodyView CodeView // request body, read-only
	sending  bool
}

func newTab(req *model.Request) *reqTab {
	t := &reqTab{req: req, url: NewTextField(req.URL)}
	t.bodyView.SetBody([]byte(req.Body))
	return t
}

type sendResult struct {
	tab  *reqTab
	resp model.Response
}

// App is the whole window. It implements ebiten.Game.
type App struct {
	ctx      Ctx
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
	settingsPath string // "" keeps settings in memory only
	menu         bool   // settings menu is open
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
	fonts, err := NewFonts(th, 1)
	if err != nil {
		return nil, err
	}
	a := &App{
		ctx:      Ctx{Th: th, F: fonts, Scale: 1},
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
	a.status = "Sending " + t.req.URL
	req := *t.req
	req.URL = model.Expand(req.URL, a.col.Vars)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()
		resp, err := a.reg.Send(ctx, req)
		if err != nil {
			resp = model.Response{Err: err.Error()}
		}
		a.results <- sendResult{tab: t, resp: resp}
	}()
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
	t.sending = false
	t.resp = &r.resp
	t.view.SetBody(r.resp.Body)
	if r.resp.Err != "" {
		a.status = "Request failed"
		return
	}
	a.status = fmt.Sprintf("%s %s: %d %s in %s", t.req.Method, t.req.URL, r.resp.Status, r.resp.StatusText, formatDuration(r.resp.Duration))
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

// Update implements ebiten.Game.
func (a *App) Update() error {
	a.ctx.Tick++
	a.ctx.In = ReadInput().scaled(a.ctx.Scale)
	a.drain()
	l := Compute(a.w, a.h)
	if a.updateMenu(l.Header) {
		return a.shotUpdate()
	}
	a.updateTree(l.Tree)
	a.updateTabStrip(l.Tabs)
	t := a.cur()
	a.updateURLBar(t, l.URL)
	a.updatePanes(t, l.Request, requestPanes, &t.paneTab)
	a.updatePanes(t, l.Response, responsePanes, &t.respTab)
	return a.shotUpdate()
}

func (a *App) updateTree(r Rect) {
	in := a.ctx.In
	if !in.Hit(r) {
		return
	}
	rows := treeRows(a.col.Folder, a.expanded)
	i := (in.Y - r.Y) / a.ctx.Th.RowH
	if i < 0 || i >= len(rows) {
		return
	}
	if row := rows[i]; row.Folder {
		a.expanded[row.Path] = !a.expanded[row.Path]
	} else {
		a.openRequest(row.Req)
	}
}

func (a *App) updateTabStrip(r Rect) {
	in := a.ctx.In
	rects, plus := a.tabStripRects(r)
	for i, tr := range rects {
		if in.Hit(tr) {
			a.active = i
		}
	}
	if in.Hit(plus) {
		a.tabs = append(a.tabs, newTab(&model.Request{Name: "New Request", Protocol: "rest", Method: "GET"}))
		a.active = len(a.tabs) - 1
	}
}

func (a *App) updateURLBar(t *reqTab, r Rect) {
	in := a.ctx.In
	protoR, methodR, field, send := urlBar(r, t.req.Protocol == "rest")
	if in.Hit(protoR) {
		t.req.Protocol = next(protocolOrder, t.req.Protocol)
		if t.req.Protocol == "rest" && t.req.Method == "" {
			t.req.Method = "GET"
		}
	}
	if in.Hit(methodR) {
		t.req.Method = next(methodOrder, t.req.Method)
	}
	if in.Click {
		t.url.Focused = field.Contains(in.X, in.Y)
		if t.url.Focused {
			t.url.Cursor = a.ctx.cursorAt(t.url.Text, in.X-(field.X+10))
		}
	}
	if t.url.Focused && t.url.Apply(in.Typed, in.Edits) {
		t.req.URL = t.url.Text
		t.req.Params = model.MergeParams(t.req.Params, t.req.URL)
	}
	if in.Hit(send) || (t.url.Focused && in.Enter) {
		a.send(t)
	}
}

// cursorAt maps a pixel offset into s to the nearest rune boundary.
func (c *Ctx) cursorAt(s string, x int) int {
	runes := []rune(s)
	best, bestDist := 0, abs(x)
	for i := 1; i <= len(runes); i++ {
		if d := abs(x - c.width(c.F.UI, string(runes[:i]))); d < bestDist {
			best, bestDist = i, d
		}
	}
	return best
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// urlBar places the protocol chip, method chip (REST only), the URL
// field and the Send button inside r.
func urlBar(r Rect, rest bool) (protoR, methodR, field, send Rect) {
	inner := Rect{r.X + 12, r.Y + 8, r.W - 24, r.H - 16}
	box, send := inner.SplitRight(88)
	send.X += 8
	send.W -= 8
	protoR, field = box.SplitLeft(64)
	if rest {
		methodR, field = field.SplitLeft(84)
	}
	return protoR, methodR, field, send
}

// Draw implements ebiten.Game.
func (a *App) Draw(screen *ebiten.Image) {
	a.drawWindow(screen)
	a.shotDraw(screen)
}
