// Package graphql is the GraphQL side of the request editor: the variables
// area and the Schema and Load schema buttons of a GraphQL body. The query
// is the body's own text area, which ui/editor/body keeps for JSON and Text
// bodies too, so the text and its undo history carry across a change of
// body type; this package reaches that area through Deps.Query and draws
// it. Sending the introspection query stays with the application, which
// this package reaches through Deps.
//
// Each open tab keeps its own state: the text area of the variables and the
// schema its last introspection answered. The state is a map from tab to a
// pointer: the view holds the address of the variables area across a frame,
// which a map value would not allow. A closed tab's state goes in Tick. The
// schema is state of the tab, not of the request: it is not saved and goes
// with the tab. Rejected: fields on the app's tab record (the pane's state
// would stay spread over the app), a slice in tab order (indexes shift when
// a tab closes), one slot for the current tab (the schema and the caret of
// the variables would be lost on a tab switch). Revisit when tabs get stable
// ids or a schema must outlive its tab or be saved.
//
// The variables area is made from the request's GraphQLVars the first time
// the tab asks for it. Nothing writes GraphQLVars but this pane (typing in
// the area, picking an operation), so making the state late shows what
// making it at open did. Only View and the click closures make a state, and
// Remember when it keeps a schema; Typing and Tick only read, so asking
// about a tab that never showed the pane leaves no trace. Edits reach the
// request only through requestedit.
//
// There is no Keys method: the window does not route keys to this pane, and
// claiming some (as ui/editor/body does for Undo and plain keys) would change
// which keys reach the shortcuts while the variables have focus, which a
// refactor must not do. Typing only feeds the window's text-focus check.
package graphql

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Component is the GraphQL pane and the state of the open tabs.
type Component struct {
	deps   Deps
	states map[*tabs.Tab]*state
}

// state is what one tab keeps of the pane between frames.
type state struct {
	vars   widget.TextArea // the variables of the GraphQL body
	schema *send.Schema    // what the last introspection answered; nil for none loaded
}

// New returns a component asking deps.
func New(deps Deps) *Component {
	return &Component{deps: deps, states: map[*tabs.Tab]*state{}}
}

// state is the state of tab, made on first use from its request.
func (c *Component) state(tab *tabs.Tab) *state {
	made, ok := c.states[tab]
	if !ok {
		made = &state{vars: widget.NewTextArea(tab.Req.GraphQLVars)}
		c.states[tab] = made
	}
	return made
}

// Remember keeps the schema when resp is an introspection answer for tab;
// any other answer, and any answer for a tab whose body is not GraphQL,
// leaves what the tab already knows.
func (c *Component) Remember(tab *tabs.Tab, resp *request.Response) {
	if tab.Req.BodyType != request.BodyGraphQL {
		return
	}
	schema, err := send.ParseSchema(resp.Body)
	if err != nil {
		return
	}
	c.state(tab).schema = schema
}

// editRequest applies msg to the request of tab.
func editRequest(tab *tabs.Tab, msg requestedit.Msg) {
	*tab.Req = requestedit.Update(*tab.Req, msg)
}
