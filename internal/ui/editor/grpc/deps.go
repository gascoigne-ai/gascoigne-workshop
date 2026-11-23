package grpc

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Deps is what the component asks of the application, each a narrow
// interface the host implements.
type Deps struct {
	Tabs        Tabs
	Methods     Methods
	Vars        Vars
	Files       Files
	Connections Connections
	Window      Window
}

// Tabs says which tabs are still open.
type Tabs interface {
	IsOpen(tab *tabs.Tab) bool
}

// Methods lists the methods of a gRPC server or of the proto files of a
// request.
type Methods interface {
	// List asks, off the UI goroutine, for the methods req can call, within
	// the timeout the host has for req. answer runs on the UI goroutine.
	List(req request.Request, answer func(methods []string, err error))
}

// Vars are the variables a request is resolved with and the folder of the
// collection it belongs to.
type Vars interface {
	VarsFor(req *request.Request) []request.KV
	// Folder is the directory of the collection that owns req, "" for none.
	Folder(req *request.Request) string
}

// Files asks the user for a file.
type Files interface {
	// Pick shows a chooser titled title; answer runs with the chosen path
	// and not at all when the user cancels.
	Pick(title string, answer func(path string))
}

// Connections ends the connection a tab holds.
type Connections interface {
	// Disconnect closes the open stream of tab or abandons the connect in
	// progress.
	Disconnect(tab *tabs.Tab)
}

// Window is what the component shows over the window.
type Window interface {
	// Open shows a menu of items at the position.
	Open(posX, posY int, items []widget.MenuItem)
	Toast(text string, level severity.Level)
	// SetStatus replaces the status line.
	SetStatus(text string)
}
