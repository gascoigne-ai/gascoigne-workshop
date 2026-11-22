package mcpexplorer

import (
	"errors"
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
)

func TestEntriesListWhatTheServerAnnounced(t *testing.T) {
	tools := []request.MCPTool{{Name: "add", Description: "Add numbers"}, {Name: "echo"}}
	resources := []request.MCPResource{{URI: "file:///a", Name: "A"}, {URI: "file:///b"}}
	prompts := []request.MCPPrompt{{Name: "greet"}}
	tests := []struct {
		kind Kind
		want []Entry
	}{
		{Tools, []Entry{{Title: "add", Detail: "Add numbers"}, {Title: "echo"}}},
		{Resources, []Entry{{Title: "A", Detail: "file:///a"}, {Title: "file:///b"}}},
		{Prompts, []Entry{{Title: "greet"}}},
	}
	for _, test := range tests {
		if got := Entries(test.kind, tools, resources, prompts); !reflect.DeepEqual(got, test.want) {
			t.Errorf("%v: %+v, want %+v", test.kind, got, test.want)
		}
	}
}

// The explorer's call state moves as the window used it before it was a
// type: Select keeps a call in flight (its answer still lands), Choose drops
// everything.
func TestTheCallStateMoves(t *testing.T) {
	tests := []struct {
		name       string
		start      State
		act        func(*State)
		want       CallState
		wantResult string
	}{
		{"new is idle", New(), func(*State) {}, Idle, ""},
		{"begin is calling with no result", State{Call: Answered, Result: "old"}, func(state *State) { state.Begin() }, Calling, ""},
		{"finish with a result is answered", State{Call: Calling}, func(state *State) { state.Finish(`{"content":[]}`, nil) }, Answered, `{"content":[]}`},
		{"finish with an error is failed", State{Call: Calling}, func(state *State) { state.Finish("", errors.New("no such tool")) }, Failed, "no such tool"},
		{"finish with isError is failed", State{Call: Calling}, func(state *State) { state.Finish(`{"isError":true}`, nil) }, Failed, `{"isError":true}`},
		{"select while calling stays calling", State{Call: Calling, Selected: 0}, func(state *State) { state.Select(1) }, Calling, ""},
		{"select after an answer is idle", State{Call: Answered, Result: "x"}, func(state *State) { state.Select(1) }, Idle, ""},
		{"select after a failure is idle", State{Call: Failed, Result: "x"}, func(state *State) { state.Select(1) }, Idle, ""},
		{"choose is idle, even while calling", State{Call: Calling, Selected: 2, Result: "x"}, func(state *State) { state.Choose(Prompts) }, Idle, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := test.start
			test.act(&state)
			if state.Call != test.want || state.Result != test.wantResult {
				t.Fatalf("%+v, want call %v result %q", state, test.want, test.wantResult)
			}
			if state.IsCalling() != (test.want == Calling) || state.IsFailed() != (test.want == Failed) {
				t.Fatalf("IsCalling %v IsFailed %v for %v", state.IsCalling(), state.IsFailed(), test.want)
			}
		})
	}
}

func TestChoosingAKindClearsTheSelectionAndTheResult(t *testing.T) {
	state := State{Kind: Tools, Selected: 2, Result: "x", Call: Failed}
	state.Choose(Prompts)
	if state.Kind != Prompts || state.Selected != None || state.Result != "" || state.Call != Idle {
		t.Fatalf("%+v", state)
	}
}

func TestSelectingAnEntryOpensItAndClearsTheLastResult(t *testing.T) {
	state := State{Selected: None, Result: "old"}
	state.Select(1)
	if state.Selected != 1 || state.Result != "" {
		t.Fatalf("%+v", state)
	}
}
