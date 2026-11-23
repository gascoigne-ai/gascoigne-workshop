package ws

import (
	"fmt"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

var (
	paneTemplate    = markup.Must("messagespane.vml", paneSource)
	toolbarTemplate = markup.Must("messagestoolbar.vml", toolbarSource)
)

// stateTexts is what the connection bar says in each state, indexed by
// tabs.ConnState: a small dense enum, so an array that a test walks.
var stateTexts = [...]string{
	tabs.ConnIdle:       "Not connected",
	tabs.ConnConnecting: "Connecting...",
	tabs.ConnOpen:       "Connected",
	tabs.ConnClosing:    "Connected",
	tabs.ConnRetrying:   "Not connected",
}

// StateText is what the connection bar says in state.
func StateText(state tabs.ConnState) string { return stateTexts[state] }

// View is the log of tab with its toolbar and the composer. The response
// side's scroll is not taken: the log scrolls itself.
func (c *Component) View(ctx *w.Ctx, tab *tabs.Tab) Node {
	state := c.state(tab)
	composer := c.deps.Composers.Composer(tab)
	return sizes.Build(ctx, paneTemplate, markup.Scope{
		"tightGap":  w.TightGap,
		"gutter":    w.Gutter,
		"composerH": ctx.Sizes.Row + 2*w.LooseGap,
		"toolbar":   c.toolbar(ctx, tab),
		"log":       &state.log,
		"composer":  composer.Shown,
		"message":   &state.composer,
		"hint":      composer.HintFor(tab.Session() != nil),
		"canFormat": composer.FormatsJSON,
		"send":      func() { c.SendMessage(tab) },
		"format":    func() { c.formatMessage(tab) },
	})
}

// toolbar is the bar above the message log: the connection state, the
// auto-reconnect chip, the pickers an MCP server enabled, Ping and Clear.
func (c *Component) toolbar(ctx *w.Ctx, tab *tabs.Tab) Node {
	pick := func(menu func(*tabs.Tab) []widget.MenuItem) func() {
		return func() { c.deps.Window.Open(ctx.In.X, ctx.In.Y, menu(tab)) }
	}
	return sizes.Build(ctx, toolbarTemplate, markup.Scope{
		"toolbarH":        ctx.Sizes.Row + w.LooseGap,
		"state":           StateText(tab.State()),
		"reconnect":       tab.Reconnect,
		"toggleReconnect": func() { tab.Reconnect = !tab.Reconnect },
		"hasTools":        len(tab.Tools) > 0,
		"toolsLabel":      fmt.Sprintf("Tools (%d)", len(tab.Tools)),
		"pickTool":        pick(c.toolMenu),
		"hasResources":    len(tab.Resources) > 0,
		"resourcesLabel":  fmt.Sprintf("Resources (%d)", len(tab.Resources)),
		"pickResource":    pick(c.resourceMenu),
		"hasPrompts":      len(tab.Prompts) > 0,
		"promptsLabel":    fmt.Sprintf("Prompts (%d)", len(tab.Prompts)),
		"pickPrompt":      pick(c.promptMenu),
		"ping":            func() { c.Ping(tab) },
		"clear":           func() { c.Clear(tab) },
	})
}

// Templates, parsed once above.
const (
	paneSource = `<col grow="1">
  <!-- same top inset as the request pane, so first rows line up -->
  <spacer h="{{gutter}}"/>
  <slot name="toolbar"/>
  <code state="log" grow="1" bg="sunken"/>
  <!-- an event stream only receives, so it has no composer -->
  <row if="composer" h="{{composerH}}" pad-x="{{gutter}}" pad-y="{{looseGap}}" gap="{{tightGap}}" border-t="border">
    <field state="message" grow="1" placeholder="{{hint}}" on-enter="send" text-menu="message"/>
    <button if="canFormat" min-w="{{formatMsgMinW}}" on-click="format" tooltip="Make the message one line of JSON, or say where it is not JSON">{ }</button>
    <primary-button min-w="{{sendMsgMinW}}" on-click="send" tooltip="Send the message (Enter)">Send</primary-button>
  </row>
</col>
`
	toolbarSource = `<row h="{{toolbarH}}" pad-x="{{gutter}}" gap="{{tightGap}}" align-items="center" border-b="border">
  <text small muted>{{state}}</text>
  <spacer/>
  <segment-chip selected="{{reconnect}}" w="{{reconnectChipW}}" h="{{compactH}}" on-click="toggleReconnect">Auto-reconnect</segment-chip>
  <button if="hasTools" min-w="{{toolsBtnMinW}}" h="{{compactH}}" on-click="pickTool" tooltip="Pick a tool: the composer gets a call with its arguments filled in">{{toolsLabel}}</button>
  <button if="hasResources" min-w="{{resourcesMinW}}" h="{{compactH}}" on-click="pickResource" tooltip="Pick a resource: the composer gets a read of it">{{resourcesLabel}}</button>
  <button if="hasPrompts" min-w="{{promptsBtnMinW}}" h="{{compactH}}" on-click="pickPrompt" tooltip="Pick a prompt: the composer gets a request for it">{{promptsLabel}}</button>
  <button min-w="{{smallBtnMinW}}" h="{{compactH}}" on-click="ping" tooltip="Send a ping frame">Ping</button>
  <button min-w="{{smallBtnMinW}}" h="{{compactH}}" on-click="clear" tooltip="Empty the message log">Clear</button>
</row>
`
)
