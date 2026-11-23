package sidebar

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/transfer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sidebar/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// BeginFrame ends or continues a drag at the start of a frame. While the
// button is held the drop target is forgotten, to be set again by the row
// under the pointer when the view updates; on the frame it is released,
// the drag is resolved: a click on a folder toggles it, a release over
// another row moves the node there, or copies it when Alt is held.
func (c *Component) BeginFrame(in core.Input) {
	if in.Held {
		c.tree.Clear()
		return
	}
	outcome := c.tree.Release(in.Alt)
	if outcome.Kind != logic.Drop {
		return
	}
	from, to := c.ownerOf(outcome.Node), c.ownerOf(outcome.Target)
	if from == nil || to == nil {
		return
	}
	c.deps.Drops.Drop(transfer.Drop(from, to, outcome.Node, outcome.Target, outcome.Mode))
}

// Tick runs the per-frame work after the view was built: a drag resting over
// a closed folder opens it. It reports whether the view must be built again.
func (c *Component) Tick() bool { return c.tree.Tick() }
