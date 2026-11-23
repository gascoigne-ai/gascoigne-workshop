package app

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/transfer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"

	"github.com/hajimehoshi/ebiten/v2"
)

// Driver is the way a program outside the app puts the window into a state
// without a user's input: the end-to-end screenshots (e2e/cmd/shot) call
// its actions, each the same path as the user action it is named after, and
// then capture a frame. It lives in this package because every action
// reaches the state the views own; a driver outside would have to export
// all of it. Its one piece of state is the sends it started, which a
// capture waits for.
type Driver struct {
	app      *App
	waiting  []*reqTab // tabs whose send the driver started
	sendNext *reqTab   // the tab OpenRequest opened, sent once the earlier sends are answered
}

// Driver returns the driver of this window.
func (a *App) Driver() *Driver { return &a.driver }

// dialogOpeners are the dialogs OpenDialog opens on the open request, by the
// name a scene gives them.
var dialogOpeners = map[string]func(app *App, node request.Node){
	"rename": (*App).renameNode,
	"delete": (*App).deleteNode,
	"move":   func(app *App, node request.Node) { app.openTransferPicker(node, transfer.Move) },
	"copy":   func(app *App, node request.Node) { app.openTransferPicker(node, transfer.Copy) },
}

// OpenRequest opens the request with this name. A name that is not in the
// collection is an error: showing the default tab instead would let a golden
// pass for the wrong scene.
func (d *Driver) OpenRequest(name string) error {
	call := d.app.findRequestByName(name)
	if call == nil {
		return fmt.Errorf("request %q is not in the collection", name)
	}
	d.app.openRequest(call)
	return nil
}

// ShowRequestPane shows the open request's pane with this label.
func (d *Driver) ShowRequestPane(label string) error {
	if d.app.editor.ShowPaneLabeled(d.app.cur().Tab, label) {
		return nil
	}
	return fmt.Errorf("pane %q is not a pane of the open request", label)
}

// ShowResponsePane shows the open request's response pane with this label.
func (d *Driver) ShowResponsePane(label string) error {
	tab := d.app.cur()
	for i, candidate := range d.app.responsePanesFor(tab) {
		if candidate.label == label {
			tab.respTab = i
			return nil
		}
	}
	return fmt.Errorf("response pane %q is not a response pane of the open request", label)
}

// ChooseBodyType chooses the body type id on the open request, as its chip does.
func (d *Driver) ChooseBodyType(id string) error {
	if !d.app.editor.Body.Offers(request.BodyType(id)) {
		return fmt.Errorf("body type %q is not one of the body chips", id)
	}
	d.app.editor.Body.ChooseType(d.app.cur().Tab, request.BodyType(id))
	d.app.keys.MarkStale() // no input reports this edit, so the cached unsaved marks would not see it
	return nil
}

// TypeBody replaces the open request's body text, as selecting all and
// typing would. It needs a JSON or Text body.
func (d *Driver) TypeBody(text string) error {
	tab := d.app.cur()
	if tab.Req.BodyType != request.BodyJSON && tab.Req.BodyType != request.BodyText {
		return fmt.Errorf("body text needs a JSON or Text body, the open request's is %q", tab.Req.BodyType)
	}
	d.app.editor.Body.Replace(tab.Tab, text)
	d.app.keys.MarkStale() // no input reports this edit, so the cached unsaved marks would not see it
	return nil
}

// TypeURL types text into the open request's URL, so its tab is marked changed.
func (d *Driver) TypeURL(text string) {
	d.app.editor.URLBar.Type(d.app.cur().Tab, text)
	d.app.keys.MarkStale() // no input reports this edit, so the cached unsaved marks would not see it
}

// WrapOn turns on Wrap for the open request's response and body, as their two
// Wrap chips would.
func (d *Driver) WrapOn() {
	d.app.resp.Code(d.app.cur().Tab).Wrap = true
	d.app.editor.Body.Area(d.app.cur().Tab).Wrap = true
}

