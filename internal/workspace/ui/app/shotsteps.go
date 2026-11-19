package app

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/tree"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/model/severity"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/transfer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/controller/editor"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"

	"github.com/hajimehoshi/ebiten/v2"
)

// Step is one thing a screenshot scene does to the window before the capture,
// such as opening a request or a dialog. A scene is a list of steps run in
// order at the setFrame frame, so a repeated step (two toasts) is two entries
// and a step with a bad argument ends the run with its error.
type Step interface {
	apply(app *App) error
}

// deferred is a step that cannot run at start because it needs what a send
// brings (a response to search, the tools a server announces). It runs once
// the sends the capture asked for are answered, is tried each frame until it
// is ready, and the capture waits for it until the shot timeout.
type deferred interface {
	Step
	ready(app *App) bool
}

// shotDialogs are the dialogs the OpenDialog step opens on the open request,
// each through the path its tree menu item takes.
var shotDialogs = map[string]func(app *App, node model.Node){
	"rename": (*App).renameNode,
	"delete": (*App).deleteNode,
	"move":   func(app *App, node model.Node) { app.openTransferPicker(node, transfer.Move) },
	"copy":   func(app *App, node model.Node) { app.openTransferPicker(node, transfer.Copy) },
}

type openRequestStep struct{ name string }

// OpenRequest opens the request with this name, so a scene other than the
// first tab can be drawn. A name that is not in the collection is an error:
// showing the default tab instead would let a golden pass for the wrong scene.
func OpenRequest(name string) Step { return openRequestStep{name} }

func (step openRequestStep) apply(app *App) error {
	request := app.findRequestByName(step.name)
	if request == nil {
		return fmt.Errorf("screenshot request %q is not in the collection", step.name)
	}
	app.openRequest(request)
	if app.shot.sendOnStart {
		app.shot.sendNext = app.cur()
	}
	return nil
}

type showPaneStep struct{ label string }

// ShowPane shows the open request's pane with this label.
func ShowPane(label string) Step { return showPaneStep{label} }

func (step showPaneStep) apply(app *App) error {
	tab := app.cur()
	for i, candidate := range requestPanesFor(tab) {
		if candidate.label == step.label {
			tab.paneTab = i
			return nil
		}
	}
	return fmt.Errorf("screenshot pane %q is not a pane of the open request", step.label)
}

type showResponsePaneStep struct{ label string }

// ShowResponsePane shows the open request's response pane with this label.
func ShowResponsePane(label string) Step { return showResponsePaneStep{label} }

func (step showResponsePaneStep) apply(app *App) error {
	tab := app.cur()
	for i, candidate := range responsePanesFor(tab) {
		if candidate.label == step.label {
			tab.respTab = i
			return nil
		}
	}
	return fmt.Errorf("screenshot response pane %q is not a response pane of the open request", step.label)
}

type chooseBodyStep struct{ id string }

// ChooseBody chooses the body type id on the open request, as its chip does.
func ChooseBody(id string) Step { return chooseBodyStep{id} }

func (step chooseBodyStep) apply(app *App) error {
	known := slices.ContainsFunc(bodyTypes, func(candidate bodyChoice) bool {
		return candidate.id == model.BodyType(step.id)
	})
	if !known {
		return fmt.Errorf("screenshot body type %q is not one of the body chips", step.id)
	}
	app.setBodyType(app.cur(), model.BodyType(step.id))
	app.markStale() // no input reports this edit, so the cached unsaved marks would not see it
	return nil
}

type typeBodyStep struct{ text string }

// TypeBody replaces the open request's body text, as selecting all and typing
// would. A body type whose editor is not the text area is an error, as the
// text would not be shown.
func TypeBody(text string) Step { return typeBodyStep{text} }

func (step typeBodyStep) apply(app *App) error {
	tab := app.cur()
	if tab.Req.BodyType != model.BodyJSON && tab.Req.BodyType != model.BodyText {
		return fmt.Errorf("screenshot body text needs a JSON or Text body, the open request's is %q", tab.Req.BodyType)
	}
	tab.body.SelectAll()
	tab.body.Insert(step.text)
	editRequest(tab, editor.BodyTyped{Text: tab.body.Text})
	app.markStale() // no input reports this edit, so the cached unsaved marks would not see it
	return nil
}

