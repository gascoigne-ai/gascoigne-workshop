package statusbar

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
)

var template = markup.Must("statusbar.vml", statusbarSource)

// View is the bar for the frame: the status, the Log button and the version.
func (c *Component) View(ctx *w.Ctx) Node {
	return template.MustBuild(sizes.Scope(ctx, markup.Scope{
		"statusH":     sizes.StatusH,
		"status":      c.deps.Status.Status(),
		"version":     c.deps.Version,
		"openLog":     c.deps.Log.Open,
		"openWelcome": c.deps.Welcome.OpenWelcome,
	}))
}

const statusbarSource = `<row h="{{statusH}}" pad-x="{{gutter}}" gap="{{gutter}}" bg="frame" control="statusbar" border-t="border">
  <text small muted ellipsis grow="1">{{status}}</text>
  <button w="{{wordBtnW}}" h="{{slimH}}" on-click="openLog" tooltip="Show the log of calls">Log</button>
  <row on-click="openWelcome" tooltip="What is in this version">
    <text small muted>{{version}}</text>
  </row>
</row>
`
