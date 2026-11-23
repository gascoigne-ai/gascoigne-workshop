package graphql

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Deps is what the component asks of the application, each a narrow
// interface the host implements.
type Deps struct {
	Tabs   Tabs
	Query  Query
	Sends  Sends
	Window Window
}

// Tabs says which tab is shown and which are still open.
type Tabs interface {
	// Current is the tab shown, nil when none is.
	Current() *tabs.Tab
	IsOpen(tab *tabs.Tab) bool
}

// Query owns the text area of the query, which is the body's text area.
type Query interface {
	Area(tab *tabs.Tab) *widget.TextArea
}

// Sends sends requests for the tabs.
type Sends interface {
	// Dispatch sends req as tab's request and shows the answer as its
	// response; req is not the tab's own request, which stays as it is.
	Dispatch(tab *tabs.Tab, req request.Request)
}

// Window is what the component shows over the window.
type Window interface {
	// Open shows a menu of items at the position.
	Open(posX, posY int, items []widget.MenuItem)
}
