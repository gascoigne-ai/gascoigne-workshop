// Package shortcutpage is the Shortcuts page of the settings dialog: a
// table of actions with the key each has, recording of a new key, conflicts
// with other actions and reset. The application lists its actions and
// applies the changed settings through Host.
package shortcutpage

import (
	"errors"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/controller/prefs"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
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
	// HostOS is the operating system the key text follows.
	HostOS() string
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
	notes     NotedFields[string] // why a recorded key was refused, by action id
	pending   map[string]PendingKey
}

// textKey is a text editing action and the combinations the text widgets
// answer for it. The table is both what Refuse keeps from other actions and
// what the page lists read-only next to the shortcuts.
type textKey struct {
	action string
	keys   []string
}

var textKeys = []textKey{
	{"Select all", []string{"Ctrl+A"}},
	{"Copy", []string{"Ctrl+C"}},
	{"Cut", []string{"Ctrl+X"}},
	{"Paste", []string{"Ctrl+V"}},
	{"Undo", []string{"Ctrl+Z"}},
	{"Redo", []string{"Ctrl+Y", "Ctrl+Shift+Z"}},
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

// Refuse says why a combination cannot be given to any action: the text
// widgets or the overlays need it.
func Refuse(binding engine.Binding) error {
	for _, entry := range textKeys {
		for _, keys := range entry.keys {
			if parsed, _ := engine.ParseBinding(keys); parsed == binding {
				return fmt.Errorf("%s belongs to text editing", binding)
			}
		}
	}
	if binding.Key == ebiten.KeyEscape && !binding.Ctrl && !binding.Alt {
		return errors.New("Escape closes menus and dialogs")
	}
	return nil
}

// Warning is what a user should know about a binding that is also typed;
// an unbound action has no key to warn about.
func Warning(binding engine.Binding) string {
	if binding.Bound && binding.TypesText() {
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
			return w.PlatformKeyText(host.HostOS(), binding.String())
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
func (p *Page) Note(id string) string { return p.notes.Note(id) }

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
	p.notes.SetNote(id, "")
	delete(p.pending, id)
}

// Clear leaves the action without a key.
func (p *Page) Clear(host Host, id string) {
	p.set(host, id, engine.Binding{})
	p.notes.SetNote(id, "")
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
	p.notes.SetNote(id, "")
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
		p.notes.SetNote(id, "")
		return
	case plain && binding.Key == ebiten.KeyBackspace:
		p.set(host, id, engine.Binding{})
	default:
		if err := Refuse(binding); err != nil {
			p.notes.SetNote(id, err.Error())
			return
		}
		if other, taken := conflict(host, id, binding); taken {
			if p.pending == nil {
				p.pending = map[string]PendingKey{}
			}
			p.pending[id] = PendingKey{Binding: binding, OtherID: other.ID, OtherName: other.Name}
			p.recording = ""
			p.notes.SetNote(id, "")
			return
		}
		p.set(host, id, binding)
	}
	p.recording = ""
	p.notes.SetNote(id, "")
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

var pageTemplate = markup.Must("shortcutpage.vml", shortcutpageSource)

// actionRow is what the template shows of one action. At most one of
// Conflict, Refusal and Warning is set: the line under the row.
type actionRow struct {
	Name       string
	Key        string // the key as users read it, or what recording asks for
	Recording  bool
	Bound      bool // shows Clear
	Overridden bool // shows the replay button; otherwise only its room
	Conflict   string
	ReplaceTip string
	Refusal    string
	Warning    string
	Record     func()
	Clear      func()
	Reset      func()
	Replace    func()
}

// textKeyRow is one read-only line of the text editing keys.
type textKeyRow struct{ Action, Keys string }

// textKeyRows are textKeys as users of this platform read them.
func textKeyRows(goos string) []textKeyRow {
	rows := make([]textKeyRow, 0, len(textKeys))
	for _, entry := range textKeys {
		rows = append(rows, textKeyRow{Action: entry.action, Keys: w.PlatformKeyText(goos, strings.Join(entry.keys, " or "))})
	}
	return rows
}

// scope is the sizes of the faces the context draws with plus the page's own
// entries, so a template holds no size as a number.
func scope(sizes w.Sizes, own markup.Scope) markup.Scope {
	merged := sizes.Scope()
	for name, value := range own {
		merged[name] = value
	}
	return merged
}

// Widths of the buttons of an action row, by role.
const (
	keyChipW    = 120 // the chip that shows and records the key
	replaceBtnW = 64  // the Replace button of a conflict
)

// View is the page: a search field and a table of actions.
func (p *Page) View(ctx *w.Ctx, host Host) Node {
	var actions []actionRow
	for _, action := range host.Actions() {
		if matches(action, KeyText(host, action.ID), p.search.Text) {
			actions = append(actions, p.row(host, action))
		}
	}
	return pageTemplate.MustBuild(scope(ctx.Sizes, markup.Scope{
		"search":   &p.search,
		"actions":  actions,
		"noMatch":  len(actions) == 0,
		"textKeys": textKeyRows(host.HostOS()),
		"keyChipW": keyChipW, "replaceBtnW": replaceBtnW,
	}))
}

func (p *Page) row(host Host, action Action) actionRow {
	_, overridden := host.Settings().Shortcuts[action.ID]
	binding := Effective(host.Settings(), action)
	row := actionRow{
		Name:       action.Name,
		Key:        KeyText(host, action.ID),
		Recording:  p.recording == action.ID,
		Bound:      binding.Bound,
		Overridden: overridden,
		Record:     func() { p.Start(action.ID) },
		Clear:      func() { p.Clear(host, action.ID) },
		Reset:      func() { p.Reset(host, action.ID) },
		Replace:    func() { p.Replace(host, action.ID) },
	}
	if row.Recording {
		row.Key = "Press keys..."
	}
	if pending, ok := p.pending[action.ID]; ok {
		row.Conflict = fmt.Sprintf("Used by %s (%s)", pending.OtherName, w.PlatformKeyText(host.HostOS(), pending.Binding.String()))
		row.ReplaceTip = "Unbind " + pending.OtherName + " and use this key here"
	} else if text := p.notes.Note(action.ID); text != "" {
		row.Refusal = text
	} else {
		row.Warning = Warning(binding)
	}
	return row
}

// Templates, parsed once above.
const (
	shortcutpageSource = `<col>
  <row h="{{rowH}}" gap="{{tightGap}}">
    <field state="search" grow="1" bg="sunken" placeholder="Search by action or key" text-menu="search"/>
  </row>
  <spacer h="{{gutter}}"/>
  <col>
    <col for="action in actions">
      <row h="{{compactH}}" gap="{{tightGap}}">
        <text small grow="1" ellipsis>{{action.Name}}</text>
        <segment-chip selected="{{action.Recording}}" w="{{keyChipW}}" h="{{compactH}}" on-click="action.Record"
                tooltip="Click, then press the keys">{{action.Key}}</segment-chip>
        <icon-button if="action.Bound" icon="unlink" w="{{iconBtnW}}" h="{{compactH}}" on-click="action.Clear"
                tooltip="Leave the action without a key"/>
        <spacer if="!action.Bound" w="{{iconBtnW}}"/>
        <icon-button if="action.Overridden" icon="replay" w="{{iconBtnW}}" h="{{compactH}}" on-click="action.Reset"
                tooltip="Back to the default key"/>
        <spacer if="!action.Overridden" w="{{iconBtnW}}"/>
      </row>
      <!-- at most one of the conflict, the refusal and the warning is set -->
      <row if="action.Conflict" h="{{compactH}}" gap="{{tightGap}}">
        <text small color="error" ellipsis grow="1">{{action.Conflict}}</text>
        <button w="{{replaceBtnW}}" h="{{compactH}}" on-click="action.Replace" tooltip="{{action.ReplaceTip}}">Replace</button>
      </row>
      <note if="action.Refusal" tone="error">{{action.Refusal}}</note>
      <note if="action.Warning" tone="muted">{{action.Warning}}</note>
    </col>
    <text if="noMatch" muted>No shortcut matches</text>
  </col>
  <caption h="{{labelH}}">Text editing</caption>
  <row for="keys in textKeys" h="{{compactH}}" gap="{{tightGap}}">
    <text small grow="1" ellipsis>{{keys.Action}}</text>
    <text small muted>{{keys.Keys}}</text>
  </row>
</col>
`
)
