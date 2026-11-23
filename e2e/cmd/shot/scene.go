//go:build !release

package main

import (
	"flag"
	"strconv"
)

// sceneFlag is one command line flag that adds a step to a screenshot scene.
type sceneFlag struct {
	name   string
	usage  string
	toggle bool // takes no value: -wrap, not -wrap=text
	repeat bool // may be given several times, each a step; otherwise the last one wins
	step   func(value string) Step
}

// sceneFlagTable lists the scene flags in the order their steps run, whatever
// the order on the command line: a request is opened before its pane is shown,
// and what waits for a response (send, find, tool) comes last. A toggle's step
// ignores its value.
var sceneFlagTable = []sceneFlag{
	{name: "set", usage: "key=value: change a setting live before the screenshot, as the settings dialog would (repeatable, for screenshots)", repeat: true, step: setSetting},
	{name: "add-collection", usage: "open the collection saved in this folder beside the others before the screenshot, as Open collection folder does (repeatable)", repeat: true, step: addCollection},
	{name: "open-all", usage: "open every request of the collection in its own tab before the screenshot", toggle: true, step: func(string) Step { return openAll() }},
	{name: "request", usage: "open the request with this name before the screenshot", step: openRequest},
	{name: "pane", usage: "show the open request's pane with this label (Params, Settings, ...) before the screenshot", step: showPane},
	{name: "response-pane", usage: "show the open request's response pane with this label (Headers, Tests, ...) before the screenshot", step: showResponsePane},
	{name: "body", usage: "choose this body type (json, text, form, multipart, graphql, file, ...) on the open request before the screenshot", step: chooseBody},
	{name: "type-body", usage: "replace the open request's body text (JSON or Text body) with this text before the screenshot", step: typeBody},
	{name: "wrap", usage: "turn on Wrap for the open request's response and body before the screenshot, as their Wrap chips do", toggle: true, step: func(string) Step { return wrapOn() }},
	{name: "dialog", usage: "open this dialog (rename, delete, or the destination picker: move, copy) on the open request before the screenshot", step: openDialog},
	{name: "toast", usage: "show this text as an error toast before the screenshot (repeatable)", repeat: true, step: showToast},
	{name: "page", usage: "open the settings dialog on this page (appearance, colors, shortcuts, requests, general, system) before the screenshot", step: settingsPage},
	{name: "font-list", usage: "open the settings dialog on Appearance with the font list of this font (ui, code) open before the screenshot", step: fontList},
	{name: "record", usage: "ACTION=KEYS: record KEYS for the shortcut ACTION on the Shortcuts page before the screenshot; a bare ACTION is left recording (repeatable)", repeat: true, step: record},
	{name: "raw-settings", usage: "open the settings dialog on the text of settings.json before the screenshot", toggle: true, step: func(string) Step { return rawSettings() }},
	{name: "sent", usage: "show the log of sent requests in the sidebar before the screenshot", toggle: true, step: func(string) Step { return sentLog() }},
	{name: "log", usage: "open the Log dialog of calls before the screenshot, as the status bar's Log button does", toggle: true, step: func(string) Step { return callLog() }},
	{name: "browse", usage: "open the in-app file browser on this folder before the screenshot, as choosing a file without a dialog program does", step: browse},
	{name: "close-all", usage: "close every collection before the screenshot, so the tree shows its empty prompt", toggle: true, step: func(string) Step { return closeAll() }},
	{name: "filter", usage: "type this text into the sidebar's filter before the screenshot", step: filter},
	{name: "type-url", usage: "type this text into the open request's URL before the screenshot, so its tab shows unsaved changes", step: typeURL},
	{name: "sending", usage: "send the open request and take the screenshot while it is in flight", toggle: true, step: func(string) Step { return send() }},
	{name: "find", usage: "search the response for this text in the find bar before the screenshot (after -send's response)", step: find},
	{name: "tool", usage: "open the call form of this tool of the open MCP request before the screenshot, once its server announced it (with -send)", step: toolForm},
}

// sceneFlags holds the values the scene flags were given, per flag name.
type sceneFlags struct {
	values map[string][]string
}

// newSceneFlags defines the scene flags of sceneFlagTable on set.
func newSceneFlags(set *flag.FlagSet) *sceneFlags {
	scene := &sceneFlags{values: map[string][]string{}}
	for _, entry := range sceneFlagTable {
		scene.define(set, entry)
	}
	return scene
}

func (scene *sceneFlags) define(set *flag.FlagSet, entry sceneFlag) {
	if entry.toggle {
		set.BoolFunc(entry.name, entry.usage, func(text string) error {
			on, err := strconv.ParseBool(text)
			if err != nil {
				return err
			}
			delete(scene.values, entry.name)
			if on {
				scene.values[entry.name] = []string{""}
			}
			return nil
		})
		return
	}
	set.Func(entry.name, entry.usage, func(text string) error {
		if text == "" {
			return nil // an empty text asks for nothing, as an unset flag
		}
		if entry.repeat {
			scene.values[entry.name] = append(scene.values[entry.name], text)
		} else {
			scene.values[entry.name] = []string{text}
		}
		return nil
	})
}

// steps is the scene the flags describe, in the order of sceneFlagTable.
func (scene *sceneFlags) steps() []Step {
	var steps []Step
	for _, entry := range sceneFlagTable {
		for _, value := range scene.values[entry.name] {
			steps = append(steps, entry.step(value))
		}
	}
	return steps
}
