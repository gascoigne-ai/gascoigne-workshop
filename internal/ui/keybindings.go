package ui

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
)

// bindingOf is the effective binding of a shortcut: the user's override
// when it is valid, else the default.
func (a *App) bindingOf(s shortcut) engine.Binding {
	if override, ok := a.settings.Shortcuts[s.id]; ok {
		if b, err := engine.ParseBinding(override); err == nil {
			return b
		}
	}
	return engine.Binding{Key: s.key, Ctrl: s.ctrl, Shift: s.shift, Alt: s.alt, Bound: true}
}

// reservedForText are the combinations the text widgets answer.
var reservedForText = []string{"Ctrl+A", "Ctrl+C", "Ctrl+V", "Ctrl+X", "Ctrl+Z", "Ctrl+Y", "Ctrl+Shift+Z"}

func isFunctionKey(k ebiten.Key) bool { return k >= ebiten.KeyF1 && k <= ebiten.KeyF12 }

// refuseBinding says why a combination cannot be given to any action: a
// text widget answers it, or Escape closes menus and dialogs.
func refuseBinding(b engine.Binding) error {
	for _, r := range reservedForText {
		if reserved, _ := engine.ParseBinding(r); reserved == b {
			return fmt.Errorf("%s belongs to text editing", b)
		}
	}
	if b.Key == ebiten.KeyEscape && !b.Ctrl && !b.Alt {
		return errors.New("Escape closes menus and dialogs")
	}
	return nil
}

// conflictWith is the other action that has combination b, if any.
func (a *App) conflictWith(id string, b engine.Binding) (shortcut, bool) {
	for _, s := range shortcuts {
		if s.id != id && a.bindingOf(s) == b {
			return s, true
		}
	}
	return shortcut{}, false
}

// pendingKey is a recorded combination that another action already has,
// waiting for the user to replace that action's binding or record again.
type pendingKey struct {
	binding   engine.Binding
	otherID   string
	otherName string
}

// startRecording makes the next key combination the binding of action id.
func (a *App) startRecording(id string) {
	a.recording = id
	delete(a.recordNotes, id)
	delete(a.pendingKeys, id)
}

// clearBinding leaves the action without a key.
func (a *App) clearBinding(id string) {
	a.setBinding(id, engine.Binding{})
	delete(a.recordNotes, id)
	delete(a.pendingKeys, id)
}

// replaceBinding gives the pending combination of action id to it and
// unbinds the action that had it.
func (a *App) replaceBinding(id string) {
	pending, ok := a.pendingKeys[id]
	if !ok {
		return
	}
	delete(a.pendingKeys, id)
	a.setBinding(pending.otherID, engine.Binding{})
	a.setBinding(id, pending.binding)
}

// defaultBinding is the built-in binding of a shortcut.
func defaultBinding(s shortcut) engine.Binding {
	return engine.Binding{Key: s.key, Ctrl: s.ctrl, Shift: s.shift, Alt: s.alt, Bound: true}
}

// recordKey takes a recorded combination from this frame's input while
// a recording is on. It reports whether a recording is on, so nothing
// else reacts to the keys. Modifier keys alone are waited out, Escape
// cancels, Backspace unbinds, and a refused combination leaves the
// recording running with a note.
func (a *App) recordKey(in core.Input) bool {
	id := a.recording
	if id == "" {
		return false
	}
	if in.Escape && !in.Ctrl && !in.Alt {
		a.recordCombo(id, engine.Binding{Key: ebiten.KeyEscape, Bound: true})
		return true
	}
	for _, k := range in.Keys {
		if engine.IsModifierKey(k) {
			continue
		}
		a.recordCombo(id, engine.Binding{Key: k, Ctrl: in.Ctrl, Shift: in.Shift, Alt: in.Alt, Bound: true})
		break
	}
	return true
}

func (a *App) recordCombo(id string, b engine.Binding) {
	plain := !b.Ctrl && !b.Alt && !b.Shift
	switch {
	case plain && b.Key == ebiten.KeyEscape:
		a.recording = ""
		delete(a.recordNotes, id)
		return
	case plain && b.Key == ebiten.KeyBackspace:
		a.setBinding(id, engine.Binding{})
	default:
		if err := refuseBinding(b); err != nil {
			if a.recordNotes == nil {
				a.recordNotes = map[string]string{}
			}
			a.recordNotes[id] = err.Error()
			return
		}
		if other, taken := a.conflictWith(id, b); taken {
			if a.pendingKeys == nil {
				a.pendingKeys = map[string]pendingKey{}
			}
			a.pendingKeys[id] = pendingKey{binding: b, otherID: other.id, otherName: other.name}
			a.recording = ""
			delete(a.recordNotes, id)
			return
		}
		a.setBinding(id, b)
	}
	a.recording = ""
	delete(a.recordNotes, id)
}

// setBinding stores a binding; one equal to the default is not stored.
func (a *App) setBinding(id string, b engine.Binding) {
	a.saveSettings(func(s *settings.Settings) {
		for _, sc := range shortcuts {
			if sc.id == id && b == defaultBinding(sc) {
				delete(s.Shortcuts, id)
				return
			}
		}
		if s.Shortcuts == nil {
			s.Shortcuts = map[string]string{}
		}
		s.Shortcuts[id] = b.String()
	})
}

// resetBinding gives the action its default key back.
func (a *App) resetBinding(id string) {
	a.saveSettings(func(s *settings.Settings) { delete(s.Shortcuts, id) })
	delete(a.recordNotes, id)
}

// shortcutText is the binding of an action as shown to users.
func (a *App) shortcutText(id string) string {
	for _, s := range shortcuts {
		if s.id == id {
			if b := a.bindingOf(s); b.Bound {
				return core.PlatformKeyText(runtime.GOOS, b.String())
			}
			return "Unbound"
		}
	}
	return ""
}

// shortcutHint is shortcutText for a menu item: nothing when unbound.
func (a *App) shortcutHint(id string) string {
	if text := a.shortcutText(id); text != "Unbound" {
		return text
	}
	return ""
}

// textKeys are the keys the text widgets answer, listed read-only in the
// settings next to the shortcuts.
var textKeys = []struct{ action, keys string }{
	{"Select all", "Ctrl+A"},
	{"Copy", "Ctrl+C"},
	{"Cut", "Ctrl+X"},
	{"Paste", "Ctrl+V"},
	{"Undo", "Ctrl+Z"},
	{"Redo", "Ctrl+Y or Ctrl+Shift+Z"},
}

// typesText reports whether a binding is a key the user also types with
// (no Ctrl or Alt, not a function key or Delete).
func typesText(b engine.Binding) bool {
	return !b.Ctrl && !b.Alt && !isFunctionKey(b.Key) && b.Key != ebiten.KeyDelete
}

// bindingWarning is what a user should know about a binding that is
// allowed but has a catch; "" when it has none.
func bindingWarning(b engine.Binding) string {
	if typesText(b) {
		return "Plain key: works only while no text field has focus"
	}
	return ""
}