type typeURLStep struct{ text string }

// TypeURL types text into the open request's URL, so its tab is marked changed.
func TypeURL(text string) Step { return typeURLStep{text} }

func (step typeURLStep) apply(app *App) error {
	app.cur().url.Insert(step.text)
	app.urlEdited(app.cur())
	app.markStale() // no input reports this edit, so the cached unsaved marks would not see it
	return nil
}

type wrapStep struct{}

// WrapOn turns on Wrap for the open request's response and body, as their two
// Wrap chips would.
func WrapOn() Step { return wrapStep{} }

func (wrapStep) apply(app *App) error {
	app.cur().view.Wrap = true
	app.cur().body.Wrap = true
	return nil
}

type openDialogStep struct{ kind string }

// OpenDialog opens the dialog of this kind (rename, delete, or the move and
// copy destination pickers) on the open request.
func OpenDialog(kind string) Step { return openDialogStep{kind} }

func (step openDialogStep) apply(app *App) error {
	open, ok := shotDialogs[step.kind]
	if !ok {
		return fmt.Errorf("screenshot dialog %q is not one of rename, delete, move, copy", step.kind)
	}
	open(app, model.Of(app.cur().Req))
	return nil
}

type toastStep struct{ text string }

// ShowToast shows text as an error toast, which stays until clicked, so a slow
// frame cannot let it fade before the capture.
func ShowToast(text string) Step { return toastStep{text} }

func (step toastStep) apply(app *App) error {
	app.toast(step.text, severity.Error)
	return nil
}

type settingsPageStep struct{ name string }

// SettingsPage opens the settings dialog on the page with this name.
func SettingsPage(name string) Step { return settingsPageStep{name} }

func (step settingsPageStep) apply(app *App) error {
	app.prefs.Cat = step.name
	app.menu = true
	return nil
}

type rawSettingsStep struct{}

// RawSettings opens the settings dialog on the text of settings.json, as its
// JSON button does.
func RawSettings() Step { return rawSettingsStep{} }

func (rawSettingsStep) apply(app *App) error {
	app.prefs.OpenRaw(app)
	app.menu = true
	return nil
}

type recordStep struct{ spec string }

// Record starts recording on the shortcut action spec names ("ACTION") and,
// when spec is "ACTION=KEYS", delivers the keys as a key press would.
func Record(spec string) Step { return recordStep{spec} }

func (step recordStep) apply(app *App) error {
	id, keys, hasKeys := strings.Cut(step.spec, "=")
	known := slices.ContainsFunc(shortcuts, func(candidate shortcut) bool { return candidate.id == id })
	if !known {
		return fmt.Errorf("screenshot record: %q is not a shortcut action", id)
	}
	app.shortcutKeys.Start(id)
	if !hasKeys {
		return nil
	}
	binding, err := engine.ParseBinding(keys)
	if err != nil {
		return fmt.Errorf("screenshot record %s: %w", id, err)
	}
	app.shortcutKeys.RecordKey(app, core.Input{Keys: []ebiten.Key{binding.Key}, Ctrl: binding.Ctrl, Shift: binding.Shift, Alt: binding.Alt})
	return nil
}

type sentLogStep struct{}

// SentLog shows the log of sent requests in the sidebar.
func SentLog() Step { return sentLogStep{} }

func (sentLogStep) apply(app *App) error {
	app.showSent = true
	return nil
}

type callLogStep struct{}

// CallLog opens the Log dialog of calls, as the status bar's Log button does.
func CallLog() Step { return callLogStep{} }

func (callLogStep) apply(app *App) error {
	app.calls.Open()
	return nil
}

type browseStep struct{ dir string }

// Browse opens the in-app file browser on dir (made absolute) with the title
// of the request body's file chooser, as choosing a file without a dialog
// program does.
func Browse(dir string) Step { return browseStep{dir} }

func (step browseStep) apply(app *App) error {
	dir, err := filepath.Abs(step.dir)
	if err != nil {
		return fmt.Errorf("screenshot file browser folder %q: %w", step.dir, err)
	}
	app.browseIn(dir, "Choose the file to send", false, "", func(string) {})
	return nil
}

type closeAllStep struct{}