// OpenDialog opens the dialog of this kind (rename, delete, or the move and
// copy destination pickers) on the open request.
func (d *Driver) OpenDialog(kind string) error {
	open, ok := dialogOpeners[kind]
	if !ok {
		return fmt.Errorf("dialog %q is not one of rename, delete, move, copy", kind)
	}
	open(d.app, request.Node{Req: d.app.cur().Req})
	return nil
}

// ShowToast shows text as an error toast, which stays until clicked, so a slow
// frame cannot let it fade before a capture.
func (d *Driver) ShowToast(text string) { d.app.toast(text, severity.Error) }

// OpenSettingsMenu opens the settings dialog.
func (d *Driver) OpenSettingsMenu() { d.app.over.Settings.Show() }

// OpenSettingsPage opens the settings dialog on the page with this name.
func (d *Driver) OpenSettingsPage(name string) {
	d.app.over.Settings.Shell.Cat = name
	d.app.over.Settings.Show()
}

// OpenRawSettings opens the settings dialog on the text of settings.json, as
// its JSON button does.
func (d *Driver) OpenRawSettings() {
	d.app.over.Settings.OpenRaw()
	d.app.over.Settings.Show()
}

// RecordShortcut starts recording on the shortcut action spec names
// ("ACTION") and, when spec is "ACTION=KEYS", delivers the keys as a key
// press would.
func (d *Driver) RecordShortcut(spec string) error {
	id, keys, hasKeys := strings.Cut(spec, "=")
	known := slices.Contains(d.app.over.Settings.ActionIDs(), id)
	if !known {
		return fmt.Errorf("record: %q is not a shortcut action", id)
	}
	d.app.over.Settings.Shortcuts.Start(id)
	if !hasKeys {
		return nil
	}
	binding, err := engine.ParseBinding(keys)
	if err != nil {
		return fmt.Errorf("record %s: %w", id, err)
	}
	d.app.over.Settings.Shortcuts.RecordKey(d.app.over.Settings, core.Input{Keys: []ebiten.Key{binding.Key}, Ctrl: binding.Ctrl, Shift: binding.Shift, Alt: binding.Alt})
	return nil
}

// ShowSentLog shows the log of sent requests in the sidebar.
func (d *Driver) ShowSentLog() { d.app.side.SetShowSent(true) }

// OpenCallLog opens the Log dialog of calls, as the status bar's Log button does.
func (d *Driver) OpenCallLog() { d.app.over.Calls().Open() }

// BrowseFiles opens the in-app file browser on dir (made absolute) with the
// title of the request body's file chooser, as choosing a file without a
// dialog program does.
func (d *Driver) BrowseFiles(dir string) error {
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("file browser folder %q: %w", dir, err)
	}
	d.app.over.Browser.Browse(absolute, "Choose the file to send", filedialog.OpenFile, "", func(string) {})
	return nil
}

// CloseAllCollections closes every collection, so the tree shows its empty prompt.
func (d *Driver) CloseAllCollections() {
	for len(d.app.cols.Set().Entries) > 0 {
		d.app.closeCollection(d.app.cols.Set().Entries[0].Collection)
	}
}

// AddCollection opens the collection saved in dir beside the open ones, as
// Open collection folder does. A folder with nothing in it is an error here,
// unlike in that prompt, where it starts a new collection: a scene meant to
// show a saved collection must not pass with an empty one.
func (d *Driver) AddCollection(dir string) error {
	loaded, err := d.app.store.Load(dir)
	if err != nil {
		return fmt.Errorf("collection %q: %w", dir, err)
	}
	if len(loaded.Folders) == 0 && len(loaded.Requests) == 0 {
		return fmt.Errorf("collection %q: the folder holds no collection", dir)
	}
	return d.app.addCollection(dir)
}

// OpenAllRequests opens every request of the collection in its own tab, so
// the tab strip overflows.
func (d *Driver) OpenAllRequests() {
	d.app.eachRoot(func(root *request.Folder) {
		tree.WalkRequests(root, "", func(_ string, call *request.Request) { d.app.openRequest(call) })
	})
}

