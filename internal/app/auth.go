package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/editor"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// authChoice is one option of the key location chooser.
type authChoice struct {
	id    model.KeyIn
	label string
}

// authField is one input of an auth type: a text field, or the key
// location chooser when choices are given (it has no id: it is not text).
// id names the text field's place in the request's Auth, which the editor
// package knows.
type authField struct {
	id      editor.AuthField
	label   string
	secret  bool
	choices []authChoice
}

// authType is a way to authenticate and the fields it needs. The table
// is the whole definition: the pane and the stored values follow it.
type authType struct {
	id     model.AuthType
	label  string
	width  int
	fields []authField
}

var authTypes = []authType{
	{id: model.AuthNone, label: "None", width: 64},
	{id: model.AuthBearer, label: "Bearer", width: 72, fields: []authField{
		{id: editor.AuthToken, label: "Token", secret: true},
	}},
	{id: model.AuthBasic, label: "Basic", width: 64, fields: []authField{
		{id: editor.AuthUser, label: "User"},
		{id: editor.AuthPassword, label: "Password", secret: true},
	}},
	{id: model.AuthAPIKey, label: "API key", width: 80, fields: []authField{
		{id: editor.AuthKeyName, label: "Name"},
		{id: editor.AuthKeyValue, label: "Value", secret: true},
		{label: "Add to",
			choices: []authChoice{{model.KeyInHeader, "Header"}, {model.KeyInQuery, "Query"}}},
	}},
}

// authTypeOf is the table entry for an auth type id ("" is none).
func authTypeOf(id model.AuthType) authType {
	for _, t := range authTypes {
		if t.id == id {
			return t
		}
	}
	return authTypes[0]
}

// authFieldByID finds a field in any type.
func authFieldByID(id editor.AuthField) (authField, bool) {
	for _, t := range authTypes {
		for _, f := range t.fields {
			if f.id == id {
				return f, true
			}
		}
	}
	return authField{}, false
}

// authInput is the text field of an Auth pane field, created on first
// use from the request's stored value. Secret fields start masked.
func (t *reqTab) authInput(id editor.AuthField) *TextField {
	if in, ok := t.auth[id]; ok {
		return in
	}
	f, _ := authFieldByID(id)
	in := new(TextField)
	if f.id != "" {
		*in = NewTextField(editor.AuthValue(t.Req.Auth, f.id))
	}
	in.Masked = f.secret && !t.showSecrets
	if t.auth == nil {
		t.auth = map[editor.AuthField]*TextField{}
	}
	t.auth[id] = in
	return in
}

// toggleSecrets shows or hides the secret fields at once.
func (t *reqTab) toggleSecrets() {
	t.showSecrets = !t.showSecrets
	for id, in := range t.auth {
		f, _ := authFieldByID(id)
		in.Masked = f.secret && !t.showSecrets
	}
}

var authPaneTemplate = markup.Must("authpane.vml", authpaneSource)

// authChoiceChipWidth is the width of every chip of a chooser field.
const authChoiceChipWidth = 70

// authChip is what the pane's template shows of one choice: an auth type
// or a value of a chooser field.
type authChip struct {
	Label  string
	Width  int
	Chosen bool
	Pick   func()
}

// authRow is what the pane's template shows of one field of the current
// type: a chooser's chips, or the text field it edits.
type authRow struct {
	Label string
	Chips []authChip
	Input *TextField
	Sync  func()
}

// authPane is the Auth tab: a type chooser and the fields of that type.
func (a *App) authPane(_ pane, tab *reqTab, scroll *ScrollState) Node {
	current := authTypeOf(tab.Req.Auth.Type)
	types := make([]authChip, 0, len(authTypes))
	for _, at := range authTypes {
		pick := func() { editRequest(tab, editor.AuthTypeChosen{Type: at.id}) }
		types = append(types, authChip{Label: at.label, Width: at.width, Chosen: at.id == current.id, Pick: pick})
	}
	hasSecret := false
	rows := make([]authRow, 0, len(current.fields))
	for _, field := range current.fields {
		hasSecret = hasSecret || field.secret
		rows = append(rows, authFieldRow(tab, field))
	}
	return authPaneTemplate.MustBuild(a.scope(markup.Scope{
		"scroll":        scroll,
		"types":         types,
		"none":          len(current.fields) == 0,
		"fields":        rows,
		"hasSecret":     hasSecret,
		"showSecrets":   tab.showSecrets,
		"toggleSecrets": tab.toggleSecrets,
	}))
}

// authFieldRow is the view model of one field: a chooser lights the stored
// choice, or its first one while nothing is stored; a text field is masked
// while it is secret and the secrets are hidden.
func authFieldRow(tab *reqTab, field authField) authRow {
	if field.choices != nil {
		stored := tab.Req.Auth.KeyIn
		chips := make([]authChip, 0, len(field.choices))
		for _, c := range field.choices {
			pick := func() { editRequest(tab, editor.AuthKeyInChosen{In: c.id}) }
			lit := stored == c.id || (stored == "" && c.id == model.KeyInHeader)
			chips = append(chips, authChip{Label: c.label, Width: authChoiceChipWidth, Chosen: lit, Pick: pick})
		}
		return authRow{Label: field.label, Chips: chips}
	}
	in := tab.authInput(field.id)
	in.Masked = field.secret && !tab.showSecrets
	sync := func() { editRequest(tab, editor.AuthFieldSet{Field: field.id, Value: in.Text}) }
	return authRow{Label: field.label, Input: in, Sync: sync}
}

// Templates, parsed once above.
const (
	authpaneSource = `<scroll state="scroll" pad="{{gutter}}" gap="{{looseGap}}">
  <row h="{{rowH}}" gap="{{tightGap}}">
    <segment-chip for="chip in types" selected="{{chip.Chosen}}" w="{{chip.Width}}" on-click="chip.Pick">{{chip.Label}}</segment-chip>
  </row>
  <lines if="none" color="muted">No authentication</lines>
  <form-row for="row in fields" label="{{row.Label}}" label-width="120">
    <row if="row.Chips" gap="{{tightGap}}">
      <segment-chip for="chip in row.Chips" selected="{{chip.Chosen}}" w="{{chip.Width}}" on-click="chip.Pick">{{chip.Label}}</segment-chip>
    </row>
    <field if="row.Input" state="row.Input" grow="1" bg="sunken" on-change="row.Sync" text-menu="row.Input"/>
  </form-row>
  <check-row if="hasSecret" on="{{showSecrets}}" on-click="toggleSecrets">
    <text>Show secrets</text>
  </check-row>
</scroll>
`
)
