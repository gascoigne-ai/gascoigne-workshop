package response

import (
	"context"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Deps is what the response panes ask of the application, each a narrow
// interface the host implements.
type Deps struct {
	Tabs     Tabs
	Slots    Slots
	Headers  HeaderTable
	Files    Files
	Exports  Exports
	Window   Window
	Hints    Hints
	Defaults Defaults
}

// Tabs says which tab the panes show and which are still open.
type Tabs interface {
	// Current is the tab shown, nil when none is open.
	Current() *tabs.Tab
	IsOpen(tab *tabs.Tab) bool
}

// Slots are the panes another component draws inside the response area.
type Slots interface {
	// Results is the Tests pane: what the assertions found, scrolled with scroll.
	Results(tab *tabs.Tab, scroll *widget.ScrollState) view.Node
}

// HeaderTable is the table of the response headers, whose column widths and
// cell colors belong to the key-value tables.
type HeaderTable interface {
	// Widths is the column widths, shared by every tab and kept with the settings of the window.
	Widths() *widget.TableState
	// CellStyler marks the {{variables}} of a cell of req.
	CellStyler(req *request.Request) func(text string) []widget.Span
}

// Files is where a body is saved.
type Files interface {
	// PickSave asks for the path of a file to save, suggested as its name.
	PickSave(title, suggested string, answer func(path string))
	// Write saves body at path off the UI goroutine and then calls done there.
	Write(path string, body []byte, done func(error))
	// Home is the home folder of the user.
	Home() (string, error)
}

// Exports turns the response and its request into what the user takes away.
type Exports interface {
	// Curl is the cURL command of the request of tab with its variables filled in.
	Curl(tab *tabs.Tab) string
	// Preview serves body for the browser to open.
	Preview(ctx context.Context, body []byte, contentType string) error
}

// Window is what the panes ask of the screen around them.
type Window interface {
	// CommitEdit writes a table cell edit in progress into its row.
	CommitEdit()
	// ShowResponse selects the Response pane of tab.
	ShowResponse(tab *tabs.Tab)
	// Open shows a context menu at the pointer position.
	Open(posX, posY int, items []widget.MenuItem)
	// Toast shows text for a moment.
	Toast(text string, level severity.Level)
}

// Hints is how an action is named with its shortcut.
type Hints interface {
	// Hint is label with the shortcut of action.
	Hint(label, action string) string
	// Shortcut is the key of action, "" when unbound.
	Shortcut(action string) string
}

// Defaults are the settings and the clock the panes fall back to.
type Defaults interface {
	// ResponseView is the setting that picks the mode of a new response view.
	ResponseView() string
	Now() time.Time
}
