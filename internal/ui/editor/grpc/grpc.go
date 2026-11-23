// Package grpc is the gRPC side of the request editor: the method picker of
// the URL bar, the Stream switch and the Proto pane, where a request names
// the .proto files (and the folders their imports are searched in) its
// methods come from instead of the server's reflection. The server, the
// listing of methods and the file chooser stay with the application, which
// this package reaches through Deps. The package is named like the protocol;
// a file that also needs google.golang.org/grpc must import that one under
// another name.
//
// Each open tab keeps its own state: whether it streams, whether its file
// lists show, the text fields of the path rows and what reading its proto
// files found. The state is a map from tab to a pointer: the views hold
// addresses of the text fields across a frame, which map values would not
// allow, and a background answer must reach the tab it was asked for. A
// closed tab's state goes in Tick. Rejected: fields on the app's tab record
// (the pane's state would stay spread over the app), a slice in tab order
// (indexes shift when a tab closes), one slot for the current tab (the
// caret of a half-typed path and the answer of a read in progress would be
// lost on a tab switch). Revisit when tabs get stable ids.
//
// A background answer writes into the state pointer taken before the work
// started, never into a fresh lookup: the tab may be closed meanwhile, and a
// lookup would make a state for it again that nothing prunes. Only View,
// ToggleStream and the click closures make a state; Streaming, ProtoPending
// and Tick only read, so asking about a tab that never showed the pane
// leaves no trace. A tab without a state does not stream and has nothing
// to read unless its request names proto files.
//
// The component owns the Stream flag, not the application: it is a choice
// of the tab that is never saved, and the Stream chip, the response panes
// and the send button read it through Streaming. Rejected: keeping it on the
// app's tab record beside the connection (the host then writes a flag of a
// pane this package draws). ToggleStream asks the host to end the open
// stream first, because a stream belongs to the mode that opened it.
//
// The request stays the one source of truth for the paths: a row's text
// field is seeded from it and written back through requestedit, and it is
// made again when the request's path no longer matches the text (a removed
// row shifts the rows below it). The fields are kept in a map by list and
// row, never shrunk; a row only costs a text field. A slice per list was
// rejected because rows are removed from the middle.
//
// There is no Typing or Keys method: the path fields are not among the
// fields the window's text-focus check knows, so a shortcut typed while one
// has focus still runs today. Adding them would change which keys reach
// the shortcuts, which a refactor must not do.
package grpc

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Component is the gRPC pane and the state of the open tabs.
type Component struct {
	deps   Deps
	states map[*tabs.Tab]*state
}

// state is what one tab keeps of the pane between frames.
type state struct {
	streaming   bool                         // the tab streams instead of making single calls
	filesChosen bool                         // the file lists show before a file is added
	inputs      map[rowKey]*widget.TextField // path fields, made on first use
	dir         string                       // the folder of the tab's collection, as the pane last saw it
	check       check                        // what reading the proto files found
}

// rowKey names one path row of the Proto pane.
type rowKey struct {
	list  requestedit.ProtoList
	index int
}

// check is what reading a request's proto files in the background found,
// and for which source.
type check struct {
	source   request.ProtoSource
	problems []string
	running  bool
}

// New returns a component asking deps.
func New(deps Deps) *Component {
	return &Component{deps: deps, states: map[*tabs.Tab]*state{}}
}

// state is the state of tab, made on first use.
func (c *Component) state(tab *tabs.Tab) *state {
	made, ok := c.states[tab]
	if !ok {
		made = &state{}
		c.states[tab] = made
	}
	return made
}

// editRequest applies msg to the request of tab.
func editRequest(tab *tabs.Tab, msg requestedit.Msg) {
	*tab.Req = requestedit.Update(*tab.Req, msg)
}
