// Package statefile stores what the app remembers between runs (window
// geometry, open tabs, pane sizes) as a small JSON file next to the
// settings. Settings are preferences users edit; state is bookkeeping.
// A missing or broken file is never fatal: the app starts fresh.
package statefile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/state"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/fileversion"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/userdir"
)

// stateFile is state.json: the state behind its version.
type stateFile struct {
	Version int `json:"version"`
	state.State
}

// readers read state.json by version; versions 0 and 1 share a layout.
var readers = []func([]byte) (stateFile, error){
	fileversion.Plain[stateFile],
	fileversion.Plain[stateFile],
}

// current is the version Save writes.
var current = len(readers) - 1

// Load reads path. A missing file or an empty path yields the empty
// state; a broken file yields the empty state and an error to report.
func (Files) Load(path string) (state.State, error) {
	if path == "" {
		return state.State{}, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return state.State{}, nil
	}
	if err != nil {
		return state.State{}, fmt.Errorf("read state: %w", err)
	}
	saved, _, err := fileversion.Read(data, readers)
	if err != nil {
		return state.State{}, fmt.Errorf("parse state %s: %w", filepath.Base(path), err)
	}
	return saved.State, nil
}

// Save writes s to path atomically. An empty path keeps it in memory.
func (Files) Save(path string, saved state.State) error {
	if path == "" {
		return nil
	}
	data, err := json.MarshalIndent(stateFile{Version: current, State: saved}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	if err := userdir.WriteFileAtomic(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	return nil
}

// DefaultPath is where state lives for this user, or "" when there is
// no config directory.
func DefaultPath() string {
	return userdir.Path("state.json")
}

// Files is the state store of the file system.
type Files struct{}

var _ workspace.StateStore = Files{}
