package assertions

import (
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
)

// Deps is what the panes ask of the application, each a narrow interface
// the host implements.
type Deps struct {
	Tabs     Tabs
	Vars     Vars
	Status   Status
	Defaults Defaults
}

// Tabs says which tab the panes show and which are still open.
type Tabs interface {
	// Current is the tab shown, nil when none is open.
	Current() *tabs.Tab
	IsOpen(tab *tabs.Tab) bool
}

// Vars is where a value captured from a response is kept.
type Vars interface {
	// SetVar stores name as value in the active environment of the
	// collection that holds req, else in the collection's variables.
	SetVar(req *request.Request, name, value string)
}

// Status is the status line of the window.
type Status interface {
	Status() string
	SetStatus(text string)
}

// Defaults are the settings a request without its own value falls back to.
type Defaults interface {
	// DefaultTimeout is how long a request without its own timeout may run.
	DefaultTimeout() time.Duration
}
