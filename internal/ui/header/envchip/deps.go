package envchip

import (
	envs "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/environments"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Deps is what the chip asks of the application, each a narrow interface
// the host implements.
type Deps struct {
	Envs    Envs
	Menus   Menus
	Dialogs Dialogs
}

// Envs are the environments of the focused collection and what changes
// them. The host commits pending edits and remembers the choice; the chip
// only says which one.
type Envs interface {
	// Focused is the focused collection and its environment choice; ok is
	// false when no collection is open.
	Focused() (col *request.Collection, selection envs.Selection, ok bool)
	// Select makes environment index the active one, -1 for none.
	Select(index int)
	Add(name string)
	Rename(index int, name string)
	Delete(index int)
}

// Menus shows a context menu at the pointer.
type Menus interface {
	Open(posX, posY int, items []widget.MenuItem)
}

// Dialogs asks the user for a name or a yes.
type Dialogs interface {
	// Prompt asks for a line of text, starting with initial selected.
	Prompt(title, initial, okLabel string, onOK func(text string))
	// Confirm asks a yes/no question; onOK runs only on the primary button.
	Confirm(title, body, okLabel string, onOK func())
}
