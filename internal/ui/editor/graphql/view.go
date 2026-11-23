package graphql

import (
	"fmt"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/format"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

var graphQLTemplate = markup.Must("graphql.vml", graphqlSource)

// View is the editor of the GraphQL body of tab: the query, the variables as
// JSON, and a button that asks the server for its schema.
func (c *Component) View(ctx *core.Ctx, tab *tabs.Tab) Node {
	st := c.state(tab)
	query := c.deps.Query.Area(tab)
	syncQuery := func() { editRequest(tab, requestedit.BodyTyped{Text: query.Text}) }
	syncVars := func() { editRequest(tab, requestedit.VarsTyped{Text: st.vars.Text}) }
	status, valid := format.JSONStatus(st.vars.Text)
	statusColor := core.Error
	if valid {
		statusColor = core.OK
	}
	query.Styler = nil
	operations := 0
	if st.schema != nil {
		operations = len(st.schema.Operations)
	}
	return graphQLTemplate.MustBuild(sizes.Scope(ctx, markup.Scope{
		"hasSchema":   operations > 0,
		"schemaLabel": fmt.Sprintf("Schema (%d)", operations),
		"openSchema":  func(posX, posY int) { c.deps.Window.Open(posX, posY, c.schemaMenu(tab, st)) },
		"loadSchema":  func() { c.loadSchema(tab) },
		"query":       query,
		"queryText":   &query.TextField,
		"syncQuery":   syncQuery,
		"hasVars":     st.vars.Text != "",
		"status":      status,
		"statusColor": statusColor,
		"vars":        &st.vars,
		"varsText":    &st.vars.TextField,
		"syncVars":    syncVars,
	}))
}

// loadSchema sends the introspection query to the tab's URL, with its
// headers and credentials, and shows the answer as the response.
func (c *Component) loadSchema(tab *tabs.Tab) {
	src := tab.Req.Clone()
	src.Method, src.BodyType = "POST", request.BodyGraphQL
	src.Body, src.GraphQLVars = send.IntrospectionQuery, ""
	c.deps.Sends.Dispatch(tab, src)
}

// schemaMenu lists the operations of the loaded schema; picking one
// replaces the tab's query and variables with a starter for it.
func (c *Component) schemaMenu(tab *tabs.Tab, st *state) []widget.MenuItem {
	if st.schema == nil {
		return nil
	}
	items := make([]widget.MenuItem, 0, len(st.schema.Operations))
	for _, op := range st.schema.Operations {
		items = append(items, widget.MenuItem{Label: op.Label, Run: func() { c.useOperation(tab, st, op) }})
	}
	return items
}

// useOperation puts the starter query and variables of op into the tab.
func (c *Component) useOperation(tab *tabs.Tab, st *state, op send.Operation) {
	*c.deps.Query.Area(tab) = widget.NewTextArea(op.Query)
	st.vars = widget.NewTextArea(op.Variables)
	editRequest(tab, requestedit.OperationChosen{Query: op.Query, Variables: op.Variables})
}

// Templates, parsed once above.
const (
	graphqlSource = `<!-- The GraphQL body: the query with the schema buttons, the variables as
     JSON with whether they parse. -->
<col grow="1" gap="{{tightGap}}">
  <row h="{{rowH}}" gap="{{tightGap}}">
    <caption>Query</caption>
    <spacer/>
    <button if="hasSchema" w="{{schemaBtnW}}" on-click-at="openSchema" tooltip="Pick an operation: the query and variables are replaced by a starter for it">{{schemaLabel}}</button>
    <button w="{{schemaBtnW}}" on-click="loadSchema" tooltip="Ask the server for its types; the answer opens in the response">Load schema</button>
  </row>
  <area state="query" grow="3" bg="sunken" on-change="syncQuery" text-menu="queryText"/>
  <row h="{{labelH}}" gap="{{tightGap}}">
    <caption>Variables (JSON)</caption>
    <spacer/>
    <text if="hasVars" small color="{{statusColor}}">{{status}}</text>
  </row>
  <area state="vars" grow="1" bg="sunken" on-change="syncVars" text-menu="varsText"/>
</col>
`
)
