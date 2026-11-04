package ui

import (
	"fmt"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/proto"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
)

// The toolbar template renders exactly what the Go view it replaced did,
// with and without the MCP pickers.
func TestMessagesToolbarTemplateMatchesTheGoView(t *testing.T) {
	a, _ := newTestApp(t)
	for _, mcp := range []bool{false, true} {
		tab := a.cur()
		tab.tools, tab.resources, tab.prompts = nil, nil, nil
		if mcp {
			tab.tools = []proto.MCPTool{{Name: "a"}, {Name: "b"}}
			tab.resources = []proto.MCPResource{{URI: "file:///x"}}
			tab.prompts = []proto.MCPPrompt{{Name: "p"}}
		}
		goView := Row(H(w.RowH+8), PadX(w.Gutter), Gap(6), AlignItems(AlignCenter), BorderB(w.Border),
			Text(connState(tab), Small(), Muted()),
			Spacer(),
			a.viewChip("Auto-reconnect", tab.reconnect, 116, func() {}),
			If(len(tab.tools) > 0, Button(fmt.Sprintf("Tools (%d)", len(tab.tools)), W(84), H(w.CompactH), OnClick(func() {}), Tooltip("Pick a tool: the composer gets a call with its arguments filled in"))),
			If(len(tab.resources) > 0, Button(fmt.Sprintf("Resources (%d)", len(tab.resources)), W(104), H(w.CompactH), OnClick(func() {}), Tooltip("Pick a resource: the composer gets a read of it"))),
			If(len(tab.prompts) > 0, Button(fmt.Sprintf("Prompts (%d)", len(tab.prompts)), W(94), H(w.CompactH), OnClick(func() {}), Tooltip("Pick a prompt: the composer gets a request for it"))),
			Button("Ping", W(56), H(w.CompactH), OnClick(func() {}), Tooltip("Send a ping frame")),
			Button("Clear", W(56), H(w.CompactH), OnClick(func() {}), Tooltip("Empty the message log")),
		)
		tplView := a.messagesToolbar(tab)
		Layout(a.ctx, goView, 700, 36)
		Layout(a.ctx, tplView, 700, 36)
		if got, want := Dump(tplView), Dump(goView); got != want {
			t.Fatalf("mcp=%v\ntemplate:\n%s\nGo:\n%s", mcp, got, want)
		}
	}
}
