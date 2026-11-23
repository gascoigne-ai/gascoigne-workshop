package mcp

import (
	"errors"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/mcpexplorer"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestExplorerCallsTheOpenToolWithTheFormsArguments(t *testing.T) {
	world := newWorld(t)
	world.announce()
	session := &fakeSession{result: `{"content":[{"type":"text","text":"found"}]}`}
	world.connect(session)
	world.pane.Open(world.tab, 0)
	made := world.state()
	made.form.Inputs["q"].Text = "go"
	world.pane.Call(world.tab)
	world.fake.until(t, "the answer", func() bool { return !world.pane.Calling(world.tab) })
	if got := session.questions(); len(got) != 1 || got[0] != `call search {"q":"go","limit":5}` {
		t.Fatalf("asked %q", got)
	}
	if made.explorer.IsFailed() || !strings.Contains(made.explorer.Result, "found") || len(made.result.Lines) == 0 {
		t.Fatalf("state %+v", made.explorer)
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
		world := newWorld(t)
		world.announce()
		session := &fakeSession{result: `{}`}
		world.connect(session)
		world.pane.Choose(world.tab, test.kind)
		world.pane.Open(world.tab, test.index)
		world.pane.Call(world.tab)
		world.fake.until(t, "the answer", func() bool { return !world.pane.Calling(world.tab) })
		if got := session.questions(); len(got) != 1 || got[0] != test.want {
			t.Errorf("kind %v asked %q, want %q", test.kind, got, test.want)
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
			world := newWorld(t)
			world.announce()
			world.connect(test.session)
			world.pane.Open(world.tab, 1)
			world.pane.Call(world.tab)
			world.fake.until(t, "the answer", func() bool { return !world.pane.Calling(world.tab) })
			made := world.state()
			if !made.explorer.IsFailed() || !strings.Contains(made.explorer.Result, test.want) {
				t.Fatalf("state %+v", made.explorer)
			}
		})
	}
}

func TestExplorerRefusesACallWithAnInvalidValue(t *testing.T) {
	world := newWorld(t)
	world.announce()
	session := &fakeSession{result: `{}`}
	world.connect(session)
	world.pane.Open(world.tab, 0)
	made := world.state()
	made.form.Inputs["limit"].Text = "lots"
	world.pane.Call(world.tab)
	if got := session.questions(); len(got) != 0 || made.explorer.IsCalling() || made.form.Problem == "" {
		t.Fatalf("asked %q state %+v problem %q", got, made.explorer, made.form.Problem)
	}
}

// Asking about a tab that never showed the pane makes no state for it.
func TestAskingAboutATabThatNeverShowedThePaneLeavesNoState(t *testing.T) {
	world := newWorld(t)
	if world.pane.Calling(world.tab) {
		t.Fatal("a tab with no state is not calling")
	}
	world.pane.Call(world.tab)
	world.pane.Tick()
	if len(world.pane.states) != 0 {
		t.Fatalf("states %v, want none", world.pane.states)
	}
}

func TestTickDropsTheStateOfAClosedTab(t *testing.T) {
	world := newWorld(t)
	world.announce()
	world.view()
	world.fake.set.Open(&request.Request{Name: "other", Protocol: request.ProtocolMCP})
	other := world.fake.set.Current()
	world.pane.View(world.ctx, other)
	if len(world.pane.states) != 2 {
		t.Fatalf("states %d, want 2", len(world.pane.states))
	}
	world.fake.set.Close(0)
	if world.pane.Tick() {
		t.Fatal("Tick changes nothing on screen")
	}
	if _, kept := world.pane.states[world.tab]; kept || len(world.pane.states) != 1 {
		t.Fatalf("states %v, want only the open tab's", world.pane.states)
	}
}

// The answer of a call lands in the state the call was made from; a tab
// closed meanwhile gets no new state for it.
func TestAnAnswerForAClosedTabMakesNoState(t *testing.T) {
	world := newWorld(t)
	world.announce()
	world.connect(&fakeSession{result: `{}`})
	world.pane.Open(world.tab, 1)
	made := world.state()
	world.pane.Call(world.tab)
	world.fake.set.Close(0)
	world.pane.Tick()
	world.fake.until(t, "the answer", func() bool { return !made.explorer.IsCalling() })
	if len(world.pane.states) != 0 {
		t.Fatalf("states %v, want none for the closed tab", world.pane.states)
	}
	if made.explorer.Result != `{}` {
		t.Fatalf("the answer went astray: %+v", made.explorer)
	}
}

// Enter in a field of the explorer's form sends the open entry.
func TestEnterInTheExplorerFormCalls(t *testing.T) {
	world := newWorld(t)
	world.announce()
	session := &fakeSession{result: `{}`}
	world.connect(session)
	world.pane.Open(world.tab, 0)
	world.clickOn(fieldAt(0))
	world.frame(core.Input{Enter: true}, nil)
	world.fake.until(t, "the answer", func() bool { return !world.pane.Calling(world.tab) })
	if got := session.questions(); len(got) != 1 || got[0] != `call search {"q":"","limit":5}` {
		t.Fatalf("asked %q", got)
	}
}

// Choosing a list or opening an entry clears the earlier answer.
func TestChoosingAListClearsTheFormAndTheAnswer(t *testing.T) {
	world := newWorld(t)
	world.announce()
	world.pane.Open(world.tab, 0)
	made := world.state()
	made.result.SetLines(plainLines([]byte("old")))
	world.pane.Choose(world.tab, mcpexplorer.Resources)
	if made.form != nil || len(made.result.Lines) != 0 || made.explorer.Selected != mcpexplorer.None {
		t.Fatalf("state %+v form %v lines %d", made.explorer, made.form, len(made.result.Lines))
	}
}
