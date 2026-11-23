// Package urlbar is the bar above the request pane: [protocol | method |
// URL field ........] [Send]. It owns the URL field of every open tab, cycles
// the protocol and the method, and claims the keys its field is typed with
// (Undo and Redo, and the plain keys no shortcut may take). The button is one
// of three: Send, Connect or Disconnect on a socket, and Cancel while a
// request runs. What a send, a connection, a gRPC method pick or a protocol
// change does, and how the URL is colored and explained, is asked of the
// application through Deps.
//
// State is one field per tab, in a map pruned by Tick, not one field for the
// current tab: switching tabs must keep each caret, selection and undo
// history. A field is made on first use from the request's URL, as a tab
// opens with it. The Params table also changes the URL; it calls Reload,
// which starts the field over with the request's URL, undo included, as it
// did before the bar was a component. Resyncing the field in Tick whenever
// its text differs from the request was rejected: any other edit of the URL
// would then reset the field, which is a change of behavior. Revisit the map
// when tabs get stable ids, since it keys the widget state by tab pointer.
//
// Whether a tab streams (gRPC) and the pane indexes stay in the window with
// the other state of the protocol panes, and are asked through Deps.
package urlbar

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// methodOrder is cycled by clicking the method chip; the protocol chip
// cycles the protocols the application offers.
var methodOrder = []string{"GET", "POST", "PUT", "PATCH", "DELETE"}

// next returns the entry after cur in order, wrapping; the first entry
// when cur is not listed.
func next[T comparable](order []T, cur T) T {
	for i, entry := range order {
		if entry == cur {
			return order[(i+1)%len(order)]
		}
	}
	return order[0]
}

// Component is the URL bar and the URL fields of the open tabs.
type Component struct {
	deps   Deps
	fields map[*tabs.Tab]*widget.TextField
}

// New returns a bar asking deps.
func New(deps Deps) *Component {
	return &Component{deps: deps, fields: map[*tabs.Tab]*widget.TextField{}}
}

// Field is the URL field of tab, made from the request's URL on first use.
// The window reads it for the variable completion and its tests set it up.
func (c *Component) Field(tab *tabs.Tab) *widget.TextField {
	field, ok := c.fields[tab]
	if !ok {
		made := widget.NewTextField(tab.Req.URL)
		field = &made
		c.fields[tab] = field
	}
	return field
}

// Reload starts the field of tab over with the request's URL, undo history
// included. It keeps the field in place, so the view's pointer stays valid.
func (c *Component) Reload(tab *tabs.Tab) { *c.Field(tab) = widget.NewTextField(tab.Req.URL) }

// Edited keeps the request and its Params table in step with the field.
func (c *Component) Edited(tab *tabs.Tab) {
	c.edit(tab, requestedit.URLTyped{Text: c.Field(tab).Text})
}

// Type inserts text at the caret of the field of tab, as typing would.
func (c *Component) Type(tab *tabs.Tab, text string) {
	c.Field(tab).Insert(text)
	c.Edited(tab)
}

// CycleProtocol gives tab the next protocol the application offers.
func (c *Component) CycleProtocol(tab *tabs.Tab) {
	offered := c.deps.Protocols.Offered(tab.Req.Protocol)
	c.deps.Protocols.Choose(tab, next(offered, tab.Req.Protocol))
}

// CycleMethod gives tab the next HTTP method of methodOrder.
func (c *Component) CycleMethod(tab *tabs.Tab) {
	c.edit(tab, requestedit.MethodChosen{Method: next(methodOrder, tab.Req.Method)})
}

// edit applies an edit to the request of tab, which is the collection's.
func (c *Component) edit(tab *tabs.Tab, msg requestedit.Msg) {
	*tab.Req = requestedit.Update(*tab.Req, msg)
}
