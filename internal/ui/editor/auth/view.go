package auth

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"

	// The view DSL is dot-imported so views read like markup.
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// authChoice is one option of the key location chooser.
type authChoice struct {
	id    request.KeyIn
	label string
}

// authField is one input of an auth type: a text field, or the key
// location chooser when choices are given (it has no id: it is not text).
// id names the text field's place in the request's Auth, which the editor
// package knows.
type authField struct {
	id      requestedit.AuthField
	label   string
	secret  bool
	choices []authChoice
}

// authType is a way to authenticate and the fields it needs. The table
// is the whole definition: the pane and the stored values follow it.
type authType struct {
	id     request.AuthType
	label  string
	width  int
	fields []authField
}

var authTypes = []authType{
	{id: request.AuthNone, label: "None", width: 64},
	{id: request.AuthBearer, label: "Bearer", width: 72, fields: []authField{
		{id: requestedit.AuthToken, label: "Token", secret: true},
	}},
	{id: request.AuthBasic, label: "Basic", width: 64, fields: []authField{
		{id: requestedit.AuthUser, label: "User"},
		{id: requestedit.AuthPassword, label: "Password", secret: true},
	}},
	{id: request.AuthAPIKey, label: "API key", width: 80, fields: []authField{
		{id: requestedit.AuthKeyName, label: "Name"},
		{id: requestedit.AuthKeyValue, label: "Value", secret: true},
		{label: "Add to",
			choices: []authChoice{{request.KeyInHeader, "Header"}, {request.KeyInQuery, "Query"}}},
	}},
}

// authTypeOf is the table entry for an auth type id ("" is none).
func authTypeOf(id request.AuthType) authType {
	for _, described := range authTypes {
		if described.id == id {
			return described
		}
	}
	return authTypes[0]
}

// authFieldByID finds a field in any type.
func authFieldByID(id requestedit.AuthField) (authField, bool) {
	for _, described := range authTypes {
		for _, field := range described.fields {
			if field.id == id {
				return field, true
			}
		}
	}
	return authField{}, false
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
	Input *widget.TextField
	Sync  func()
}

// View is the Auth pane of tab: a type chooser and the fields of that type.
// scroll is the pane's scroll state.
func (c *Component) View(ctx *w.Ctx, tab *tabs.Tab, scroll *widget.ScrollState) Node {
	current := authTypeOf(tab.Req.Auth.Type)
	types := make([]authChip, 0, len(authTypes))
	for _, at := range authTypes {
		pick := func() { c.edit(tab, requestedit.AuthTypeChosen{Type: at.id}) }
		types = append(types, authChip{Label: at.label, Width: at.width, Chosen: at.id == current.id, Pick: pick})
	}
	hasSecret := false
	rows := make([]authRow, 0, len(current.fields))
	for _, field := range current.fields {
		hasSecret = hasSecret || field.secret
		rows = append(rows, c.authFieldRow(tab, field))
	}
	return sizes.Build(ctx, authPaneTemplate, markup.Scope{
		"scroll":        scroll,
		"types":         types,
		"none":          len(current.fields) == 0,
		"fields":        rows,
		"hasSecret":     hasSecret,
		"showSecrets":   c.state(tab).showSecrets,
		"toggleSecrets": func() { c.toggleSecrets(tab) },
	})
}

// authFieldRow is the view model of one field: a chooser lights the stored
// choice, or its first one while nothing is stored; a text field is masked
// while it is secret and the secrets are hidden.
func (c *Component) authFieldRow(tab *tabs.Tab, field authField) authRow {
	if field.choices != nil {
		stored := tab.Req.Auth.KeyIn
		chips := make([]authChip, 0, len(field.choices))
		for _, choice := range field.choices {
			pick := func() { c.edit(tab, requestedit.AuthKeyInChosen{In: choice.id}) }
			lit := stored == choice.id || (stored == "" && choice.id == request.KeyInHeader)
			chips = append(chips, authChip{Label: choice.label, Width: authChoiceChipWidth, Chosen: lit, Pick: pick})
		}
		return authRow{Label: field.label, Chips: chips}
	}
	input := c.Field(tab, field.id)
	input.Masked = field.secret && !c.state(tab).showSecrets
	sync := func() { c.edit(tab, requestedit.AuthFieldSet{Field: field.id, Value: input.Text}) }
	return authRow{Label: field.label, Input: input, Sync: sync}
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
