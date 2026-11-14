package ui

import (
	"embed"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/controller/editor"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// authChoice is one option of a chooser field.
type authChoice struct{ id, label string }

// authField is one input of an auth type: a text field, or a chooser
// when choices are given. id names the field's place in the request's Auth,
// which the editor package knows.
type authField struct {
	id, label string
	secret    bool
	choices   []authChoice
}

// authType is a way to authenticate and the fields it needs. The table
// is the whole definition: the pane and the stored values follow it.
type authType struct {
	id, label string
	width     int
	fields    []authField
}

var authTypes = []authType{
	{id: model.AuthNone, label: "None", width: 64},
	{id: model.AuthBearer, label: "Bearer", width: 72, fields: []authField{
		{id: "token", label: "Token", secret: true},
	}},
	{id: model.AuthBasic, label: "Basic", width: 64, fields: []authField{
		{id: "user", label: "User"},
		{id: "password", label: "Password", secret: true},
	}},
	{id: model.AuthAPIKey, label: "API key", width: 80, fields: []authField{
		{id: "keyname", label: "Name"},
		{id: "keyvalue", label: "Value", secret: true},
		{id: "keyin", label: "Add to",
			choices: []authChoice{{model.KeyInHeader, "Header"}, {model.KeyInQuery, "Query"}}},
	}},
}

// authTypeOf is the table entry for an auth type id ("" is none).
func authTypeOf(id string) authType {
	for _, t := range authTypes {
		if t.id == id {
			return t
		}
	}
	return authTypes[0]
}

// authFieldByID finds a field in any type.
func authFieldByID(id string) (authField, bool) {
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
func (t *reqTab) authInput(id string) *TextField {
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
		t.auth = map[string]*TextField{}
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

//go:embed authpane.vml
var authTemplates embed.FS

var authPaneTemplate = markup.MustLoad(authTemplates, "authpane.vml")

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
	return authPaneTemplate.MustBuild(markup.Scope{
		"gutter":        w.Gutter,
		"rowH":          w.RowH,
		"scroll":        scroll,
		"types":         types,
		"none":          len(current.fields) == 0,
		"fields":        rows,
		"hasSecret":     hasSecret,
		"showSecrets":   tab.showSecrets,
		"toggleSecrets": tab.toggleSecrets,
	})
}

// authFieldRow is the view model of one field: a chooser lights the stored
// choice, or its first one while nothing is stored; a text field is masked
// while it is secret and the secrets are hidden.
func authFieldRow(tab *reqTab, field authField) authRow {
	if field.choices != nil {
		stored := editor.AuthValue(tab.Req.Auth, field.id)
		chips := make([]authChip, 0, len(field.choices))
		for _, c := range field.choices {
			pick := func() { editRequest(tab, editor.AuthFieldSet{Field: field.id, Value: c.id}) }
			lit := stored == c.id || (stored == "" && c.id == field.choices[0].id)
			chips = append(chips, authChip{Label: c.label, Width: authChoiceChipWidth, Chosen: lit, Pick: pick})
		}
		return authRow{Label: field.label, Chips: chips}
	}
	in := tab.authInput(field.id)
	in.Masked = field.secret && !tab.showSecrets
	sync := func() { editRequest(tab, editor.AuthFieldSet{Field: field.id, Value: in.Text}) }
	return authRow{Label: field.label, Input: in, Sync: sync}
}
