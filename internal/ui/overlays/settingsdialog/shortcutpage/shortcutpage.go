// Package shortcutpage is the Shortcuts page of the settings dialog: a
// table of actions with the key each has, recording of a new key, conflicts
// with other actions and reset. The application lists its actions and
// applies the changed settings through Host.
package shortcutpage

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/prefs"
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
	selected  string // the action id of the selected row, "" for none
	table     TableState
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

// override is the binding the user gave an action: stored in the settings
// and parsable. A text that does not parse is ignored, so Effective, the
// Source column and the Reset button all follow this one test.
func override(current settings.Settings, action Action) (engine.Binding, bool) {
	text, ok := current.Shortcuts[action.ID]
	if !ok {
		return engine.Binding{}, false
	}
	binding, err := engine.ParseBinding(text)
	return binding, err == nil
}

// Effective is the binding of an action: the user's override when it is
// valid, else the default.
func Effective(current settings.Settings, action Action) engine.Binding {
	if binding, ok := override(current, action); ok {
		return binding
	}
	return action.Default
}

// Where the binding of an action comes from, as the Source column says it.
const (
	SourceDefault = "Default"
	SourceUser    = "User"
)

// Source says where the binding of an action comes from: the user's
// override, or the built-in default. It is derived from the settings, so it
// can never disagree with Effective.
func Source(current settings.Settings, action Action) string {
	if _, ok := override(current, action); ok {
		return SourceUser
	}
	return SourceDefault
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

// Selected is the action id of the selected row, "" when none is.
func (p *Page) Selected() string { return p.selected }

// Start selects action id and makes the next key combination its binding.
func (p *Page) Start(id string) {
	p.selected = id
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

// visible are the actions the filter leaves, in display order.
func (p *Page) visible(host Host) []Action {
	var shown []Action
	for _, action := range host.Actions() {
		if matches(action, KeyText(host, action.ID), p.search.Text) {
			shown = append(shown, action)
		}
	}
	return shown
}

// selection is the index of the selected row among shown, -1 when none is
// selected or the filter hides it: a hidden row shows no selection, and
// arrow keys start again from the top.
func (p *Page) selection(shown []Action) int {
	return slices.IndexFunc(shown, func(action Action) bool { return action.ID == p.selected })
}

// Navigate moves the selection with the arrow keys through the rows the
// filter leaves (Down with none selected picks the first; both stop at the
// ends) and starts recording the selected row on Enter. While a key is
// being recorded the keys belong to the recording (RecordKey), so Navigate
// does nothing.
func (p *Page) Navigate(input w.Input, host Host) {
	if p.recording != "" {
		return
	}
	shown := p.visible(host)
	index := p.selection(shown)
	switch {
	case input.Pressed(ebiten.KeyArrowDown) && len(shown) > 0:
		p.selected = shown[min(index+1, len(shown)-1)].ID
	case input.Pressed(ebiten.KeyArrowUp) && index > 0:
		p.selected = shown[index-1].ID
	case input.Enter && index >= 0:
		p.Start(p.selected)
	}
}

var (
	pageTemplate   = markup.Must("shortcutpage.vml", pageSource)
	detailTemplate = markup.Must("shortcutdetail.vml", detailSource)
)

// detail is what the template shows under the selected row: its buttons and
// at most one of Conflict, Refusal and Warning.
type detail struct {
	HasButtons bool // Bound or Overridden: shows the row of buttons
	Bound      bool // shows Unbind
	Overridden bool // shows Reset
	Conflict   string
	ReplaceTip string
	Refusal    string
	Warning    string
	Unbind     func()
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

// Widths of the buttons in the detail of a row, by role.
const (
	unbindBtnW  = 72 // Unbind
	resetBtnW   = 64 // Reset
	replaceBtnW = 64 // Replace of a conflict
)

// The columns of the table: Action takes the room, the keys and the source
// the least that shows the longest key text (Ctrl+Shift+Tab) and "Default"
// at the narrowest the settings dialog gives the page.
var (
	tableColumns = []string{"Action", "Keystrokes", "Source"}
	tableWeights = []float64{4, 4, 2}
)

// Columns of the table, by position.
const (
	keystrokesCol = 1
	sourceCol     = 2
)

// View is the page: a filter field, the table of actions and the read-only
// text editing keys.
func (p *Page) View(ctx *w.Ctx, host Host) Node {
	shown := p.visible(host)
	return pageTemplate.MustBuild(scope(ctx.Sizes, markup.Scope{
		"search":   &p.search,
		"table":    p.tableNode(ctx, host, shown),
		"noMatch":  len(shown) == 0,
		"textKeys": textKeyRows(host.HostOS()),
	}))
}

// tableNode is the table of the shown actions: one row each, the selected
// one lit with a pencil and its detail under it.
func (p *Page) tableNode(ctx *w.Ctx, host Host, shown []Action) Node {
	selected := p.selection(shown)
	rows := make([]TableRow, len(shown))
	for index, action := range shown {
		rows[index] = TableRow{Cells: []string{action.Name, KeyText(host, action.ID), Source(host.Settings(), action)}}
	}
	return Table(TableSpec{
		Columns: tableColumns, Weights: tableWeights, Rows: rows, Zebra: true, State: &p.table,
		Selected:   &selected,
		OnRowClick: func(row int) { p.Start(shown[row].ID) },
		CellNode: func(row, col int) Node {
			switch col {
			case keystrokesCol:
				return p.keystrokes(host, shown[row])
			case sourceCol:
				return Text(Source(host.Settings(), shown[row]), Small(), Muted(), Ellipsis(), Grow(1))
			}
			return nil
		},
		RowLead: func(row int) Node {
			if row == selected {
				return Icon("pencil")
			}
			return nil
		},
		RowDetail: func(row int) Node {
			if row != selected {
				return nil
			}
			return p.detailNode(ctx, host, shown[row])
		},
	})
}

// keystrokes is the Keystrokes cell of an action: its keys as chips, an
// accent chip while recording, "Unbound" when it has no key.
func (p *Page) keystrokes(host Host, action Action) Node {
	switch {
	case p.recording == action.ID:
		return Row(Bg(w.Accent), Control("primary"), PadX(w.NearGap), HOf(w.CompactOf),
			Text("Press keys...", Small(), Color(w.AccentText)))
	case !Effective(host.Settings(), action).Bound:
		return Text("Unbound", Small(), Muted())
	}
	return KeyChips(KeyText(host, action.ID), AlignSelf(AlignCenter))
}

// detailNode is what the selected row shows under it.
func (p *Page) detailNode(ctx *w.Ctx, host Host, action Action) Node {
	id := action.ID
	shown := detail{
		Bound:      Effective(host.Settings(), action).Bound,
		Overridden: Source(host.Settings(), action) == SourceUser,
		Unbind:     func() { p.Clear(host, id) },
		Reset:      func() { p.Reset(host, id) },
		Replace:    func() { p.Replace(host, id) },
	}
	shown.HasButtons = shown.Bound || shown.Overridden
	if pending, ok := p.pending[id]; ok {
		shown.Conflict = fmt.Sprintf("Used by %s (%s)", pending.OtherName, w.PlatformKeyText(host.HostOS(), pending.Binding.String()))
		shown.ReplaceTip = "Unbind " + pending.OtherName + " and use this key here"
	} else if text := p.notes.Note(id); text != "" {
		shown.Refusal = text
	} else {
		shown.Warning = Warning(Effective(host.Settings(), action))
	}
	return detailTemplate.MustBuild(scope(ctx.Sizes, markup.Scope{
		"detail": shown, "unbindBtnW": unbindBtnW, "resetBtnW": resetBtnW, "replaceBtnW": replaceBtnW,
	}))
}

// Templates, parsed once above.
const (
	pageSource = `<col>
  <row h="{{rowH}}" gap="{{tightGap}}">
    <field state="search" grow="1" bg="sunken" placeholder="Filter by action or key" text-menu="search"/>
  </row>
  <spacer h="{{gutter}}"/>
  <slot name="table"/>
  <text if="noMatch" muted>No shortcut matches</text>
  <caption h="{{labelH}}">Text editing</caption>
  <row for="keys in textKeys" h="{{compactH}}" gap="{{tightGap}}">
    <text small grow="1" ellipsis>{{keys.Action}}</text>
    <text small muted>{{keys.Keys}}</text>
  </row>
</col>
`
	// at most one of the conflict, the refusal and the warning is set
	detailSource = `<col pad-x="{{cellPad}}" pad-b="{{tightGap}}" gap="{{tightGap}}">
  <row if="detail.HasButtons" h="{{compactH}}" gap="{{tightGap}}">
    <button if="detail.Bound" w="{{unbindBtnW}}" h="{{compactH}}" on-click="detail.Unbind"
            tooltip="Leave the action without a key">Unbind</button>
    <button if="detail.Overridden" w="{{resetBtnW}}" h="{{compactH}}" on-click="detail.Reset"
            tooltip="Back to the default key">Reset</button>
  </row>
  <row if="detail.Conflict" h="{{compactH}}" gap="{{tightGap}}">
    <text small color="error" ellipsis grow="1">{{detail.Conflict}}</text>
    <button w="{{replaceBtnW}}" h="{{compactH}}" on-click="detail.Replace" tooltip="{{detail.ReplaceTip}}">Replace</button>
  </row>
  <note if="detail.Refusal" tone="error">{{detail.Refusal}}</note>
  <note if="detail.Warning" tone="muted">{{detail.Warning}}</note>
</col>
`
)
