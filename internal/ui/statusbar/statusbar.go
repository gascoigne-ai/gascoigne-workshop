// Package statusbar is the line under the window: the last status, the Log
// button and the version. It holds no state of its own; the text comes from
// Deps and the two buttons call back through it.
package statusbar

// Component draws the status bar.
type Component struct{ deps Deps }

// New returns the bar reading and calling deps.
func New(deps Deps) *Component { return &Component{deps: deps} }
