package tabstrip

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Deps is what the strip asks of the application, each a narrow interface
// the host implements.
type Deps struct {
	Tabs     Tabs
	Marks    Marks
	Menus    Menus
	Requests Requests
	Hints    Hints
}

// Tabs is the list of open tabs the strip shows and changes.
type Tabs interface {
	Tabs() []*tabs.Tab
	Active() int
	// Update applies msg to the list. The host runs what a change of the
	// list drags along (the widgets of new tabs, the restore of the active one).
	Update(msg tabs.Msg)
}

// Marks are what a tab shows beside its name.
type Marks interface {
	// Dirty reports whether tab differs from its saved copy.
	Dirty(tab *tabs.Tab) bool
	// Stripe is the color of the collection holding req, none for a request in no collection.
	Stripe(req *request.Request) w.Token
}

// Menus shows the context menu of a tab.
type Menus interface {
	OpenTabMenu(posX, posY, index int)
}

// Requests makes the "+" button work.
type Requests interface {
	NewRequest()
}

// Hints adds the shortcut of an action to its tooltip.
type Hints interface {
	Hint(label, action string) string
}
