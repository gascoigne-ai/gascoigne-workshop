package header

import "github.com/gascoigne-ai/gascoigne-workshop/internal/ui/header/envchip"

// Deps is what the header asks of the application, each a narrow interface
// the host implements. Env is the chip's own.
type Deps struct {
	Title    Title
	Settings Settings
	Hints    Hints
	Env      envchip.Deps
}

// Title is the name shown at the left of the banner.
type Title interface {
	// Title is the focused collection's name, "" when none is open.
	Title() string
}

// Settings is the dialog the gear opens.
type Settings interface {
	Toggle()
}

// Hints adds the shortcut of an action to its tooltip.
type Hints interface {
	Hint(label, action string) string
}
