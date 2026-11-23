package body

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Deps is what the body pane asks of the application, each a narrow
// interface the host implements.
type Deps struct {
	Tabs    Tabs
	Slots   Slots
	Vars    Vars
	Files   Files
	Schemas Schemas
	Window  Window
}

// Tabs says which tab the pane shows and which are still open.
type Tabs interface {
	// Current is the tab shown, nil when none is open.
	Current() *tabs.Tab
	IsOpen(tab *tabs.Tab) bool
}

// Slots are the parts of the pane another component draws.
type Slots interface {
	// FormTable is the table of form fields of tab; asked only while the
	// body is a Form or Multipart one.
	FormTable(tab *tabs.Tab, scroll *widget.ScrollState) view.Node
	// GraphQL is the editor of a GraphQL body.
	GraphQL(tab *tabs.Tab) view.Node
	// Message is a status line shown in place of the Schema pane.
	Message(text string) view.Node
}

// Vars are the variables a request is resolved with.
type Vars interface {
	VarsFor(req *request.Request) []request.KV
}

// Files is what the file body asks of the disk and the file chooser.
type Files interface {
	// Check asks whether path is a file, off the UI goroutine; answer runs
	// on it.
	Check(path string, answer func(found bool))
	// Pick lets the user choose a file; answer runs with the path chosen.
	Pick(title string, answer func(path string))
}

// Schemas describes the request message of a gRPC method.
type Schemas interface {
	// Describe asks for the message of tab's request in the background:
	// the host expands the variables and applies the timeout when it is
	// called, and answer runs on the UI goroutine.
	Describe(tab *tabs.Tab, answer func(schema request.MessageSchema, err error))
}

// Window is what the pane does to the rest of the window.
type Window interface {
	// CommitEdit writes a table cell edit in progress into its row.
	CommitEdit()
	// ShowBody makes the Body pane the one tab shows.
	ShowBody(tab *tabs.Tab)
	SetStatus(text string)
}
