package port

import "github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/model/state"

// StateStore keeps what the app remembers between runs.
type StateStore interface {
	// Load reads the state at path; a missing file or an empty path is the
	// empty state.
	Load(path string) (state.State, error)
	// Save writes the state to path; an empty path keeps it in memory.
	Save(path string, s state.State) error
}
