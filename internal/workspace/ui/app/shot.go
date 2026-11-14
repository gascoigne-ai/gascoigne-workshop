package ui

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"strconv"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"

	"github.com/hajimehoshi/ebiten/v2"
)

// settleFrames lets fonts and layout warm up before a capture.
const (
	settleFrames = 10
	shotTimeout  = 1200 // frames to wait for a response before giving up
	setFrame     = 5    // the frame at which live settings are applied, early enough to be drawn before the capture
)

// screenshot saves one frame to a PNG and ends the program. It exists
// so the UI can be checked without a human at the window.
type screenshot struct {
	path        string
	sendOnStart bool
	frames      int
	sent        bool
	waiting     []*reqTab // tabs whose send the capture waits for
	sendNext    *reqTab   // the tab -request opened, sent once the earlier sends are answered
	captured    bool
	err         error
	set         []string // key=value settings applied live before the capture, as the dialog would
	request     string   // name of the request to open before the capture
	page        string   // settings page to show before the capture
	sentLog     bool     // show the log of sent requests in the sidebar
	find        string   // text to search the response for in the find bar
	found       bool     // the find bar is open with find typed in
	filter      string   // text to type into the sidebar's filter
	sending     bool     // send the open request and capture while it is in flight
	openAll     bool     // open every request in its own tab, so the tab strip overflows
	url         string   // text to type into the open request's URL, so its tab is marked changed
	pane        string   // label of the request pane to show, such as Settings
	toasts      []string // texts shown as error toasts
	dialog      string   // kind of dialog to open on the open request, a key of shotDialogs
}

// shotDialogs are the dialogs -dialog opens on the open request, each
// through the path its tree menu item takes.
var shotDialogs = map[string]func(a *App, node tree.Node){
	"rename": (*App).renameNode,
	"delete": (*App).deleteNode,
}

// Err is the error of the capture, if any.
func (a *App) Err() error { return a.shot.err }

func (a *App) shotUpdate() error {
	s := &a.shot
	if s.path == "" {
		return nil
	}
	if s.captured {
		return ebiten.Termination
	}
	s.frames++
	if s.frames == setFrame {
		for _, spec := range s.set {
			a.applyShotSetting(spec)
		}
		a.showShotTarget()
	}
	if s.sendOnStart && !s.sent {
		s.sent = true
		a.shotSend(a.cur())
	}
	if s.sendNext != nil && s.allAnswered() { // in turn, so the sent log's order is fixed
		next := s.sendNext
		s.sendNext = nil
		a.shotSend(next)
	}
	if s.find != "" && !s.found && s.frames >= setFrame && s.sendsDone() { // after the response, or there is nothing to search
		s.found = true
		a.shotFind()
	}
	return nil
}

// shotFind opens the find bar on the current response and types the query,
// as Ctrl+F and typing would.
func (a *App) shotFind() {
	a.openFind()
	t := a.cur()
	t.find.Insert(a.shot.find)
	a.runFind(t)
}

// sendsDone reports whether every send the capture asked for has its outcome.
func (s *screenshot) sendsDone() bool {
	return !s.sendOnStart || (s.sent && s.sendNext == nil && s.allAnswered())
}

// shotSend sends t's request and makes the capture wait for its outcome.
func (a *App) shotSend(t *reqTab) {
	a.shot.waiting = append(a.shot.waiting, t)
	a.send(t)
}

// allAnswered reports whether every send of the capture has its outcome.
func (s *screenshot) allAnswered() bool {
	for _, t := range s.waiting {
		if !answered(t) {
			return false
		}
	}
	return true
}

// ready reports whether the frame about to be drawn should be saved.
func (s *screenshot) ready(a *App) bool {
	if s.path == "" || s.captured || s.frames < settleFrames {
		return false
	}
	if !s.sendsDone() && s.frames < shotTimeout {
		return false
	}
	if s.sending && !a.cur().Sending && s.frames < shotTimeout {
		return false
	}
	return s.find == "" || s.found || s.frames >= shotTimeout
}

