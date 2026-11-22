// Package mcpexplorer is the UI-free state of the MCP explorer: which list
// is shown, which entry is open and what its last call answered.
package mcpexplorer

import (
	"encoding/json"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
)

// Kind is the list the explorer shows.
type Kind int

// The lists a server can announce.
const (
	Tools Kind = iota
	Resources
	Prompts
)

// None is State.Selected when no entry is open.
const None = -1

// Entry is one line of a list: a title and a muted detail under it.
type Entry struct{ Title, Detail string }

// CallState is where the open entry's call is.
type CallState int

// The states of a call.
const (
	Idle     CallState = iota // nothing sent, or the answer was dropped by opening another entry
	Calling                   // sent, waiting for the answer
	Answered                  // the server answered and the answer is not an error
	Failed                    // the server or the tool answered with an error
)

// State is what the explorer remembers between frames.
type State struct {
	Kind     Kind
	Selected int // index into the list, None for no entry
	Call     CallState
	Result   string // the last answer: its JSON, or the server's error message
}

// IsCalling reports whether a call is waiting for its answer.
func (state State) IsCalling() bool { return state.Call == Calling }

// IsFailed reports whether Result is an error.
func (state State) IsFailed() bool { return state.Call == Failed }

// New is a state showing the tools with nothing open.
func New() State { return State{Selected: None} }

// Choose switches to another list.
func (state *State) Choose(kind Kind) {
	*state = State{Kind: kind, Selected: None}
}

// Select opens an entry of the current list and drops the last answer. A
// call in flight stays one: its answer still lands, on the entry then open.
func (state *State) Select(index int) {
	state.Selected, state.Result = index, ""
	if state.Call != Calling {
		state.Call = Idle
	}
}

// Begin marks a call as sent.
func (state *State) Begin() { state.Call, state.Result = Calling, "" }

// Finish records the answer of the call. A tool that says isError in its
// result failed as well, though the server answered.
func (state *State) Finish(result string, err error) {
	if err != nil {
		state.Call, state.Result = Failed, err.Error()
		return
	}
	state.Call, state.Result = Answered, result
	if reportsError(result) {
		state.Call = Failed
	}
}

func reportsError(result string) bool {
	var answer struct {
		IsError bool `json:"isError"`
	}
	return json.Unmarshal([]byte(result), &answer) == nil && answer.IsError
}

// Entries are the lines of the list of kind.
func Entries(kind Kind, tools []request.MCPTool, resources []request.MCPResource, prompts []request.MCPPrompt) []Entry {
	var out []Entry
	switch kind {
	case Tools:
		for _, tool := range tools {
			out = append(out, Entry{Title: tool.Name, Detail: tool.Description})
		}
	case Resources:
		for _, resource := range resources {
			if resource.Name == "" {
				out = append(out, Entry{Title: resource.URI})
				continue
			}
			out = append(out, Entry{Title: resource.Name, Detail: resource.URI})
		}
	case Prompts:
		for _, prompt := range prompts {
			out = append(out, Entry{Title: prompt.Name})
		}
	}
	return out
}
