package mcpexplorer

import (
	"errors"
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

func TestEntriesListWhatTheServerAnnounced(t *testing.T) {
	tools := []model.MCPTool{{Name: "add", Description: "Add numbers"}, {Name: "echo"}}
	resources := []model.MCPResource{{URI: "file:///a", Name: "A"}, {URI: "file:///b"}}
	prompts := []model.MCPPrompt{{Name: "greet"}}
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

func TestChoosingAKindClearsTheSelectionAndTheResult(t *testing.T) {
	state := State{Kind: Tools, Selected: 2, Result: "x", Failed: true}
	state.Choose(Prompts)
	if state.Kind != Prompts || state.Selected != None || state.Result != "" || state.Failed {
		t.Fatalf("%+v", state)
	}
}

func TestSelectingAnEntryClearsTheLastResult(t *testing.T) {
	state := State{Selected: None, Result: "old"}
	state.Select(1)
	if state.Selected != 1 || state.Result != "" {
		t.Fatalf("%+v", state)
	}
}

func TestFinishKeepsTheResultAndFlagsFailures(t *testing.T) {
	tests := []struct {
		name   string
		result string
		err    error
		failed bool
	}{
		{"result", `{"content":[]}`, nil, false},
		{"tool reports isError", `{"isError":true,"content":[]}`, nil, true},
		{"server error", "", errors.New("no such tool"), true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := State{}
			state.Begin()
			if !state.Calling {
				t.Fatal("Begin does not mark the call")
			}
			state.Finish(test.result, test.err)
			want := test.result
			if test.err != nil {
				want = test.err.Error()
			}
			if state.Calling || state.Failed != test.failed || state.Result != want {
				t.Fatalf("%+v", state)
			}
		})
	}
}
