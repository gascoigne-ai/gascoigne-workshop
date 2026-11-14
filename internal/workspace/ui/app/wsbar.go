package ui

import (
	"embed"
	"fmt"

	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

//go:embed messagestoolbar.vml
var messagesToolbarSource embed.FS

var messagesToolbarTemplate = markup.MustLoad(messagesToolbarSource, "messagestoolbar.vml")

// messagesToolbar is the bar above the message log: the connection state,
// the auto-reconnect chip, the pickers an MCP server enabled, Ping and
// Clear.
func (a *App) messagesToolbar(t *reqTab) Node {
	pick := func(menu func(*reqTab) []MenuItem) func() {
		return func() { a.openMenu(a.ctx.In.X, a.ctx.In.Y, menu(t)) }
	}
	return messagesToolbarTemplate.MustBuild(markup.Scope{
		"toolbarH":        w.RowH + 8,
		"gutter":          w.Gutter,
		"compactH":        w.CompactH,
		"state":           connState(t),
		"reconnect":       t.Reconnect,
		"toggleReconnect": func() { t.Reconnect = !t.Reconnect },
		"hasTools":        len(t.Tools) > 0,
		"toolsLabel":      fmt.Sprintf("Tools (%d)", len(t.Tools)),
		"pickTool":        pick(a.toolMenu),
		"hasResources":    len(t.Resources) > 0,
		"resourcesLabel":  fmt.Sprintf("Resources (%d)", len(t.Resources)),
		"pickResource":    pick(a.resourceMenu),
		"hasPrompts":      len(t.Prompts) > 0,
		"promptsLabel":    fmt.Sprintf("Prompts (%d)", len(t.Prompts)),
		"pickPrompt":      pick(a.promptMenu),
		"ping":            func() { a.pingSocket(t) },
		"clear":           func() { a.clearLog(t) },
	})
}
