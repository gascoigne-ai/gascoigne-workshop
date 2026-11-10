package ui

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"testing"

	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// The toolbar template renders exactly what the Go view it replaced did,
// with and without the MCP pickers.
func TestMessagesToolbarTemplateMatchesTheGoView(t *testing.T) {
	a, _ := newTestApp(t)
	for _, mcp := range []bool{false, true} {
		tab := a.cur()
		tab.Tools, tab.Resources, tab.Prompts = nil, nil, nil
		if mcp {
			tab.Tools = []model.MCPTool{{Name: "a"}, {Name: "b"}}
			tab.Resources = []model.MCPResource{{URI: "file:///x"}}
			tab.Prompts = []model.MCPPrompt{{Name: "p"}}
		}
		goView := Row(H(w.RowH+8), PadX(w.Gutter), Gap(6), AlignItems(AlignCenter), BorderB(w.Border),
			Text(connState(tab), Small(), Muted()),
			Spacer(),
			a.viewChip("Auto-reconnect", tab.Reconnect, 116, func() {}),
			If(len(tab.Tools) > 0, Button(fmt.Sprintf("Tools (%d)", len(tab.Tools)), MinW(84), H(w.CompactH), OnClick(func() {}), Tooltip("Pick a tool: the composer gets a call with its arguments filled in"))),
			If(len(tab.Resources) > 0, Button(fmt.Sprintf("Resources (%d)", len(tab.Resources)), MinW(104), H(w.CompactH), OnClick(func() {}), Tooltip("Pick a resource: the composer gets a read of it"))),
			If(len(tab.Prompts) > 0, Button(fmt.Sprintf("Prompts (%d)", len(tab.Prompts)), MinW(94), H(w.CompactH), OnClick(func() {}), Tooltip("Pick a prompt: the composer gets a request for it"))),
			Button("Ping", MinW(56), H(w.CompactH), OnClick(func() {}), Tooltip("Send a ping frame")),
			Button("Clear", MinW(56), H(w.CompactH), OnClick(func() {}), Tooltip("Empty the message log")),
		)
		tplView := a.messagesToolbar(tab)
		Layout(a.ctx, goView, 700, 36)
		Layout(a.ctx, tplView, 700, 36)
		if got, want := Dump(tplView), Dump(goView); got != want {
			t.Fatalf("mcp=%v\ntemplate:\n%s\nGo:\n%s", mcp, got, want)
		}
	}
}
