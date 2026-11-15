package app

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/controller/editor"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"

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
	rawSettings bool     // open the settings dialog on the text of settings.json
	sentLog     bool     // show the log of sent requests in the sidebar
	find        string   // text to search the response for in the find bar
	found       bool     // the find bar is open with find typed in
	filter      string   // text to type into the sidebar's filter
	sending     bool     // send the open request and capture while it is in flight
	openAll     bool     // open every request in its own tab, so the tab strip overflows
	url         string   // text to type into the open request's URL, so its tab is marked changed
	pane        string   // label of the request pane to show, such as Settings
	body        string   // body type to choose on the open request, such as file
	typeBody    string   // text that replaces the open request's body, as selecting all and typing would
	wrap        bool     // turn on the response's and the body's Wrap, as their chips would
	respPane    string   // label of the response pane to show, such as Tests
	toasts      []string // texts shown as error toasts
	dialog      string   // kind of dialog to open on the open request, a key of shotDialogs
	record      []string // ACTION=KEYS recorded on the Shortcuts page, as a user would; a bare ACTION is left recording
	callLog     bool     // open the Log dialog of calls, as the status bar's Log button does
	tool        string   // name of the open MCP request's tool whose call form to open, once the server announced it
	toolOpened  bool     // the form of tool is open
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
	if s.tool != "" && !s.toolOpened && s.frames >= setFrame && s.sendsDone() {
		a.shotToolForm()
	}
	return nil
}

// shotToolForm opens the call form of the capture's tool once the open
// request's server announced it, as picking it from the tool menu would.
// Once the server announced tools without it, the run ends with an error
// rather than waiting for the timeout.
func (a *App) shotToolForm() {
	t := a.cur()
	for _, tool := range t.Tools {
		if tool.Name == a.shot.tool {
			a.shot.toolOpened = true
			a.openToolForm(t, tool)
			return
		}
	}
	if len(t.Tools) == 0 {
		return // not announced yet
	}
	names := make([]string, len(t.Tools))
	for i, tool := range t.Tools {
		names[i] = tool.Name
	}
	a.shot.err = fmt.Errorf("screenshot tool %q is not one the server announced: %s", a.shot.tool, strings.Join(names, ", "))
	a.shot.captured = true
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
	if s.tool != "" && !s.toolOpened && s.frames < shotTimeout {
		return false
	}
	return s.find == "" || s.found || s.frames >= shotTimeout
}

// answered reports whether what -send started has an outcome to show: a
// response, or for a socket an open connection (an MCP server also has
// said who it is) or a failed one.
func answered(t *reqTab) bool {
	if !isSocket(t) {
		return t.Resp != nil && !t.Sending // a tab sent again still holds its earlier response
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
	if s.respPane != "" && !a.showShotResponsePane(s.respPane) {
		s.err = fmt.Errorf("screenshot response pane %q is not a response pane of the open request", s.respPane)
		s.captured = true
		return
	}
	if s.body != "" && !a.chooseShotBody(s.body) {
		s.err = fmt.Errorf("screenshot body type %q is not one of the body chips", s.body)
		s.captured = true
		return
	}
	if s.typeBody != "" && !a.typeShotBody(s.typeBody) {
		s.err = fmt.Errorf("screenshot body text needs a JSON or Text body, the open request's is %q", a.cur().Req.BodyType)
		s.captured = true
		return
	}
	if s.wrap {
		a.cur().view.Wrap = true
		a.cur().body.Wrap = true
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
	if s.tool != "" && !s.sendOnStart {
		s.err = fmt.Errorf("screenshot tool %q needs -send, which connects to the server that announces it", s.tool)
		s.captured = true
		return
	}
	for _, text := range s.toasts {
		a.toast(text, true)
	}
	if s.page != "" {
		a.prefs.Cat = s.page
		a.menu = true
	}
	for _, spec := range s.record {
		if err := a.shotRecord(spec); err != nil {
			s.err = err
			s.captured = true
			return
		}
	}
	if s.rawSettings {
		a.prefs.OpenRaw(a)
		a.menu = true
	}
	if s.sentLog {
		a.showSent = true
	}
	if s.callLog {
		a.calls.Open()
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

// shotRecord starts recording on the action spec names and, when spec
// gives keys, delivers them as the frame's input the way a key press is.
func (a *App) shotRecord(spec string) error {
	id, keys, hasKeys := strings.Cut(spec, "=")
	known := slices.ContainsFunc(shortcuts, func(candidate shortcut) bool { return candidate.id == id })
	if !known {
		return fmt.Errorf("screenshot record: %q is not a shortcut action", id)
	}
	a.shortcutKeys.Start(id)
	if !hasKeys {
		return nil
	}
	binding, err := engine.ParseBinding(keys)
	if err != nil {
		return fmt.Errorf("screenshot record %s: %w", id, err)
	}
	a.shortcutKeys.RecordKey(a, core.Input{Keys: []ebiten.Key{binding.Key}, Ctrl: binding.Ctrl, Shift: binding.Shift, Alt: binding.Alt})
	return nil
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

// showShotResponsePane selects the open request's response pane with this
// label; it reports whether the request has one.
func (a *App) showShotResponsePane(label string) bool {
	tab := a.cur()
	for i, candidate := range responsePanesFor(tab) {
		if candidate.label == label {
			tab.respTab = i
			return true
		}
	}
	return false
}

// chooseShotBody chooses the body type id on the open request, as its chip
// does; it reports whether id is one of the chips.
func (a *App) chooseShotBody(id string) bool {
	known := slices.ContainsFunc(bodyTypes, func(candidate struct {
		id, label string
		width     int
	}) bool {
		return candidate.id == id
	})
	if !known {
		return false
	}
	a.setBodyType(a.cur(), id)
	a.markStale() // no input reports this edit, so the cached unsaved marks would not see it
	return true
}

// typeShotBody replaces the open request's body text, as selecting all and
// typing would; it reports false for a body type whose editor is not the
// text area, where the text would not be shown.
func (a *App) typeShotBody(text string) bool {
	t := a.cur()
	if t.Req.BodyType != model.BodyJSON && t.Req.BodyType != model.BodyText {
		return false
	}
	t.body.SelectAll()
	t.body.Insert(text)
	editRequest(t, editor.BodyTyped{Text: t.body.Text})
	a.markStale() // no input reports this edit, so the cached unsaved marks would not see it
	return true
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
