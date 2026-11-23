// Package header is the banner over the main column: the focused
// collection's name, the environment chip and the settings button.
//
// The chip is a child component built from Deps.Env; the header only gives
// it a slot. The banner has no state of its own, so it has no Tick or Keys.
package header

import "github.com/gascoigne-ai/gascoigne-workshop/internal/ui/header/envchip"

// Component is the banner.
type Component struct {
	deps Deps
	chip *envchip.Component
}

// New returns a banner asking deps.
func New(deps Deps) *Component {
	return &Component{deps: deps, chip: envchip.New(deps.Env)}
}