// answered reports whether what -send started has an outcome to show: a
// response, or for a socket an open connection (an MCP server also has
// said who it is) or a failed one.
func answered(t *reqTab) bool {
	if !isSocket(t) {
		return t.Resp != nil
	}
	if t.Connecting {
		return false
	}
	if t.Session == nil {
		return len(t.wsView.Lines) > 0 // the error is in the log
	}
	return t.Req.Protocol != "mcp" || t.Server != nil
}

func (a *App) shotDraw(screen *ebiten.Image) {
	s := &a.shot
	if !s.ready(a) {
		return
	}
	s.captured = true
	b := screen.Bounds()
	img := &image.RGBA{Pix: make([]byte, 4*b.Dx()*b.Dy()), Stride: 4 * b.Dx(), Rect: image.Rect(0, 0, b.Dx(), b.Dy())}
	screen.ReadPixels(img.Pix)
	f, err := os.Create(s.path)
	if err != nil {
		s.err = fmt.Errorf("create screenshot: %w", err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		s.err = fmt.Errorf("encode screenshot: %w", err)
	}
}

// parseSetting reads "key=value": true and false are booleans, whole numbers
// are numbers, anything else is text.
func parseSetting(spec string) (key string, value any, err error) {
	key, text, ok := strings.Cut(spec, "=")
	if !ok || key == "" {
		return "", nil, fmt.Errorf("%q is not key=value", spec)
	}
	switch {
	case text == "true" || text == "false":
		return key, text == "true", nil
	}
	if number, convErr := strconv.Atoi(text); convErr == nil {
		return key, number, nil
	}
	return key, text, nil
}

// applyShotSetting changes one setting through the path the settings dialog
// uses, so a screenshot shows a live change and not a start-up one.
func (a *App) applyShotSetting(spec string) {
	key, value, err := parseSetting(spec)
	if err != nil {
		a.shot.err = err
		return
	}
	a.changeSettings(func(s *settings.Settings) { *s = settings.With(*s, key, value) })
}

// showShotTarget opens the request and the settings page a capture asks for,
// so a scene other than the first tab can be drawn. A request that is not in
// the collection ends the run with an error: showing the default tab instead
// would let a golden pass for the wrong scene.
func (a *App) showShotTarget() {
	s := &a.shot
	if s.openAll {
		walkRequests(&a.col.Folder, "", func(_ string, request *model.Request) { a.openRequest(request) })
	}
	if s.request != "" {
		req := a.findRequestByName(s.request)
		if req == nil {
			s.err = fmt.Errorf("screenshot request %q is not in the collection", s.request)
			s.captured = true
			return
		}
		a.openRequest(req)
		if s.sendOnStart {
			s.sendNext = a.cur()
		}
	}
	if s.pane != "" && !a.showShotPane(s.pane) {
		s.err = fmt.Errorf("screenshot pane %q is not a pane of the open request", s.pane)
		s.captured = true
		return
	}
	if s.dialog != "" {
		open, ok := shotDialogs[s.dialog]
		if !ok {
			s.err = fmt.Errorf("screenshot dialog %q is not one of rename, delete", s.dialog)
			s.captured = true
			return
		}
		open(a, tree.Of(a.cur().Req))
	}
	for _, text := range s.toasts {
		a.toast(text, true)
	}
	if s.page != "" {
		a.prefs.Cat = s.page
		a.menu = true
	}
	if s.sentLog {
		a.showSent = true
	}
	if s.filter != "" {
		a.filter.Insert(s.filter)
	}
	if s.url != "" {
		a.cur().url.Insert(s.url)
		a.urlEdited(a.cur())
		a.markStale() // no input reports this edit, so the cached unsaved marks would not see it
	}
	if s.sending {
		a.send(a.cur())
	}
}

// showShotPane selects the open request's pane with this label; it reports
// whether the request has one.
func (a *App) showShotPane(label string) bool {
	tab := a.cur()
	for i, candidate := range requestPanesFor(tab) {
		if candidate.label == label {
			tab.paneTab = i
			return true
		}
	}
	return false
}

func (a *App) findRequestByName(name string) *model.Request {
	var found *model.Request
	walkRequests(&a.col.Folder, "", func(_ string, req *model.Request) {
		if found == nil && req.Name == name {
			found = req
		}
	})
	return found
}