// FilterTree types text into the sidebar's filter.
func (d *Driver) FilterTree(text string) { d.app.side.Filter().Insert(text) }

// ChangeSetting changes a setting through the path the settings dialog takes,
// so the window reacts as it does to a click there.
func (d *Driver) ChangeSetting(key string, value settings.Value) error {
	var setErr error
	d.app.changeSettings(func(prefs *settings.Settings) { setErr = settings.Set(prefs, key, value) })
	return setErr
}

// Find opens the find bar on the current response and types text into it, as
// Ctrl+F and typing would.
func (d *Driver) Find(text string) {
	d.app.resp.OpenFind()
	d.app.resp.Find(d.app.cur().Tab, text)
}

// OpenToolForm opens the call form of the named tool of the open MCP request.
// It reports false, with no error, while the server has announced no tools
// yet; a tool missing from the announced ones is an error.
func (d *Driver) OpenToolForm(name string) (opened bool, err error) {
	tab := d.app.cur()
	for _, tool := range tab.Tools {
		if tool.Name == name {
			d.app.editor.MCP.OpenToolForm(tab.Tab, tool)
			return true, nil
		}
	}
	if len(tab.Tools) == 0 {
		return false, nil
	}
	names := make([]string, len(tab.Tools))
	for i, tool := range tab.Tools {
		names[i] = tool.Name
	}
	return false, fmt.Errorf("tool %q is not one the server announced: %s", name, strings.Join(names, ", "))
}

// Send sends the open request, as the Send button does, and remembers it so
// AllAnswered can tell when its outcome is in.
func (d *Driver) Send() {
	tab := d.app.cur()
	d.waiting = append(d.waiting, tab)
	d.app.send(tab)
}

// SendAfterAnswers marks the open request to be sent by ServeSends once every
// earlier send is answered, so the sent log keeps its order.
func (d *Driver) SendAfterAnswers() { d.sendNext = d.app.cur() }

// ServeSends sends the request SendAfterAnswers marked, if the earlier sends
// are answered. Call it once per frame.
func (d *Driver) ServeSends() {
	if d.sendNext == nil || !d.AllAnswered() {
		return
	}
	tab := d.sendNext
	d.sendNext = nil
	d.waiting = append(d.waiting, tab)
	d.app.send(tab)
}

// SendsPending reports whether a send marked by SendAfterAnswers has not
// gone out yet.
func (d *Driver) SendsPending() bool { return d.sendNext != nil }

// AllAnswered reports whether every send the driver started has its outcome.
func (d *Driver) AllAnswered() bool {
	return !slices.ContainsFunc(d.waiting, func(tab *reqTab) bool { return !d.app.answered(tab) })
}

// Settled reports whether the open request shows nothing still loading: its
// Proto pane waits for the files to be read and its Schema pane for the
// message schema.
func (d *Driver) Settled() bool {
	return !d.app.editor.Loading(d.app.cur().Tab)
}

// SendWithoutWaiting sends the open request and does not wait for its
// outcome, so a capture shows it in flight.
func (d *Driver) SendWithoutWaiting() { d.app.send(d.app.cur()) }

// answered reports whether what a driver sent has an outcome to show: a
// response, or for a socket an open connection (an MCP server also has
// said who it is) or a failed one.
func (a *App) answered(tab *reqTab) bool {
	if !a.editor.Socket(tab.Tab) {
		return tab.Resp != nil && !a.sends.Sending(tab.Req) // a tab sent again still holds its earlier response
	}
	if tab.State() == tabs.ConnConnecting {
		return false
	}
	if tab.Session() == nil {
		return len(a.editor.WS.Lines(tab.Tab)) > 0 // the error is in the log
	}
	return !a.editor.Explores(tab.Tab) || tab.Server != nil // an explorer waits for the server to say who it is
}

func (a *App) findRequestByName(name string) *request.Request {
	var found *request.Request
	a.eachRoot(func(root *request.Folder) {
		tree.WalkRequests(root, "", func(_ string, req *request.Request) {
			if found == nil && req.Name == name {
				found = req
			}
		})
	})
	return found
}
