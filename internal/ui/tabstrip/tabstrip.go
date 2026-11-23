// Package tabstrip is the strip of open requests above the request pane,
// with a "+" for a new one. It owns the strip's scroll offset and ends a
// tab drag when the pointer is released.
//
// One scroll slot, not a state per tab: the strip has no per-tab widget
// state, so a map pruned as tabs close would hold nothing. Revisit when a
// tab gains state of its own (rename in place, pinning).
//
// The drag lives in tabs.Set (Drag, DragOver), not here, because the
// window's grab cursor and the order of tabs read it too; the component
// only sends the messages. The tab menu is built by the application until
// a menus component exists, so the strip asks for it through Deps.Menus.
package tabstrip

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// DirtyMark is shown on tabs whose request changed since it was opened.
const DirtyMark = "•"

// Component draws the strip and keeps its scroll.
type Component struct {
	deps   Deps
	scroll widget.ScrollState
}

// New returns a strip asking deps.
func New(deps Deps) *Component { return &Component{deps: deps} }

// ScrollX is how far the strip is scrolled sideways.
func (c *Component) ScrollX() int { return c.scroll.X }

// BeginFrame ends a tab drag once no button is held.
func (c *Component) BeginFrame(in core.Input) {
	if in.Held || in.MiddleHeld {
		return
	}
	c.deps.Tabs.Update(tabs.DragEnd{})
}
