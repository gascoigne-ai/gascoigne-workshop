//go:build !release

package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/app"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
)

// Step is one thing a scene does to the window before the capture, through
// the app's Driver. A step that fails ends the run with its message, so a
// scene that could not be set up never produces a picture that looks right.
type Step interface {
	apply(scene *scene) error
}

// scene is what a step works on: the driver of the window and whether the
// run sends the first request at start.
type scene struct {
	driver      *app.Driver
	sendOnStart bool
}

// deferred is a step that cannot run at start because it needs what a send
// brings back, such as the response to search or the tools a server
// announces. It is tried again after every frame once the sends are
// answered, until it reports ready.
type deferred interface {
	Step
	ready() bool
}

type openRequestStep struct{ name string }

// openRequest opens the request with this name, so a scene other than the
// first tab can be drawn. With send on start the request is sent too, after
// the earlier sends are answered.
func openRequest(name string) Step { return openRequestStep{name} }

func (step openRequestStep) apply(scene *scene) error {
	if err := scene.driver.OpenRequest(step.name); err != nil {
		return err
	}
	if scene.sendOnStart {
		scene.driver.SendAfterAnswers()
	}
	return nil
}

type showPaneStep struct{ label string }

// showPane shows the open request's pane with this label.
func showPane(label string) Step { return showPaneStep{label} }

func (step showPaneStep) apply(scene *scene) error { return scene.driver.ShowRequestPane(step.label) }

type showResponsePaneStep struct{ label string }

// showResponsePane shows the open request's response pane with this label.
func showResponsePane(label string) Step { return showResponsePaneStep{label} }

func (step showResponsePaneStep) apply(scene *scene) error {
	return scene.driver.ShowResponsePane(step.label)
}

type chooseBodyStep struct{ id string }

// chooseBody chooses the body type id on the open request, as its chip does.
func chooseBody(id string) Step { return chooseBodyStep{id} }

func (step chooseBodyStep) apply(scene *scene) error { return scene.driver.ChooseBodyType(step.id) }

type typeBodyStep struct{ text string }

// typeBody replaces the open request's body text, as selecting all and typing
// would. It needs a JSON or Text body, so put -body before it.
func typeBody(text string) Step { return typeBodyStep{text} }

func (step typeBodyStep) apply(scene *scene) error { return scene.driver.TypeBody(step.text) }

type typeURLStep struct{ text string }

// typeURL types text into the open request's URL, so its tab is marked changed.
func typeURL(text string) Step { return typeURLStep{text} }

func (step typeURLStep) apply(scene *scene) error {
	scene.driver.TypeURL(step.text)
	return nil
}

type wrapStep struct{}

// wrapOn turns on Wrap for the open request's response and body, as their two
// Wrap chips would.
func wrapOn() Step { return wrapStep{} }

func (wrapStep) apply(scene *scene) error {
	scene.driver.WrapOn()
	return nil
}

type openDialogStep struct{ kind string }

// openDialog opens the dialog of this kind (rename, delete, or the move and
// copy destination pickers) on the open request.
func openDialog(kind string) Step { return openDialogStep{kind} }

func (step openDialogStep) apply(scene *scene) error { return scene.driver.OpenDialog(step.kind) }

type toastStep struct{ text string }

// showToast shows text as an error toast, which stays until clicked, so a slow
// frame cannot let it fade before the capture.
func showToast(text string) Step { return toastStep{text} }

func (step toastStep) apply(scene *scene) error {
	scene.driver.ShowToast(step.text)
	return nil
}

type settingsPageStep struct{ name string }

// settingsPage opens the settings dialog on the page with this name.
func settingsPage(name string) Step { return settingsPageStep{name} }

func (step settingsPageStep) apply(scene *scene) error {
	scene.driver.OpenSettingsPage(step.name)
	return nil
}

type rawSettingsStep struct{}

// rawSettings opens the settings dialog on the text of settings.json, as its
// JSON button does.
func rawSettings() Step { return rawSettingsStep{} }

func (rawSettingsStep) apply(scene *scene) error {
	scene.driver.OpenRawSettings()
	return nil
}

type recordStep struct{ spec string }

// record starts recording on the shortcut action spec names ("ACTION") and,
// when spec is "ACTION=KEYS", delivers the keys as a key press would.
func record(spec string) Step { return recordStep{spec} }