// CloseAll closes every collection, so the tree shows its empty prompt.
func CloseAll() Step { return closeAllStep{} }

func (closeAllStep) apply(app *App) error {
	for len(app.open.Entries) > 0 {
		app.closeCollection(app.open.Entries[0].Collection)
	}
	return nil
}

type addCollectionStep struct{ dir string }

// AddCollection opens the collection saved in dir beside the open ones, as
// Open collection folder does. A folder with nothing in it is an error here,
// unlike in that prompt, where it starts a new collection: a scene meant to
// show a saved collection must not pass with an empty one.
func AddCollection(dir string) Step { return addCollectionStep{dir} }

func (step addCollectionStep) apply(app *App) error {
	loaded, err := app.store.Load(step.dir)
	if err != nil {
		return fmt.Errorf("screenshot collection %q: %w", step.dir, err)
	}
	if len(loaded.Folders) == 0 && len(loaded.Requests) == 0 {
		return fmt.Errorf("screenshot collection %q: the folder holds no collection", step.dir)
	}
	return app.addCollection(step.dir)
}

type openAllStep struct{}

// OpenAll opens every request of the collection in its own tab, so the tab
// strip overflows.
func OpenAll() Step { return openAllStep{} }

func (openAllStep) apply(app *App) error {
	app.eachRoot(func(root *model.Folder) {
		tree.WalkRequests(root, "", func(_ string, request *model.Request) { app.openRequest(request) })
	})
	return nil
}

type filterStep struct{ text string }

// Filter types text into the sidebar's filter.
func Filter(text string) Step { return filterStep{text} }

func (step filterStep) apply(app *App) error {
	app.filter.Insert(step.text)
	return nil
}

type sendOpenStep struct{ done bool }

// Send sends the open request and captures while it is in flight, so the URL
// bar shows Cancel. It is deferred, so it does not disturb the sends of
// SendOnStart: the capture waits for those and then starts this one.
func Send() Step { return &sendOpenStep{} }

func (step *sendOpenStep) apply(app *App) error {
	app.send(app.cur())
	step.done = true
	return nil
}

func (step *sendOpenStep) ready(*App) bool { return step.done }

type setSettingStep struct{ spec string }

// Set changes the setting "key=value" through the path the settings dialog
// uses, so a screenshot shows a live change and not a start-up one.
func Set(spec string) Step { return setSettingStep{spec} }

func (step setSettingStep) apply(app *App) error {
	key, value, err := parseSetting(step.spec)
	if err != nil {
		return err
	}
	app.changeSettings(func(prefs *settings.Settings) { *prefs = settings.With(*prefs, key, value) })
	return nil
}

type findStep struct {
	text string
	done bool
}

// Find opens the find bar on the current response and types text into it, as
// Ctrl+F and typing would, once every send is answered, or there is nothing
// to search.
func Find(text string) Step { return &findStep{text: text} }

func (step *findStep) apply(app *App) error {
	app.openFind()
	tab := app.cur()
	tab.find.Insert(step.text)
	app.runFind(tab)
	step.done = true
	return nil
}

func (step *findStep) ready(*App) bool { return step.done }

type toolFormStep struct {
	name string
	done bool
}

// ToolForm opens the call form of the named tool of the open MCP request, as
// picking it from the composer's tool menu does, once the server announced
// it. Without a send nothing connects and announces it, and a server that
// announced tools without it ends the run with an error rather than waiting
// for the timeout.
func ToolForm(name string) Step { return &toolFormStep{name: name} }

func (step *toolFormStep) apply(app *App) error {
	if !app.shot.sendOnStart {
		return fmt.Errorf("screenshot tool %q needs -send, which connects to the server that announces it", step.name)
	}
	tab := app.cur()
	for _, tool := range tab.Tools {
		if tool.Name == step.name {
			step.done = true
			app.openToolForm(tab, tool)
			return nil
		}
	}
	if len(tab.Tools) == 0 {
		return nil // not announced yet
	}
	names := make([]string, len(tab.Tools))
	for i, tool := range tab.Tools {
		names[i] = tool.Name
	}
	return fmt.Errorf("screenshot tool %q is not one the server announced: %s", step.name, strings.Join(names, ", "))
}

func (step *toolFormStep) ready(*App) bool { return step.done }
