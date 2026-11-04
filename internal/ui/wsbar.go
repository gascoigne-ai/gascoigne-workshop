package ui

import (
	"fmt"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/component"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
)

// messagesToolbar is the bar above the message log, written as a template:
// the connection state, the auto-reconnect chip (a hand-built node in a
// slot), the pickers an MCP server enabled, Ping and Clear.
func (a *App) messagesToolbar(t *reqTab) Node {
	pick := func(menu func(*reqTab) []menuItem) func() {
		return func() { a.openMenu(a.ctx.In.X, a.ctx.In.Y, menu(t)) }
	}
	return component.View(`
		<row h="36" pad-x="12" gap="6" align-items="center" border-b="border">
		  <text small muted>{{state}}</text>
		  <spacer/>
		  <slot name="reconnect"/>
		  <button if="hasTools" w="84" h="24" onclick="pickTool" tip="Pick a tool: the composer gets a call with its arguments filled in">{{toolsLabel}}</button>
		  <button if="hasResources" w="104" h="24" onclick="pickResource" tip="Pick a resource: the composer gets a read of it">{{resourcesLabel}}</button>
		  <button if="hasPrompts" w="94" h="24" onclick="pickPrompt" tip="Pick a prompt: the composer gets a request for it">{{promptsLabel}}</button>
		  <button w="56" h="24" onclick="ping" tip="Send a ping frame">Ping</button>
		  <button w="56" h="24" onclick="clear" tip="Empty the message log">Clear</button>
		</row>`,
		component.Scope{
			"state":          connState(t),
			"reconnect":      a.viewChip("Auto-reconnect", t.reconnect, 116, func() { t.reconnect = !t.reconnect }),
			"hasTools":       len(t.tools) > 0,
			"toolsLabel":     fmt.Sprintf("Tools (%d)", len(t.tools)),
			"pickTool":       pick(a.toolMenu),
			"hasResources":   len(t.resources) > 0,
			"resourcesLabel": fmt.Sprintf("Resources (%d)", len(t.resources)),
			"pickResource":   pick(a.resourceMenu),
			"hasPrompts":     len(t.prompts) > 0,
			"promptsLabel":   fmt.Sprintf("Prompts (%d)", len(t.prompts)),
			"pickPrompt":     pick(a.promptMenu),
			"ping":           func() { a.pingSocket(t) },
			"clear":          func() { a.clearLog(t) },
		})
}
