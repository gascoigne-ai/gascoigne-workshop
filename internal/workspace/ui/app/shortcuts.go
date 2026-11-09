package ui

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/ui/shortcutpage"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
)

// shortcut is a global key binding. A binding fires only when its
// modifiers match exactly, and a plain key (one the user also types
// with) is held back while a text field has the keyboard.
type shortcut struct {
	id, name         string // id names the action in the settings file
	key              ebiten.Key
	ctrl, shift, alt bool
	run              func(a *App)
}

// shortcuts is the whole key map; the settings menu lists it.
var shortcuts = []shortcut{
	{"send", "Send request", ebiten.KeyEnter, true, false, false, (*App).sendCurrent},
	{"focus-url", "Focus URL", ebiten.KeyL, true, false, false, (*App).focusURL},
	{"new-request", "New request", ebiten.KeyT, true, false, false, (*App).newRequest},
	{"close-tab", "Close tab", ebiten.KeyW, true, false, false, func(a *App) { a.closeTab(a.active) }},
	{"next-tab", "Next tab", ebiten.KeyTab, true, false, false, func(a *App) { a.switchTab(1) }},
	{"prev-tab", "Previous tab", ebiten.KeyTab, true, true, false, func(a *App) { a.switchTab(-1) }},
	{"duplicate-row", "Duplicate row", ebiten.KeyD, true, false, false, (*App).duplicateRow},
	{"move-row-up", "Move row up", ebiten.KeyArrowUp, false, false, true, func(a *App) { a.moveRow(-1) }},
	{"move-row-down", "Move row down", ebiten.KeyArrowDown, false, false, true, func(a *App) { a.moveRow(1) }},
	{"delete-row", "Delete row", ebiten.KeyDelete, false, false, false, (*App).deleteRow},
	{"complete-variable", "Complete variable", ebiten.KeySpace, true, false, false, (*App).completeVariable},
	{"save", "Save collection", ebiten.KeyS, true, false, false, (*App).saveWorkspace},
	{"open-workspace", "Open collection folder", ebiten.KeyO, true, false, false, (*App).promptWorkspace},
	{"find", "Find in response", ebiten.KeyF, true, false, false, (*App).openFind},
	{"settings", "Settings", ebiten.KeyComma, true, false, false, func(a *App) { a.menu = true }},
}

// label is the binding as users read it, for example "Ctrl+Shift+Tab".
func (s shortcut) label() string {
	var parts []string
	if s.ctrl {
		parts = append(parts, "Ctrl")
	}
	if s.alt {
		parts = append(parts, "Alt")
	}
	if s.shift {
		parts = append(parts, "Shift")
	}
	name, ok := engine.KeyNames[s.key]
	if !ok {
		name = s.key.String()
	}
	return strings.Join(append(parts, name), "+")
}

// runShortcuts runs the bindings whose key went down this frame.
func (a *App) runShortcuts(in core.Input) {
	for _, k := range in.Keys {
		for _, s := range shortcuts {
			b := a.bindingOf(s)
			if shortcutpage.TypesText(b) && a.textFocused() {
				continue // the key is for typing
			}
			if b.Bound && b.Key == k && b.Ctrl == in.Ctrl && b.Shift == in.Shift && b.Alt == in.Alt {
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
	for _, t := range a.tabs {
		t.view.Focused, t.body.Focused = false, false
	}
	a.cur().url.Focused = true
	a.cur().url.SelectAll()
}

// switchTab activates the tab dir places away, wrapping at the ends.
func (a *App) switchTab(dir int) {
	a.active = (a.active + dir + len(a.tabs)) % len(a.tabs)
}

// closeTab closes tab i; closing the last one leaves a blank request.
func (a *App) closeTab(i int) {
	if a.edit != nil && a.edit.tab == a.tabs[i] {
		a.cancelEdit()
	}
	a.tabs = append(a.tabs[:i], a.tabs[i+1:]...)
	switch {
	case len(a.tabs) == 0:
		a.tabs = append(a.tabs, a.makeTab(&model.Request{Name: "New Request", Protocol: "rest", Method: "GET"}))
		a.active = 0
	case i < a.active:
		a.active--
	case a.active >= len(a.tabs):
		a.active = len(a.tabs) - 1
	}
}
