package ui

import (
	"embed"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/controller/editor"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

//go:embed graphql.vml
var graphQLTemplates embed.FS

var graphQLTemplate = markup.MustLoad(graphQLTemplates, "graphql.vml")

// graphQLBody is the editor of a GraphQL body: the query, the variables as
// JSON, and a button that asks the server for its schema.
func (a *App) graphQLBody(t *reqTab) Node {
	syncQuery := func() { editRequest(t, editor.BodyTyped{Text: t.body.Text}) }
	syncVars := func() { editRequest(t, editor.VarsTyped{Text: t.gqlVars.Text}) }
	status, valid := jsonStatus(t.gqlVars.Text)
	statusColor := w.Error
	if valid {
		statusColor = w.OK
	}
	t.body.Styler = nil
	operations := 0
	if t.schema != nil {
		operations = len(t.schema.Operations)
	}
	return graphQLTemplate.MustBuild(markup.Scope{
		"rowH":        w.RowH,
		"labelH":      w.LabelH,
		"hasSchema":   operations > 0,
		"schemaLabel": fmt.Sprintf("Schema (%d)", operations),
		"openSchema":  func(x, y int) { a.openMenu(x, y, a.schemaMenu(t)) },
		"loadSchema":  func() { a.loadSchema(t) },
		"query":       &t.body,
		"queryText":   &t.body.TextField,
		"syncQuery":   syncQuery,
		"hasVars":     t.gqlVars.Text != "",
		"status":      status,
		"statusColor": statusColor,
		"vars":        &t.gqlVars,
		"varsText":    &t.gqlVars.TextField,
		"syncVars":    syncVars,
	})
}

// loadSchema sends the introspection query to the tab's URL, with its
// headers and credentials, and shows the answer as the response.
func (a *App) loadSchema(t *reqTab) {
	src := t.Req.Clone()
	src.Method, src.BodyType = "POST", model.BodyGraphQL
	src.Body, src.GraphQLVars = model.IntrospectionQuery, ""
	a.dispatch(t, &src)
}

// schemaMenu lists the operations of the loaded schema; picking one
// replaces the tab's query and variables with a starter for it.
func (a *App) schemaMenu(t *reqTab) []MenuItem {
	if t.schema == nil {
		return nil
	}
	items := make([]MenuItem, 0, len(t.schema.Operations))
	for _, op := range t.schema.Operations {
		items = append(items, MenuItem{Label: op.Label, Run: func() { a.useOperation(t, op) }})
	}
	return items
}

// useOperation puts the starter query and variables of op into the tab.
func (a *App) useOperation(t *reqTab, op model.Operation) {
	t.body = NewTextArea(op.Query)
	t.gqlVars = NewTextArea(op.Variables)
	editRequest(t, editor.OperationChosen{Query: op.Query, Variables: op.Variables})
}

// rememberSchema keeps the schema when resp is an introspection answer;
// any other answer leaves what the tab already knows.
func (a *App) rememberSchema(t *reqTab, resp *model.Response) {
	if t.Req.BodyType != model.BodyGraphQL {
		return
	}
	if schema, err := model.ParseSchema(resp.Body); err == nil {
		t.schema = schema
	}
}
