// Package mcpexplorer is the UI-free state of the MCP explorer: which list
// is shown, which entry is open and what its last call answered.
package mcpexplorer

import (
	"encoding/json"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
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

// State is what the explorer remembers between frames.
type State struct {
	Kind     Kind
	Selected int    // index into the list, None for no entry
	Calling  bool   // a call is waiting for its answer
	Result   string // the last answer: its JSON, or the server's error message
	Failed   bool   // Result is an error
}

// New is a state showing the tools with nothing open.
func New() State { return State{Selected: None} }

// Choose switches to another list.
func (state *State) Choose(kind Kind) {
	*state = State{Kind: kind, Selected: None}
}

// Select opens an entry of the current list.
func (state *State) Select(index int) {
	state.Selected, state.Result, state.Failed = index, "", false
}

// Begin marks a call as sent.
func (state *State) Begin() { state.Calling, state.Result, state.Failed = true, "", false }

// Finish records the answer of the call. A tool that says isError in its
// result failed as well, though the server answered.
func (state *State) Finish(result string, err error) {
	state.Calling = false
	if err != nil {
		state.Result, state.Failed = err.Error(), true
		return
	}
	state.Result, state.Failed = result, reportsError(result)
}

func reportsError(result string) bool {
	var answer struct {
		IsError bool `json:"isError"`
	}
	return json.Unmarshal([]byte(result), &answer) == nil && answer.IsError
}

// Entries are the lines of the list of kind.
func Entries(kind Kind, tools []model.MCPTool, resources []model.MCPResource, prompts []model.MCPPrompt) []Entry {
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
