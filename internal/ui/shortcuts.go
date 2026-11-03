package ui

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	"github.com/gascoigne-ai/gascoigne-workshop/widget"
)

// shortcut is a global key binding. A binding fires only when its
// modifiers match exactly, so plain keys never trigger anything while
// typing.
type shortcut struct {
	name        string
	key         ebiten.Key
	ctrl, shift bool
	run         func(a *App)
}

// shortcuts is the whole key map; the settings menu lists it.
var shortcuts = []shortcut{
	{"Send request", ebiten.KeyEnter, true, false, (*App).sendCurrent},
	{"Focus URL", ebiten.KeyL, true, false, (*App).focusURL},
	{"New request", ebiten.KeyT, true, false, (*App).newRequest},
	{"Close tab", ebiten.KeyW, true, false, func(a *App) { a.closeTab(a.active) }},
	{"Next tab", ebiten.KeyTab, true, false, func(a *App) { a.switchTab(1) }},
	{"Previous tab", ebiten.KeyTab, true, true, func(a *App) { a.switchTab(-1) }},
	{"Settings", ebiten.KeyComma, true, false, func(a *App) { a.menu = true }},
}

// keyNames spells keys whose ebiten name is not what users read.
var keyNames = map[ebiten.Key]string{ebiten.KeyComma: ","}

// label is the binding as users read it, for example "Ctrl+Shift+Tab".
func (s shortcut) label() string {
	var parts []string
	if s.ctrl {
		parts = append(parts, "Ctrl")
	}
	if s.shift {
		parts = append(parts, "Shift")
	}
	name, ok := keyNames[s.key]
	if !ok {
		name = s.key.String()
	}
	return strings.Join(append(parts, name), "+")
}

// runShortcuts runs the bindings whose key went down this frame.
func (a *App) runShortcuts(in widget.Input) {
	for _, k := range in.Keys {
		for _, s := range shortcuts {
			if s.key == k && s.ctrl == in.Ctrl && s.shift == in.Shift {
				s.run(a)
			}
		}
	}
}

func (a *App) sendCurrent() { a.send(a.cur()) }

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
		a.tabs = append(a.tabs, newTab(&model.Request{Name: "New Request", Protocol: "rest", Method: "GET"}))
		a.active = 0
	case i < a.active:
		a.active--
	case a.active >= len(a.tabs):
		a.active = len(a.tabs) - 1
	}
}
