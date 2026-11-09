package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/ui/shortcutpage"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
)

// action is the shortcut as the Shortcuts page knows it.
func (s shortcut) action() shortcutpage.Action {
	return shortcutpage.Action{ID: s.id, Name: s.name, Default: engine.Binding{Key: s.key, Ctrl: s.ctrl, Shift: s.shift, Alt: s.alt, Bound: true}}
}

// bindingOf is the effective binding of a shortcut: the user's override
// when it is valid, else the default.
func (a *App) bindingOf(s shortcut) engine.Binding {
	return shortcutpage.Effective(a.settings, s.action())
}

// shortcutHint is the key of an action for a menu item: nothing when unbound.
func (a *App) shortcutHint(id string) string { return shortcutpage.Hint(pageHost{a}, id) }