func (step recordStep) apply(scene *scene) error { return scene.driver.RecordShortcut(step.spec) }

type sentLogStep struct{}

// sentLog shows the log of sent requests in the sidebar.
func sentLog() Step { return sentLogStep{} }

func (sentLogStep) apply(scene *scene) error {
	scene.driver.ShowSentLog()
	return nil
}

type callLogStep struct{}

// callLog opens the Log dialog of calls, as the status bar's Log button does.
func callLog() Step { return callLogStep{} }

func (callLogStep) apply(scene *scene) error {
	scene.driver.OpenCallLog()
	return nil
}

type browseStep struct{ dir string }

// browse opens the in-app file browser on dir, as choosing a file without a
// dialog program does.
func browse(dir string) Step { return browseStep{dir} }

func (step browseStep) apply(scene *scene) error { return scene.driver.BrowseFiles(step.dir) }

type closeAllStep struct{}

// closeAll closes every collection, so the tree shows its empty prompt.
func closeAll() Step { return closeAllStep{} }

func (closeAllStep) apply(scene *scene) error {
	scene.driver.CloseAllCollections()
	return nil
}

type addCollectionStep struct{ dir string }

// addCollection opens the collection saved in dir beside the open ones, as
// Open collection folder does. A folder with nothing in it is an error here,
// unlike in that prompt, where it starts a new collection: a scene meant to
// show a saved collection must not pass with an empty one.
func addCollection(dir string) Step { return addCollectionStep{dir} }

func (step addCollectionStep) apply(scene *scene) error { return scene.driver.AddCollection(step.dir) }

type openAllStep struct{}

// openAll opens every request of the collection in its own tab, so the tab
// strip overflows.
func openAll() Step { return openAllStep{} }

func (openAllStep) apply(scene *scene) error {
	scene.driver.OpenAllRequests()
	return nil
}

type filterStep struct{ text string }

// filter types text into the sidebar's filter.
func filter(text string) Step { return filterStep{text} }

func (step filterStep) apply(scene *scene) error {
	scene.driver.FilterTree(step.text)
	return nil
}

type sendOpenStep struct{ done bool }

// send sends the open request and captures while it is in flight, so the URL
// bar shows Cancel and the status bar shows the send in progress. It waits
// for the first frame, which is why it is deferred.
func send() Step { return &sendOpenStep{} }

func (step *sendOpenStep) apply(scene *scene) error {
	scene.driver.SendWithoutWaiting()
	step.done = true
	return nil
}

func (step *sendOpenStep) ready() bool { return step.done }

type setSettingStep struct{ spec string }

// setSetting changes the setting "key=value" through the path the settings
// dialog takes, so the window reacts as it does to a click there.
func setSetting(spec string) Step { return setSettingStep{spec} }

func (step setSettingStep) apply(scene *scene) error {
	key, value, err := parseSetting(step.spec)
	if err != nil {
		return err
	}
	return scene.driver.ChangeSetting(key, settingValue(value))
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

// settingValue is what parseSetting read as the typed value of a setting.
func settingValue(parsed any) settings.Value {
	switch value := parsed.(type) {
	case bool:
		return settings.Flag(value)
	case int:
		return settings.Number(value)
	}
	return settings.Text(fmt.Sprint(parsed))
}

type findStep struct {
	text string
	done bool
}

// find opens the find bar on the current response and types text into it, as
// Ctrl+F and typing would. It waits for the response, so it is deferred.
func find(text string) Step { return &findStep{text: text} }

func (step *findStep) apply(scene *scene) error {
	scene.driver.Find(step.text)
	step.done = true
	return nil
}

func (step *findStep) ready() bool { return step.done }

type toolFormStep struct {
	name string
	done bool
}

// toolForm opens the call form of the named tool of the open MCP request, as
// its Call button does. It needs send on start, which connects to the server
// that announces the tool, so it is deferred until the tool is there.
func toolForm(name string) Step { return &toolFormStep{name: name} }

func (step *toolFormStep) apply(scene *scene) error {
	if !scene.sendOnStart {
		return fmt.Errorf("tool %q needs -send, which connects to the server that announces it", step.name)
	}
	opened, err := scene.driver.OpenToolForm(step.name)
	step.done = opened
	return err
}

func (step *toolFormStep) ready() bool { return step.done }
