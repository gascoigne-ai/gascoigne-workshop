// Package body is the Body pane of the request editor: the type chips, the
// text area of a JSON or Text body, the path of a file body, and the Schema
// pane of a gRPC tab, whose Fill body button writes into that text area.
//
// Each open tab keeps its own state (the text area, the path field, the chip
// row's scroll, the last answer about the file and the message schema), made
// from the request's Body the first time the tab asks for it. The state is a
// map from tab to a pointer: the views hold addresses into it across a frame
// and the answers of the background questions write through it, which map
// values would not allow. Nothing writes the request's Body between opening
// a tab and its first use, so making the state late shows what making it at
// open did. A closed tab's state goes in Tick, its unsaved draft with it, as
// before. Rejected: fields on the app's tab record (the pane's state would
// stay spread over the app), a slice in tab order (indexes shift when a tab
// closes), one slot for the current tab (the caret and undo history of the
// other tabs would be lost). Revisit when tabs get stable ids or a draft
// must outlive its tab.
//
// The text area is also the query editor of a GraphQL body, which
// ui/editor/graphql draws: the app answers Slots.GraphQL with its View, and
// that package asks for the area with Area through its Query port. The Schema pane lives here, not with the Proto pane, because its
// Fill body button edits the text area this package owns; the pane table
// (which panes a gRPC tab has) stays with the app. The form table is
// another component's: the app answers Slots.FormTable and the pane asks
// only while the body is a Form or Multipart one.
package body

import (
	"slices"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/format"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Component is the body pane and the body state of the open tabs.
type Component struct {
	deps   Deps
	states map[*tabs.Tab]*state
}

// state is what one tab keeps of its body between frames.
type state struct {
	area      widget.TextArea    // the JSON or Text body, and the GraphQL query
	file      widget.TextField   // the path of a file body
	fileCheck fileCheck          // what the disk said about file's path
	typesX    widget.ScrollState // sideways scroll of the body type chips
	schema    messageSchema      // the Schema pane of a gRPC tab
}

// New returns a pane asking deps.
func New(deps Deps) *Component {
	return &Component{deps: deps, states: map[*tabs.Tab]*state{}}
}

// state is the state of tab, made from the request's Body on first use.
func (c *Component) state(tab *tabs.Tab) *state {
	made, ok := c.states[tab]
	if !ok {
		made = &state{area: widget.NewTextArea(tab.Req.Body), file: widget.NewTextField(tab.Req.Body)}
		c.states[tab] = made
	}
	return made
}

// Area is the text area of tab. A new value goes in through the pointer
// (*Area(tab) = ...), never into a copy, so the views keep their address.
func (c *Component) Area(tab *tabs.Tab) *widget.TextArea { return &c.state(tab).area }

// File is the path field of tab, replaced through the pointer like Area.
func (c *Component) File(tab *tabs.Tab) *widget.TextField { return &c.state(tab).file }

// ChooseType gives the body of tab the type id, first writing a table cell
// edit in progress so the change sees the rows as they are.
func (c *Component) ChooseType(tab *tabs.Tab, id request.BodyType) {
	c.deps.Window.CommitEdit()
	state := c.state(tab)
	c.edit(tab, requestedit.BodyTypeChosen{Type: id, Text: state.area.Text, FilePath: state.file.Text})
}

// Offers reports whether id is one of the body chips.
func (c *Component) Offers(id request.BodyType) bool {
	return slices.ContainsFunc(bodyTypes, func(choice bodyChoice) bool { return choice.id == id })
}

// Replace puts text in place of the area's, as selecting all and typing it
// would, so it can be undone.
func (c *Component) Replace(tab *tabs.Tab, text string) {
	area := c.Area(tab)
	area.SelectAll()
	area.Insert(text)
	c.edit(tab, requestedit.BodyTyped{Text: area.Text})
}

// ToggleWrap switches the area between scrolling sideways and breaking long
// lines at the width of the pane, per tab like the response's Wrap.
func (c *Component) ToggleWrap(tab *tabs.Tab) { c.Area(tab).Wrap = !c.Area(tab).Wrap }

// format re-indents a JSON body. Invalid JSON is left alone and reported.
func (c *Component) format(tab *tabs.Tab) {
	out, err := format.IndentJSON(c.Area(tab).Text)
	if err != nil {
		c.deps.Window.SetStatus("Cannot format body: " + err.Error())
		return
	}
	c.Replace(tab, out)
}

// edit applies an edit to the request of tab, which is the collection's.
func (c *Component) edit(tab *tabs.Tab, msg requestedit.Msg) {
	*tab.Req = requestedit.Update(*tab.Req, msg)
}
