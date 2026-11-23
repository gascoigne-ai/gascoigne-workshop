package app

import (
	"errors"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"regexp"
	"strconv"
	"strings"
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

func TestExplorerCallsTheOpenToolWithTheFormsArguments(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	tab.Req.Protocol = "mcp"
	announce(a, tab)
	session := &fakeSession{result: `{"content":[{"type":"text","text":"found"}]}`}
	openSession(tab, session)
	a.exploreOpen(tab, 0)
	tab.form.Inputs["q"].Text = "go"
	a.exploreCall(tab)
	until(t, a, "the answer", func() bool { return !tab.explorer.IsCalling() })
	if len(session.asked) != 1 || session.asked[0] != `call search {"q":"go","limit":5}` {
		t.Fatalf("asked %q", session.asked)
	}
	if tab.explorer.IsFailed() || !strings.Contains(tab.explorer.Result, "found") || len(tab.resultView.Lines) == 0 {
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
		a := testApp(t)
		tab := a.cur()
		tab.Req.Protocol = "mcp"
		announce(a, tab)
		session := &fakeSession{result: `{}`}
		openSession(tab, session)
		a.exploreChoose(tab, test.kind)
		a.exploreOpen(tab, test.index)
		a.exploreCall(tab)
		until(t, a, "the answer", func() bool { return !tab.explorer.IsCalling() })
		if len(session.asked) != 1 || session.asked[0] != test.want {
			t.Errorf("kind %v asked %q, want %q", test.kind, session.asked, test.want)
		}
	}
}

func TestExplorerShowsTheServersErrorAndAToolThatReportsOne(t *testing.T) {
	tests := []struct {
		name    string
		session *fakeSession
		want    string
	}{
		{"server error", &fakeSession{err: errors.New("no such tool")}, "no such tool"},
		{"isError result", &fakeSession{result: `{"isError":true,"content":[]}`}, "isError"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := testApp(t)
			tab := a.cur()
			tab.Req.Protocol = "mcp"
			announce(a, tab)
			openSession(tab, test.session)
			a.exploreOpen(tab, 1)
			a.exploreCall(tab)
			until(t, a, "the answer", func() bool { return !tab.explorer.IsCalling() })
			if !tab.explorer.IsFailed() || !strings.Contains(tab.explorer.Result, test.want) {
				t.Fatalf("state %+v", tab.explorer)
			}
		})
	}
}

func TestExplorerRefusesACallWithAnInvalidValue(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	tab.Req.Protocol = "mcp"
	announce(a, tab)
	session := &fakeSession{result: `{}`}
	openSession(tab, session)
	a.exploreOpen(tab, 0)
	tab.form.Inputs["limit"].Text = "lots"
	a.exploreCall(tab)
	if len(session.asked) != 0 || tab.explorer.IsCalling() || tab.form.Problem == "" {
		t.Fatalf("asked %q state %+v problem %q", session.asked, tab.explorer, tab.form.Problem)
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
		a.exploreChoose(tab, kind)
		a.Step(core.Input{})
		a.exploreOpen(tab, 0)
		a.exploreCall(tab)
		until(t, a, "the answer", func() bool { return !tab.explorer.IsCalling() })
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

// The empty list's note sat flush against the edge of the list and the
// tab strip had no line under it, unlike every other tab strip.
func TestExplorerListInsetsItsNoteAndRulesTheTabStrip(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	tab.Req.Protocol = "mcp"
	list := a.explorerList(tab)
	Layout(a.ctx, list, explorerListW, 300)
	dump := Dump(list)
	if !strings.Contains(dump, "borderB=border") {
		t.Errorf("the tab strip has no rule under it:\n%s", dump)
	}
	match := regexp.MustCompile(`text="Nothing announced".*rect=(\d+),`).FindStringSubmatch(dump)
	if match == nil {
		t.Fatalf("no note in:\n%s", dump)
	}
	if x, _ := strconv.Atoi(match[1]); x < core.Gutter {
		t.Errorf("the note starts at x=%d, want at least the gutter %d", x, core.Gutter)
	}
}
