package app

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/controller/editor"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

var graphQLTemplate = markup.Must("graphql.vml", graphqlSource)

// graphQLBody is the editor of a GraphQL body: the query, the variables as
// JSON, and a button that asks the server for its schema.
func (a *App) graphQLBody(tab *reqTab) Node {
	syncQuery := func() { editRequest(tab, editor.BodyTyped{Text: tab.body.Text}) }
	syncVars := func() { editRequest(tab, editor.VarsTyped{Text: tab.gqlVars.Text}) }
	status, valid := jsonStatus(tab.gqlVars.Text)
	statusColor := w.Error
	if valid {
		statusColor = w.OK
	}
	tab.body.Styler = nil
	operations := 0
	if tab.schema != nil {
		operations = len(tab.schema.Operations)
	}
	return graphQLTemplate.MustBuild(markup.Scope{
		"rowH":        w.RowH,
		"labelH":      w.LabelH,
		"hasSchema":   operations > 0,
		"schemaLabel": fmt.Sprintf("Schema (%d)", operations),
		"openSchema":  func(posX, posY int) { a.openMenu(posX, posY, a.schemaMenu(tab)) },
		"loadSchema":  func() { a.loadSchema(tab) },
		"query":       &tab.body,
		"queryText":   &tab.body.TextField,
		"syncQuery":   syncQuery,
		"hasVars":     tab.gqlVars.Text != "",
		"status":      status,
		"statusColor": statusColor,
		"vars":        &tab.gqlVars,
		"varsText":    &tab.gqlVars.TextField,
		"syncVars":    syncVars,
	})
}

// loadSchema sends the introspection query to the tab's URL, with its
// headers and credentials, and shows the answer as the response.
func (a *App) loadSchema(tab *reqTab) {
	src := tab.Req.Clone()
	src.Method, src.BodyType = "POST", model.BodyGraphQL
	src.Body, src.GraphQLVars = model.IntrospectionQuery, ""
	a.dispatch(tab, &src)
}

// schemaMenu lists the operations of the loaded schema; picking one
// replaces the tab's query and variables with a starter for it.
func (a *App) schemaMenu(tab *reqTab) []MenuItem {
	if tab.schema == nil {
		return nil
	}
	items := make([]MenuItem, 0, len(tab.schema.Operations))
	for _, op := range tab.schema.Operations {
		items = append(items, MenuItem{Label: op.Label, Run: func() { a.useOperation(tab, op) }})
	}
	return items
}

// useOperation puts the starter query and variables of op into the tab.
func (a *App) useOperation(tab *reqTab, op model.Operation) {
	tab.body = NewTextArea(op.Query)
	tab.gqlVars = NewTextArea(op.Variables)
	editRequest(tab, editor.OperationChosen{Query: op.Query, Variables: op.Variables})
}

// rememberSchema keeps the schema when resp is an introspection answer;
// any other answer leaves what the tab already knows.
func (a *App) rememberSchema(tab *reqTab, resp *model.Response) {
	if tab.Req.BodyType != model.BodyGraphQL {
		return
	}
	if schema, err := model.ParseSchema(resp.Body); err == nil {
		tab.schema = schema
	}
}

// Templates, parsed once above.
const (
	graphqlSource = `<!-- The GraphQL body: the query with the schema buttons, the variables as
     JSON with whether they parse. -->
<col grow="1" gap="6">
  <row h="{{rowH}}" gap="6">
    <caption>Query</caption>
    <spacer/>
    <button if="hasSchema" w="104" on-click-at="openSchema" tooltip="Pick an operation: the query and variables are replaced by a starter for it">{{schemaLabel}}</button>
    <button w="104" on-click="loadSchema" tooltip="Ask the server for its types; the answer opens in the response">Load schema</button>
  </row>
  <area state="query" grow="3" bg="sunken" on-change="syncQuery" text-menu="queryText"/>
  <row h="{{labelH}}" gap="6">
    <caption>Variables (JSON)</caption>
    <spacer/>
    <text if="hasVars" small color="{{statusColor}}">{{status}}</text>
  </row>
  <area state="vars" grow="1" bg="sunken" on-change="syncVars" text-menu="varsText"/>
</col>
`
)
