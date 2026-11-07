package ui

import (
	"fmt"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/adapter/proto"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// graphQLBody is the editor of a GraphQL body: the query, the variables as
// JSON, and a button that asks the server for its schema.
func (a *App) graphQLBody(t *reqTab) Node {
	syncQuery := func() { t.req.Body = t.body.Text }
	syncVars := func() { t.req.GraphQLVars = t.gqlVars.Text }
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
	return Col(Grow(1), Gap(6),
		Row(H(w.RowH), Gap(6),
			Caption("Query"),
			Spacer(),
			If(operations > 0, Button(fmt.Sprintf("Schema (%d)", operations), W(104),
				OnClick(func() { a.openMenu(a.ctx.In.X, a.ctx.In.Y, a.schemaMenu(t)) }), Tooltip("Pick an operation: the query and variables are replaced by a starter for it"))),
			Button("Load schema", W(104), OnClick(func() { a.loadSchema(t) }), Tooltip("Ask the server for its types; the answer opens in the response")),
		),
		Area(&t.body, Grow(3), Bg(w.Sunken), OnChange(syncQuery), a.textMenuOpt(&t.body.TextField, syncQuery)),
		Row(H(w.LabelH), Gap(6),
			Caption("Variables (JSON)"),
			Spacer(),
			If(t.gqlVars.Text != "", Text(status, Small(), Color(statusColor))),
		),
		Area(&t.gqlVars, Grow(1), Bg(w.Sunken), OnChange(syncVars), a.textMenuOpt(&t.gqlVars.TextField, syncVars)),
	)
}

// loadSchema sends the introspection query to the tab's URL, with its
// headers and credentials, and shows the answer as the response.
func (a *App) loadSchema(t *reqTab) {
	src := t.req.Clone()
	src.Method, src.BodyType = "POST", model.BodyGraphQL
	src.Body, src.GraphQLVars = proto.IntrospectionQuery, ""
	a.dispatch(t, &src)
}

// schemaMenu lists the operations of the loaded schema; picking one
// replaces the tab's query and variables with a starter for it.
func (a *App) schemaMenu(t *reqTab) []menuItem {
	if t.schema == nil {
		return nil
	}
	items := make([]menuItem, 0, len(t.schema.Operations))
	for _, op := range t.schema.Operations {
		items = append(items, menuItem{label: op.Label, run: func() { a.useOperation(t, op) }})
	}
	return items
}

// useOperation puts the starter query and variables of op into the tab.
func (a *App) useOperation(t *reqTab, op proto.Operation) {
	t.body = NewTextArea(op.Query)
	t.gqlVars = NewTextArea(op.Variables)
	t.req.Body, t.req.GraphQLVars = op.Query, op.Variables
}

// rememberSchema keeps the schema when resp is an introspection answer;
// any other answer leaves what the tab already knows.
func (a *App) rememberSchema(t *reqTab, resp *model.Response) {
	if t.req.BodyType != model.BodyGraphQL {
		return
	}
	if schema, err := proto.ParseSchema(resp.Body); err == nil {
		t.schema = schema
	}
}
