package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/controller/mcpexplorer"
	reqport "github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// requesterSession answers every command with a fixed result and records
// what it was asked.
type requesterSession struct {
	asked  []string
	result string
	err    error
}

func (session *requesterSession) Events() <-chan reqport.Event { return nil }
func (session *requesterSession) Send(string) error            { return nil }
func (session *requesterSession) Ping() error                  { return nil }
func (session *requesterSession) Close() error                 { return nil }
func (session *requesterSession) Request(_ context.Context, text string) (string, error) {
	session.asked = append(session.asked, text)
	return session.result, session.err
}

// announce puts a server's lists into the tab, as its events would.
func announce(a *App, tab *reqTab) {
	for _, list := range []string{
		`{"result":{"tools":[{"name":"search","description":"Find","inputSchema":{"type":"object","properties":{"q":{"type":"string"},"limit":{"type":"integer","default":5}},"required":["q"]}},{"name":"ping"}]}}`,
		`{"result":{"resources":[{"uri":"file:///a.txt","name":"A"}]}}`,
		`{"result":{"prompts":[{"name":"review","arguments":[{"name":"code","required":true}]}]}}`,
	} {
		a.applyWS(wsEvent{tab: tab, hasEv: true, ev: reqport.Event{Dir: reqport.Received, Kind: "response", Text: list}})
	}
}

func TestExplorerCallsTheOpenToolWithTheFormsArguments(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.Req.Protocol = "mcp"
	announce(a, tab)
	session := &requesterSession{result: `{"content":[{"type":"text","text":"found"}]}`}
	tab.Session = session
	a.exploreOpen(tab, 0)
	tab.form.Inputs["q"].Text = "go"
	a.exploreCall(tab)
	until(t, a, "the answer", func() bool { return !tab.explorer.Calling })
	if len(session.asked) != 1 || session.asked[0] != `call search {"q":"go","limit":5}` {
		t.Fatalf("asked %q", session.asked)
	}
	if tab.explorer.Failed || !strings.Contains(tab.explorer.Result, "found") || len(tab.resultView.Lines) == 0 {
		t.Fatalf("state %+v", tab.explorer)
	}
}

func TestExplorerCommandsPerKind(t *testing.T) {
	tests := []struct {
		kind  mcpexplorer.Kind
		index int
		want  string
	}{
		{mcpexplorer.Tools, 1, `call ping {}`},
		{mcpexplorer.Resources, 0, `read file:///a.txt`},
		{mcpexplorer.Prompts, 0, `prompt review {"code":""}`},
	}
	for _, test := range tests {
		a, _ := newTestApp(t)
		tab := a.cur()
		tab.Req.Protocol = "mcp"
		announce(a, tab)
		session := &requesterSession{result: `{}`}
		tab.Session = session
		a.exploreChoose(tab, test.kind)
		a.exploreOpen(tab, test.index)
		a.exploreCall(tab)
		until(t, a, "the answer", func() bool { return !tab.explorer.Calling })
		if len(session.asked) != 1 || session.asked[0] != test.want {
			t.Errorf("kind %v asked %q, want %q", test.kind, session.asked, test.want)
		}
	}
}

func TestExplorerShowsTheServersErrorAndAToolThatReportsOne(t *testing.T) {
	tests := []struct {
		name    string
		session *requesterSession
		want    string
	}{
		{"server error", &requesterSession{err: errors.New("no such tool")}, "no such tool"},
		{"isError result", &requesterSession{result: `{"isError":true,"content":[]}`}, "isError"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a, _ := newTestApp(t)
			tab := a.cur()
			tab.Req.Protocol = "mcp"
			announce(a, tab)
			tab.Session = test.session
			a.exploreOpen(tab, 1)
			a.exploreCall(tab)
			until(t, a, "the answer", func() bool { return !tab.explorer.Calling })
			if !tab.explorer.Failed || !strings.Contains(tab.explorer.Result, test.want) {
				t.Fatalf("state %+v", tab.explorer)
			}
		})
	}
}

func TestExplorerRefusesACallWithAnInvalidValue(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.Req.Protocol = "mcp"
	announce(a, tab)
	session := &requesterSession{result: `{}`}
	tab.Session = session
	a.exploreOpen(tab, 0)
	tab.form.Inputs["limit"].Text = "lots"
	a.exploreCall(tab)
	if len(session.asked) != 0 || tab.explorer.Calling || tab.form.Problem == "" {
		t.Fatalf("asked %q state %+v problem %q", session.asked, tab.explorer, tab.form.Problem)
	}
}

func TestExplorerAndTrafficPanesDrawWithoutPanicking(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.Req.Protocol = "mcp"
	a.step(core.Input{}) // no server yet
	a.applyWS(wsEvent{tab: tab, hasEv: true, ev: reqport.Event{Dir: reqport.Info, Kind: "initialize", Text: `{"protocolVersion":"2025-06-18","capabilities":{"tools":{}},"serverInfo":{"name":"mock","version":"1"},"instructions":"Use echo."}`}})
	announce(a, tab)
	tab.Session = &requesterSession{result: `{"content":[]}`}
	for _, kind := range []mcpexplorer.Kind{mcpexplorer.Tools, mcpexplorer.Resources, mcpexplorer.Prompts} {
		a.exploreChoose(tab, kind)
		a.step(core.Input{})
		a.exploreOpen(tab, 0)
		a.exploreCall(tab)
		until(t, a, "the answer", func() bool { return !tab.explorer.Calling })
	}
	tab.respTab = 1 // the JSON-RPC traffic
	a.step(core.Input{})
	if tab.Server == nil || tab.Server.Name != "mock" {
		t.Fatalf("server %+v", tab.Server)
	}
}

// Choosing another protocol while the second response pane of MCP is open
// must not leave the tab past the end of the shorter list.
func TestLeavingMCPShowsTheFirstResponsePane(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.Req.Protocol = "mcp"
	tab.respTab = 1
	a.cycleProtocol(tab) // mcp -> sse
	if tab.respTab != 0 {
		t.Fatalf("respTab %d", tab.respTab)
	}
	a.step(core.Input{})
}
