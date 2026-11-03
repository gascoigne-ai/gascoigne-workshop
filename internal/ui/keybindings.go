package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/widget"
)

// binding is a key with modifiers; an unbound action has none.
type binding struct {
	key              ebiten.Key
	ctrl, shift, alt bool
	bound            bool
}

// isModifierKey reports whether k is a modifier key itself.
func isModifierKey(k ebiten.Key) bool {
	switch k {
	case ebiten.KeyAlt, ebiten.KeyAltLeft, ebiten.KeyAltRight,
		ebiten.KeyControl, ebiten.KeyControlLeft, ebiten.KeyControlRight,
		ebiten.KeyShift, ebiten.KeyShiftLeft, ebiten.KeyShiftRight,
		ebiten.KeyMeta, ebiten.KeyMetaLeft, ebiten.KeyMetaRight:
		return true
	}
	return false
}

// keyLabel is how a key is written in key strings: the friendly name
// from keyNames, else Ebitengine's.
func keyLabel(k ebiten.Key) string {
	if name, ok := keyNames[k]; ok {
		return name
	}
	return k.String()
}

// keyByName finds a bindable key by its lower-case name, friendly or
// Ebitengine's.
var keyByName = func() map[string]ebiten.Key {
	m := map[string]ebiten.Key{}
	for k := ebiten.Key(0); k <= ebiten.KeyMax; k++ {
		if isModifierKey(k) || strings.HasPrefix(k.String(), "Key(") {
			continue
		}
		m[strings.ToLower(k.String())] = k
		m[strings.ToLower(keyLabel(k))] = k
	}
	return m
}()

// parseBinding reads "Ctrl+Shift+Tab" (modifiers in any order, names
// case-insensitive). "" is a valid, unbound binding.
func parseBinding(s string) (binding, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return binding{}, nil
	}
	parts := strings.Split(s, "+")
	var b binding
	for _, mod := range parts[:len(parts)-1] {
		flag := map[string]*bool{"ctrl": &b.ctrl, "control": &b.ctrl, "alt": &b.alt, "shift": &b.shift}[strings.ToLower(strings.TrimSpace(mod))]
		if flag == nil || *flag {
			return binding{}, fmt.Errorf("%q is not a modifier, or is repeated", mod)
		}
		*flag = true
	}
	key, ok := keyByName[strings.ToLower(strings.TrimSpace(parts[len(parts)-1]))]
	if !ok {
		return binding{}, fmt.Errorf("%q is not a key", parts[len(parts)-1])
	}
	b.key, b.bound = key, true
	return b, nil
}

// String writes the binding as users read it: modifiers in the order
// Ctrl, Alt, Shift, then the key; "" when unbound.
func (b binding) String() string {
	if !b.bound {
		return ""
	}
	var parts []string
	if b.ctrl {
		parts = append(parts, "Ctrl")
	}
	if b.alt {
		parts = append(parts, "Alt")
	}
	if b.shift {
		parts = append(parts, "Shift")
	}
	return strings.Join(append(parts, keyLabel(b.key)), "+")
}

// bindingOf is the effective binding of a shortcut: the user's override
// when it is valid, else the default.
func (a *App) bindingOf(s shortcut) binding {
	if override, ok := a.settings.Shortcuts[s.id]; ok {
		if b, err := parseBinding(override); err == nil {
			return b
		}
	}
	return binding{key: s.key, ctrl: s.ctrl, shift: s.shift, alt: s.alt, bound: true}
}

// reservedForText are the combinations the text widgets answer.
var reservedForText = []string{"Ctrl+A", "Ctrl+C", "Ctrl+V", "Ctrl+X", "Ctrl+Z", "Ctrl+Y", "Ctrl+Shift+Z"}

func isFunctionKey(k ebiten.Key) bool { return k >= ebiten.KeyF1 && k <= ebiten.KeyF12 }

// checkBinding says why a combination cannot be given to action id: it
// belongs to text editing, it is a plain key that would fire while
// typing, it closes overlays (plain Escape), or another action has it.
func (a *App) checkBinding(id string, b binding) error {
	for _, r := range reservedForText {
		if reserved, _ := parseBinding(r); reserved == b {
			return fmt.Errorf("%s belongs to text editing", b)
		}
	}
	switch {
	case b.key == ebiten.KeyEscape && !b.ctrl && !b.alt:
		return errors.New("Escape closes menus and dialogs")
	case !b.ctrl && !b.alt && !isFunctionKey(b.key) && b.key != ebiten.KeyDelete:
		return errors.New("Use Ctrl or Alt: plain keys are for typing")
	}
	for _, s := range shortcuts {
		if s.id != id && a.bindingOf(s) == b {
			return fmt.Errorf("Already used by %s", s.name)
		}
	}
	return nil
}

// startRecording makes the next key combination the binding of action id.
func (a *App) startRecording(id string) {
	a.recording = id
	delete(a.recordNotes, id)
}

// defaultBinding is the built-in binding of a shortcut.
func defaultBinding(s shortcut) binding {
	return binding{key: s.key, ctrl: s.ctrl, shift: s.shift, alt: s.alt, bound: true}
}

// recordKey takes a recorded combination from this frame's input while
// a recording is on. It reports whether a recording is on, so nothing
// else reacts to the keys. Modifier keys alone are waited out, Escape
// cancels, Backspace unbinds, and a refused combination leaves the
// recording running with a note.
func (a *App) recordKey(in widget.Input) bool {
	id := a.recording
	if id == "" {
		return false
	}
	if in.Escape && !in.Ctrl && !in.Alt {
		a.recordCombo(id, binding{key: ebiten.KeyEscape, bound: true})
		return true
	}
	for _, k := range in.Keys {
		if isModifierKey(k) {
			continue
		}
		a.recordCombo(id, binding{key: k, ctrl: in.Ctrl, shift: in.Shift, alt: in.Alt, bound: true})
		break
	}
	return true
}

func (a *App) recordCombo(id string, b binding) {
	plain := !b.ctrl && !b.alt && !b.shift
	switch {
	case plain && b.key == ebiten.KeyEscape:
		a.recording = ""
		delete(a.recordNotes, id)
		return
	case plain && b.key == ebiten.KeyBackspace:
		a.setBinding(id, binding{})
	default:
		if err := a.checkBinding(id, b); err != nil {
			if a.recordNotes == nil {
				a.recordNotes = map[string]string{}
			}
			a.recordNotes[id] = err.Error()
			return
		}
		a.setBinding(id, b)
	}
	a.recording = ""
	delete(a.recordNotes, id)
}

// setBinding stores a binding; one equal to the default is not stored.
func (a *App) setBinding(id string, b binding) {
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
			if b := a.bindingOf(s); b.bound {
				return b.String()
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
