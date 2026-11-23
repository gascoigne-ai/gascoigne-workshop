package overlays

import "github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog"

// Tick runs the per-frame work after the view was built: every PollTicks
// frames the settings file is checked for a change made outside the program.
// It reports whether the view must be built again, which a poll always says,
// since a change replaces the settings and the raw text.
func (c *Component) Tick() bool {
	if c.deps.Ctx.Tick%settingsdialog.PollTicks != 0 {
		return false
	}
	c.Settings.Poll()
	return true
}
