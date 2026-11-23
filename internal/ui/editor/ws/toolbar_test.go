package ws

import (
	"fmt"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// The toolbar template renders exactly what the Go view it replaced did,
// with and without the MCP pickers.
func TestMessagesToolbarTemplateMatchesTheGoView(t *testing.T) {
	world := newWorld(t, request.ProtocolWebSocket)
	tab := world.tab
	for _, mcp := range []bool{false, true} {
		tab.Tools, tab.Resources, tab.Prompts = nil, nil, nil
		if mcp {
			tab.Tools = []request.MCPTool{{Name: "a"}, {Name: "b"}}
			tab.Resources = []request.MCPResource{{URI: "file:///x"}}
			tab.Prompts = []request.MCPPrompt{{Name: "p"}}
		}
		goView := Row(H(w.RowH+8), PadX(w.Gutter), Gap(6), AlignItems(AlignCenter), BorderB(w.Border),
			Text(StateText(tab.State()), Small(), Muted()),
			Spacer(),
			viewChip("Auto-reconnect", tab.Reconnect, 116, func() {}),
			If(len(tab.Tools) > 0, widget.Button(fmt.Sprintf("Tools (%d)", len(tab.Tools)), MinW(84), H(w.CompactH), OnClick(func() {}), Tooltip("Pick a tool: the composer gets a call with its arguments filled in"))),
			If(len(tab.Resources) > 0, widget.Button(fmt.Sprintf("Resources (%d)", len(tab.Resources)), MinW(104), H(w.CompactH), OnClick(func() {}), Tooltip("Pick a resource: the composer gets a read of it"))),
			If(len(tab.Prompts) > 0, widget.Button(fmt.Sprintf("Prompts (%d)", len(tab.Prompts)), MinW(94), H(w.CompactH), OnClick(func() {}), Tooltip("Pick a prompt: the composer gets a request for it"))),
			widget.Button("Ping", MinW(56), H(w.CompactH), OnClick(func() {}), Tooltip("Send a ping frame")),
			widget.Button("Clear", MinW(56), H(w.CompactH), OnClick(func() {}), Tooltip("Empty the message log")),
		)
		tplView := world.pane.toolbar(world.ctx, tab)
		Layout(world.ctx, goView, 700, 36)
		Layout(world.ctx, tplView, 700, 36)
		if got, want := Dump(tplView), Dump(goView); got != want {
			t.Fatalf("mcp=%v\ntemplate:\n%s\nGo:\n%s", mcp, got, want)
		}
	}
}

// viewChip is a segment chip of the width, the way the toolbar template
// writes its Auto-reconnect chip.
func viewChip(label string, selected bool, width int, onClick func()) Node {
	return widget.SegmentChip(label, selected, W(width), HOf(w.CompactOf), OnClick(onClick))
}
