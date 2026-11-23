package urlbar

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Deps is what the bar asks of the application, each a narrow interface
// the host implements.
type Deps struct {
	Tabs        Tabs
	Sends       Sends
	Connections Connections
	GRPC        GRPC
	Protocols   Protocols
	Styles      Styles
	Hints       Hints
}

// Tabs says which tab the bar shows and which are still open.
type Tabs interface {
	// Current is the tab shown, nil when none is open.
	Current() *tabs.Tab
	IsOpen(tab *tabs.Tab) bool
}

// Sends starts and stops the request of a tab.
type Sends interface {
	// Send sends the request of tab; on a socket tab it opens or closes the connection.
	Send(tab *tabs.Tab)
	Cancel(tab *tabs.Tab)
	Sending(tab *tabs.Tab) bool
}

// Connections is the connection of a socket tab.
type Connections interface {
	// Socket reports whether tab keeps a connection open: its protocol always
	// does, a gRPC tab while it streams.
	Socket(tab *tabs.Tab) bool
	ToggleConnection(tab *tabs.Tab)
}

// GRPC is what the method and Stream chips of a gRPC tab do. Whether the tab
// streams is state of the window, beside its other gRPC state.
type GRPC interface {
	PickMethod(tab *tabs.Tab)
	Streaming(tab *tabs.Tab) bool
	ToggleStream(tab *tabs.Tab)
}

// Protocols are the protocols the chip cycles through.
type Protocols interface {
	// Offered are the protocols the chip cycles through for a request on
	// current: the experimental ones only when settings switch them on, except
	// the one the request already uses.
	Offered(current request.Protocol) []request.Protocol
	// Choose gives tab the protocol: a connection belongs to the protocol that
	// opened it, so it is closed first, and the pane indexes start over.
	Choose(tab *tabs.Tab, protocol request.Protocol)
}

// Styles colors and explains what the URL field shows.
type Styles interface {
	// Segments colors the {{variables}} of raw, the URL text of req: defined
	// ones as such, unresolved ones as errors.
	Segments(req *request.Request, raw string) []widget.Span
	// Tip is what the URL of tab resolves to, for the tooltip of the field; "" when it has no variable.
	Tip(tab *tabs.Tab) string
}

// Hints adds the shortcut of an action to its tooltip.
type Hints interface {
	Hint(label, action string) string
}
