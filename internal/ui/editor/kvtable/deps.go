package kvtable

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Deps is what the tables ask of the application, each a narrow interface
// the host implements.
type Deps struct {
	Tables  Tables
	Tabs    Tabs
	Menus   Menus
	Secrets Secrets
	Styles  Styles
}

// Tables holds the rows of every table.
type Tables interface {
	// Rows are the rows of table, edited in place.
	Rows(table logic.Table) *[]request.KV
	// Changed runs after the rows of table changed: the Params rewrite the URL.
	Changed(table logic.Table)
}

// Tabs says which tabs and which table are showing.
type Tabs interface {
	// IsOpen reports whether tab is still open.
	IsOpen(tab *tabs.Tab) bool
	// Visible is the editable table the request pane shows, if any.
	Visible() (logic.Table, bool)
}

// Menus shows a context menu at the pointer position.
type Menus interface {
	Open(posX, posY int, items []widget.MenuItem)
}

// Secrets is the secret mark of the variables of the active environment.
type Secrets interface {
	// Secret says whether row of table can be marked, and if it is marked.
	Secret(table logic.Table, row int) (secret, ok bool)
	ToggleSecret(row int)
}

// Styles colors what a cell shows.
type Styles interface {
	// CellStyler marks the {{variables}} of a cell of req, defined or not.
	CellStyler(req *request.Request) func(text string) []widget.Span
}
