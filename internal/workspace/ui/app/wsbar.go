package app

import (
	"fmt"

	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

var messagesToolbarTemplate = markup.Must("messagestoolbar.vml", messagestoolbarSource)

// messagesToolbar is the bar above the message log: the connection state,
// the auto-reconnect chip, the pickers an MCP server enabled, Ping and
// Clear.
func (a *App) messagesToolbar(tab *reqTab) Node {
	pick := func(menu func(*reqTab) []MenuItem) func() {
		return func() { a.openMenu(a.ctx.In.X, a.ctx.In.Y, menu(tab)) }
	}
	return messagesToolbarTemplate.MustBuild(markup.Scope{
		"toolbarH":        w.RowH + 8,
		"gutter":          w.Gutter,
		"compactH":        w.CompactH,
		"state":           connState(tab),
		"reconnect":       tab.Reconnect,
		"toggleReconnect": func() { tab.Reconnect = !tab.Reconnect },
		"hasTools":        len(tab.Tools) > 0,
		"toolsLabel":      fmt.Sprintf("Tools (%d)", len(tab.Tools)),
		"pickTool":        pick(a.toolMenu),
		"hasResources":    len(tab.Resources) > 0,
		"resourcesLabel":  fmt.Sprintf("Resources (%d)", len(tab.Resources)),
		"pickResource":    pick(a.resourceMenu),
		"hasPrompts":      len(tab.Prompts) > 0,
		"promptsLabel":    fmt.Sprintf("Prompts (%d)", len(tab.Prompts)),
		"pickPrompt":      pick(a.promptMenu),
		"ping":            func() { a.pingSocket(tab) },
		"clear":           func() { a.clearLog(tab) },
	})
}

// Templates, parsed once above.
const (
	messagestoolbarSource = `<row h="{{toolbarH}}" pad-x="{{gutter}}" gap="6" align-items="center" border-b="border">
  <text small muted>{{state}}</text>
  <spacer/>
  <choice selected="{{reconnect}}" w="116" h="{{compactH}}" on-click="toggleReconnect">Auto-reconnect</choice>
  <button if="hasTools" min-w="84" h="{{compactH}}" on-click="pickTool" tooltip="Pick a tool: the composer gets a call with its arguments filled in">{{toolsLabel}}</button>
  <button if="hasResources" min-w="104" h="{{compactH}}" on-click="pickResource" tooltip="Pick a resource: the composer gets a read of it">{{resourcesLabel}}</button>
  <button if="hasPrompts" min-w="94" h="{{compactH}}" on-click="pickPrompt" tooltip="Pick a prompt: the composer gets a request for it">{{promptsLabel}}</button>
  <button min-w="56" h="{{compactH}}" on-click="ping" tooltip="Send a ping frame">Ping</button>
  <button min-w="56" h="{{compactH}}" on-click="clear" tooltip="Empty the message log">Clear</button>
</row>
`
)
