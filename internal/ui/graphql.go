package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/proto"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
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
	return Col(Grow(1), Gap(6),
		Row(H(w.RowH), Gap(6),
			Caption("Query"),
			Spacer(),
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
