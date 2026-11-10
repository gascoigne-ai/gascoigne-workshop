package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/ui/shortcutpage"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
)

// shortcut is a global key binding. A binding fires only when its
// modifiers match exactly, and a plain key (one the user also types
// with) is held back while a text field has the keyboard.
type shortcut struct {
	id, name string         // id names the action in the settings file
	def      engine.Binding // the binding until the user rebinds it
	run      func(a *App)
}

// shortcuts is the whole key map; the settings menu lists it.
var shortcuts = []shortcut{
	{"send", "Send request", keys(ebiten.KeyEnter, true, false, false), (*App).sendCurrent},
	{"focus-url", "Focus URL", keys(ebiten.KeyL, true, false, false), (*App).focusURL},
	{"new-request", "New request", keys(ebiten.KeyT, true, false, false), (*App).newRequest},
	{"close-tab", "Close tab", keys(ebiten.KeyW, true, false, false), func(a *App) { a.closeTab(a.list.Active) }},
	{"next-tab", "Next tab", keys(ebiten.KeyTab, true, false, false), func(a *App) { a.switchTab(1) }},
	{"prev-tab", "Previous tab", keys(ebiten.KeyTab, true, true, false), func(a *App) { a.switchTab(-1) }},
	{"duplicate-row", "Duplicate row", keys(ebiten.KeyD, true, false, false), (*App).duplicateRow},
	{"move-row-up", "Move row up", keys(ebiten.KeyArrowUp, false, false, true), func(a *App) { a.moveRow(-1) }},
	{"move-row-down", "Move row down", keys(ebiten.KeyArrowDown, false, false, true), func(a *App) { a.moveRow(1) }},
	{"delete-row", "Delete row", keys(ebiten.KeyDelete, false, false, false), (*App).deleteRow},
	{"complete-variable", "Complete variable", keys(ebiten.KeySpace, true, false, false), (*App).completeVariable},
	{"save", "Save collection", keys(ebiten.KeyS, true, false, false), (*App).saveWorkspace},
	{"open-workspace", "Open collection folder", keys(ebiten.KeyO, true, false, false), (*App).promptWorkspace},
	{"find", "Find in response", keys(ebiten.KeyF, true, false, false), (*App).openFind},
	{"settings", "Settings", keys(ebiten.KeyComma, true, false, false), func(a *App) { a.menu = true }},
}

// keys is a default binding for the shortcut table.
func keys(key ebiten.Key, ctrl, shift, alt bool) engine.Binding {
	return engine.Binding{Key: key, Ctrl: ctrl, Shift: shift, Alt: alt, Bound: true}
}

// runShortcuts runs the bindings whose key went down this frame.
func (a *App) runShortcuts(in core.Input) {
	for _, k := range in.Keys {
		for _, s := range shortcuts {
			b := a.bindingOf(s)
			if shortcutpage.TypesText(b) && a.textFocused() {
				continue // the key is for typing
			}
			if b.Matches(in, k) {
				s.run(a)
			}
		}
	}
}

// sendCurrent sends the request; while a table cell is edited, Ctrl+Enter
// adds a row below instead.
func (a *App) sendCurrent() {
	if a.edit != nil {
		a.insertRowBelow()
		return
	}
	a.send(a.cur())
}

// focusURL puts the keyboard in the URL field with its text selected.
func (a *App) focusURL() {
	a.commitEdit()
	a.cur().url.Focused = true // the context takes the keyboard from whatever had it
	a.cur().url.SelectAll()
}

// switchTab activates the tab dir places away, wrapping at the ends.
func (a *App) switchTab(dir int) { a.updateTabs(tabs.Step{By: dir}) }

// closeTab closes tab i; closing the last one leaves a blank request.
func (a *App) closeTab(i int) { a.updateTabs(tabs.Close{Index: i}) }
