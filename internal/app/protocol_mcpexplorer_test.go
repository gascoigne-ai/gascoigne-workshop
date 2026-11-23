package app

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/mcpexplorer"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// announce puts a server's lists into the tab, as its events would.
func announce(app *App, tab *reqTab) {
	for _, list := range []string{
		`{"result":{"tools":[{"name":"search","description":"Find","inputSchema":{"type":"object","properties":{"q":{"type":"string"},"limit":{"type":"integer","default":5}},"required":["q"]}},{"name":"ping"}]}}`,
		`{"result":{"resources":[{"uri":"file:///a.txt","name":"A"}]}}`,
		`{"result":{"prompts":[{"name":"review","arguments":[{"name":"code","required":true}]}]}}`,
	} {
		app.applyWS(wsReceived{wsTarget{tab: tab}, listEvent(list)})
	}
}

func TestExplorerAndTrafficPanesDrawWithoutPanicking(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	tab.Req.Protocol = "mcp"
	a.Step(core.Input{}) // no server yet
	a.applyWS(wsReceived{wsTarget{tab: tab}, request.Event{Dir: request.Info, Kind: request.KindInitialize, Text: `{"protocolVersion":"2025-06-18","capabilities":{"tools":{}},"serverInfo":{"name":"mock","version":"1"},"instructions":"Use echo."}`}})
	announce(a, tab)
	openSession(tab, &fakeSession{result: `{"content":[]}`})
	for _, kind := range []mcpexplorer.Kind{mcpexplorer.Tools, mcpexplorer.Resources, mcpexplorer.Prompts} {
		a.mcp.Choose(tab.Tab, kind)
		a.Step(core.Input{})
		a.mcp.Open(tab.Tab, 0)
		a.mcp.Call(tab.Tab)
		until(t, a, "the answer", func() bool { return !a.mcp.Calling(tab.Tab) })
	}
	tab.respTab = 1 // the JSON-RPC traffic
	a.Step(core.Input{})
	if tab.Server == nil || tab.Server.Name != "mock" {
		t.Fatalf("server %+v", tab.Server)
	}
}

// Choosing another protocol while the second response pane of MCP is open
// must not leave the tab past the end of the shorter list.
func TestLeavingMCPShowsTheFirstResponsePane(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	tab.Req.Protocol = "mcp"
	tab.respTab = 1
	a.urlbar.CycleProtocol(tab.Tab) // mcp -> sse
	if tab.respTab != 0 {
		t.Fatalf("respTab %d", tab.respTab)
	}
	a.Step(core.Input{})
}
