package ws

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/outbox"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Deps is what the pane asks of the application, each a narrow interface
// the host implements.
type Deps struct {
	Tabs      Tabs
	Outbox    Outbox
	Tools     Tools
	Window    Window
	Composers Composers
}

// Tabs says which tab the pane shows and which are still open.
type Tabs interface {
	// Current is the tab shown, nil when none is open.
	Current() *tabs.Tab
	IsOpen(tab *tabs.Tab) bool
}

// Outbox runs the sends of a tab one after the other, off the UI goroutine.
type Outbox interface {
	// Queue runs send after the sends and pings already asked for on tab.
	// The host notes a failure in the log of tab.
	Queue(tab *tabs.Tab, send outbox.Send)
}

// Tools opens the form of an MCP tool.
type Tools interface {
	// OpenToolForm shows the form of tool; its answer fills the composer of
	// tab, through SetComposer or ComposeCall.
	OpenToolForm(tab *tabs.Tab, tool request.MCPTool)
}

// Window is what the pane shows over the window.
type Window interface {
	// Open shows a menu of items at the pointer position.
	Open(posX, posY int, items []widget.MenuItem)
	Toast(text string, level severity.Level)
}

// Composers says how the Messages pane of a tab's protocol composes.
type Composers interface {
	Composer(tab *tabs.Tab) Composer
}

// Composer is how one protocol composes messages.
type Composer struct {
	Shown       bool   // the pane has a field to send from
	FormatsJSON bool   // the field offers { } to make the message one line of JSON
	Hint        string // what the field says while connected; "" is "Message"
}

// HintFor is what the field says: how to connect while it cannot send, the
// protocol's own hint or "Message" once it can.
func (composer Composer) HintFor(connected bool) string {
	if !connected {
		return "Connect to send messages"
	}
	if composer.Hint != "" {
		return composer.Hint
	}
	return "Message"
}
