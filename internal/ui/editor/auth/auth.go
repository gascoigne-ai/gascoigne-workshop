// Package auth is the Auth pane of the request editor: the auth type chips
// and the fields of the chosen type. It also owns the two things the user
// is asked about before a request does more than talk to its server: the
// trust prompt (Allowed) and the cookie jar view (ShowCookies). The jar,
// the protocol registry and the list of what is trusted stay with the
// application, which this package reaches through Deps.
//
// Each open tab keeps its own field state, made from the request's Auth the
// first time a field is asked for. The state is a map from tab to a
// pointer: the views hold addresses of the text fields across a frame,
// which map values would not allow. A field is seeded from the request only
// on creation; after that it is the edit buffer (caret, undo history) and
// every change writes back through requestedit.AuthFieldSet, so the request
// stays the one stored source of truth. A closed tab's state goes in Tick.
// Rejected: fields on the app's tab record (the pane's state would stay
// spread over the app), a slice in tab order (indexes shift when a tab
// closes), one slot for the current tab (the caret and undo history of the
// other tabs would be lost). Revisit when tabs get stable ids or the
// visibility of secrets stops being per tab.
//
// There is no Keys method: a frame that only types text is skipped by the
// window's own text-focus check, and no auth key needs claiming. The helper
// that tells a typing frame from a shortcut exists in the URL bar and the
// body already; moving it to the toolkit (for example engine.TypesOnly) is
// for the task that folds the copies, and the toolkit is not touched without
// the user's approval.
package auth

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Component is the auth pane and the field state of the open tabs.
type Component struct {
	deps   Deps
	states map[*tabs.Tab]*state
}

// state is what one tab keeps of its auth pane between frames.
type state struct {
	fields      map[requestedit.AuthField]*widget.TextField // made on first use, one per text field id
	showSecrets bool                                        // not persisted: secrets are hidden again after a restart
}

// New returns a pane asking deps.
func New(deps Deps) *Component {
	return &Component{deps: deps, states: map[*tabs.Tab]*state{}}
}

// state is the state of tab, made on first use.
func (c *Component) state(tab *tabs.Tab) *state {
	made, ok := c.states[tab]
	if !ok {
		made = &state{fields: map[requestedit.AuthField]*widget.TextField{}}
		c.states[tab] = made
	}
	return made
}

// Field is the text field of an Auth pane field of tab, created on first
// use from the request's stored value. Secret fields start masked.
func (c *Component) Field(tab *tabs.Tab, id requestedit.AuthField) *widget.TextField {
	state := c.state(tab)
	if field, ok := state.fields[id]; ok {
		return field
	}
	described, _ := authFieldByID(id)
	field := new(widget.TextField)
	if described.id != "" {
		*field = widget.NewTextField(requestedit.AuthValue(tab.Req.Auth, described.id))
	}
	field.Masked = described.secret && !state.showSecrets
	state.fields[id] = field
	return field
}

// toggleSecrets shows or hides the secret fields of tab at once.
func (c *Component) toggleSecrets(tab *tabs.Tab) {
	state := c.state(tab)
	state.showSecrets = !state.showSecrets
	for id, field := range state.fields {
		described, _ := authFieldByID(id)
		field.Masked = described.secret && !state.showSecrets
	}
}

// edit applies an edit to the request of tab, which is the collection's.
func (c *Component) edit(tab *tabs.Tab, msg requestedit.Msg) {
	*tab.Req = requestedit.Update(*tab.Req, msg)
}
