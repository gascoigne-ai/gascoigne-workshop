// Package state stores what the app remembers between runs (window
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

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/fileversion"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/userdir"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/model/state"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/port"
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
func Load(path string) (state.State, error) {
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
func Save(path string, saved state.State) error {
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
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace state: %w", err)
	}
	return nil
}

// DefaultPath is where state lives for this user, or "" when there is
// no config directory.
func DefaultPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, userdir.Name, "state.json")
}

// Files is the state store of the file system.
type Files struct{}

// Load implements port.StateStore.
func (Files) Load(path string) (state.State, error) { return Load(path) }

// Save implements port.StateStore.
func (Files) Save(path string, saved state.State) error { return Save(path, saved) }

var _ port.StateStore = Files{}
