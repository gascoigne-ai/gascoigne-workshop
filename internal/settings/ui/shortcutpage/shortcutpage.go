// Package shortcutpage is the Shortcuts page of the settings dialog: a
// table of actions with the key each has, recording of a new key, conflicts
// with other actions and reset. The application lists its actions and
// applies the changed settings through Host.
package shortcutpage

import (
	"errors"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/controller/prefs"
	"runtime"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Action is something a key can do.
type Action struct {
	ID      string         // names the action in the settings file
	Name    string         // as users read it
	Default engine.Binding // the built-in key
}

// Host is what the page needs of the application.
type Host interface {
	// Settings are the settings in effect.
	Settings() settings.Settings
	// Send asks the application to change the settings (prefs.Edit).
	Send(msg prefs.Msg)
	// Actions are the actions that have keys, in display order.
	Actions() []Action
	// TextMenu is the right-click menu of a text field.
	TextMenu(field *TextField, changed func()) Option
}

// PendingKey is a recorded combination that another action already has,
// waiting for the user to replace that action's binding or give up.
type PendingKey struct {
	Binding   engine.Binding
	OtherID   string
	OtherName string
}

// Page is the state of the Shortcuts page.
type Page struct {
	search    TextField
	recording string
	notes     map[string]string // why a recorded key was refused, by action id
	pending   map[string]PendingKey
}

// ReservedForText are the combinations the text widgets answer.
var ReservedForText = []string{"Ctrl+A", "Ctrl+C", "Ctrl+V", "Ctrl+X", "Ctrl+Z", "Ctrl+Y", "Ctrl+Shift+Z"}

// TextKeys are the keys the text widgets answer, listed read-only next to
// the shortcuts.
var TextKeys = []struct{ Action, Keys string }{
	{"Select all", "Ctrl+A"},
	{"Copy", "Ctrl+C"},
	{"Cut", "Ctrl+X"},
	{"Paste", "Ctrl+V"},
	{"Undo", "Ctrl+Z"},
	{"Redo", "Ctrl+Y or Ctrl+Shift+Z"},
}

// Effective is the binding of an action: the user's override when it is
// valid, else the default.
func Effective(current settings.Settings, action Action) engine.Binding {
	if override, ok := current.Shortcuts[action.ID]; ok {
		if binding, err := engine.ParseBinding(override); err == nil {
			return binding
		}
	}
	return action.Default
}

func isFunctionKey(key ebiten.Key) bool { return key >= ebiten.KeyF1 && key <= ebiten.KeyF12 }

// Refuse says why a combination cannot be given to any action: the text
// widgets or the overlays need it.
func Refuse(binding engine.Binding) error {
	for _, reserved := range ReservedForText {
		if parsed, _ := engine.ParseBinding(reserved); parsed == binding {
			return fmt.Errorf("%s belongs to text editing", binding)
		}
	}
	if binding.Key == ebiten.KeyEscape && !binding.Ctrl && !binding.Alt {
		return errors.New("Escape closes menus and dialogs")
	}
	return nil
}

// TypesText reports whether a binding is a key the user also types with,
// so it fires only while no text field has focus.
func TypesText(binding engine.Binding) bool {
	return !binding.Ctrl && !binding.Alt && !isFunctionKey(binding.Key) && binding.Key != ebiten.KeyDelete
}

// Warning is what a user should know about a binding that is also typed.
func Warning(binding engine.Binding) string {
	if TypesText(binding) {
		return "Plain key: works only while no text field has focus"
	}
	return ""
}

// KeyText is the binding of an action as shown to users.
func KeyText(host Host, id string) string {
	for _, action := range host.Actions() {
		if action.ID != id {
			continue
		}
		if binding := Effective(host.Settings(), action); binding.Bound {
			return w.PlatformKeyText(runtime.GOOS, binding.String())
		}
		return "Unbound"
	}
	return ""
}

// Hint is KeyText for a menu item: nothing when unbound.
func Hint(host Host, id string) string {
	if text := KeyText(host, id); text != "Unbound" {
		return text
	}
	return ""
}

// Recording is the action whose key is being recorded, "" when none.
func (p *Page) Recording() string { return p.recording }

// Note is why the last recorded key for action id was refused.
func (p *Page) Note(id string) string { return p.notes[id] }

// Pending is the recorded key another action has, waiting for a choice.
func (p *Page) Pending(id string) (PendingKey, bool) {
	pending, ok := p.pending[id]
	return pending, ok
}

// PendingCount is how many actions wait for a choice.
func (p *Page) PendingCount() int { return len(p.pending) }

// Start makes the next key combination the binding of action id.
func (p *Page) Start(id string) {
	p.recording = id
	delete(p.notes, id)
	delete(p.pending, id)
}

// Clear leaves the action without a key.
func (p *Page) Clear(host Host, id string) {
	p.set(host, id, engine.Binding{})
	delete(p.notes, id)
	delete(p.pending, id)
}

// Replace gives the pending combination of action id to it and unbinds the
// action that had it.
func (p *Page) Replace(host Host, id string) {
	pending, ok := p.pending[id]
	if !ok {
		return
	}
	delete(p.pending, id)
	p.set(host, pending.OtherID, engine.Binding{})
	p.set(host, id, pending.Binding)
}

// Reset gives the action its default key back.
func (p *Page) Reset(host Host, id string) {
	host.Send(prefs.Edit{Edit: func(current *settings.Settings) { delete(current.Shortcuts, id) }})
	delete(p.notes, id)
}

// conflict is the other action that has combination binding, if any.
func conflict(host Host, id string, binding engine.Binding) (Action, bool) {
	for _, action := range host.Actions() {
		if action.ID != id && Effective(host.Settings(), action) == binding {
			return action, true
		}
	}
	return Action{}, false
}

// Check says why combination binding cannot be given to action id as it
// stands: refused outright, or taken by another action (which the page
// offers to replace).
func Check(host Host, id string, binding engine.Binding) error {
	if err := Refuse(binding); err != nil {
		return err
	}
	if other, taken := conflict(host, id, binding); taken {
		return fmt.Errorf("Already used by %s", other.Name)
	}
	return nil
}

// RecordKey takes a recorded combination from this frame's input while an
// action is recording; it reports whether the input was consumed.
func (p *Page) RecordKey(host Host, input w.Input) bool {
	id := p.recording
	if id == "" {
		return false
	}
	if binding, ok := engine.BindingFrom(input); ok {
		p.record(host, id, binding)
	}
	return true
}

func (p *Page) record(host Host, id string, binding engine.Binding) {
	plain := !binding.Ctrl && !binding.Alt && !binding.Shift
	switch {
	case plain && binding.Key == ebiten.KeyEscape:
		p.recording = ""
		delete(p.notes, id)
		return
	case plain && binding.Key == ebiten.KeyBackspace:
		p.set(host, id, engine.Binding{})
	default:
		if err := Refuse(binding); err != nil {
			if p.notes == nil {
				p.notes = map[string]string{}
			}
			p.notes[id] = err.Error()
			return
		}
		if other, taken := conflict(host, id, binding); taken {
			if p.pending == nil {
				p.pending = map[string]PendingKey{}
			}
			p.pending[id] = PendingKey{Binding: binding, OtherID: other.ID, OtherName: other.Name}
			p.recording = ""
			delete(p.notes, id)
			return
		}
		p.set(host, id, binding)
	}
	p.recording = ""
	delete(p.notes, id)
}

// set stores a binding; one equal to the default is not stored.
func (p *Page) set(host Host, id string, binding engine.Binding) {
	host.Send(prefs.Edit{Edit: func(current *settings.Settings) {
		for _, action := range host.Actions() {
			if action.ID == id && binding == action.Default {
				delete(current.Shortcuts, id)
				return
			}
		}
		if current.Shortcuts == nil {
			current.Shortcuts = map[string]string{}
		}
		current.Shortcuts[id] = binding.String()
	}})
}

// matches reports whether an action's name or its key text contains the
// search text, ignoring case.
func matches(action Action, keyText, query string) bool {
	needle := strings.ToLower(strings.TrimSpace(query))
	return needle == "" || strings.Contains(strings.ToLower(action.Name), needle) || strings.Contains(strings.ToLower(keyText), needle)
}

// View is the page: a search field and a table of actions.
func (p *Page) View(host Host) Node {
	var rows []Item
	for _, action := range host.Actions() {
		if matches(action, KeyText(host, action.ID), p.search.Text) {
			rows = append(rows, p.row(host, action))
		}
	}
	if len(rows) == 0 {
		rows = append(rows, Text("No shortcut matches", Muted()))
	}
	return Col(
		Row(H(w.RowH), Gap(6),
			Field(&p.search, Grow(1), Bg(w.Sunken), Placeholder("Search by action or key"), host.TextMenu(&p.search, nil)),
		),
		Spacer(H(w.Gutter)),
		Col(rows...),
		Caption("Text editing", H(w.LabelH)),
		For(TextKeys, func(_ int, row struct{ Action, Keys string }) Node {
			return Row(H(w.CompactH), Gap(6),
				Text(row.Action, Small(), Grow(1), Ellipsis()),
				Text(w.PlatformKeyText(runtime.GOOS, row.Keys), Small(), Muted()),
			)
		}),
	)
}

func (p *Page) row(host Host, action Action) Node {
	_, overridden := host.Settings().Shortcuts[action.ID]
	binding := Effective(host.Settings(), action)
	label, button := KeyText(host, action.ID), Button
	if p.recording == action.ID {
		label, button = "Press keys...", PrimaryButton
	}
	var note Node
	if pending, ok := p.pending[action.ID]; ok {
		note = Row(H(w.CompactH), Gap(6),
			Text(fmt.Sprintf("Used by %s (%s)", pending.OtherName, w.PlatformKeyText(runtime.GOOS, pending.Binding.String())), Small(), Color(w.Error), Ellipsis(), Grow(1)),
			Button("Replace", W(64), H(w.CompactH), OnClick(func() { p.Replace(host, action.ID) }), Tooltip("Unbind "+pending.OtherName+" and use this key here")),
		)
	} else if text := p.notes[action.ID]; text != "" {
		note = Row(H(w.NoteH), Text(text, Small(), Color(w.Error), Ellipsis(), Grow(1)))
	} else if warning := Warning(binding); warning != "" {
		note = Row(H(w.NoteH), Text(warning, Small(), Color(w.Muted), Ellipsis(), Grow(1)))
	}
	return Col(
		Row(H(w.CompactH), Gap(6),
			Text(action.Name, Small(), Grow(1), Ellipsis()),
			button(label, W(96), H(w.CompactH), OnClick(func() { p.Start(action.ID) }), Tooltip("Click, then press the keys")),
			If(binding.Bound, Button("Clear", W(44), H(w.CompactH), OnClick(func() { p.Clear(host, action.ID) }), Tooltip("Leave the action without a key"))),
			If(overridden, Button("Reset", W(44), H(w.CompactH), OnClick(func() { p.Reset(host, action.ID) }), Tooltip("Back to the default key"))),
		),
		note,
	)
}
