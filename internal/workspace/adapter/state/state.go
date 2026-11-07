// Package state stores what the app remembers between runs (window
// geometry, open tabs, pane sizes) as a small JSON file next to the
// settings. Settings are preferences users edit; state is bookkeeping.
// A missing or broken file is never fatal: the app starts fresh.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Smallest window the app restores.
const (
	MinWindowW = 640
	MinWindowH = 400
)

// Window is the window's size and position in device-independent pixels.
// A zero size means nothing was remembered.
type Window struct {
	W int `json:"w"`
	H int `json:"h"`
	X int `json:"x"`
	Y int `json:"y"`
}

// Clamp keeps the window inside a monitor of mw by mh: not larger, not
// smaller than the minimum, and not positioned off screen. An unset
// window stays unset.
func (w Window) Clamp(mw, mh int) Window {
	if w.W == 0 || w.H == 0 {
		return Window{}
	}
	w.W = max(MinWindowW, min(w.W, mw))
	w.H = max(MinWindowH, min(w.H, mh))
	w.X = max(0, min(w.X, mw-w.W))
	w.Y = max(0, min(w.Y, mh-w.H))
	return w
}

// Tab is one open request, found again by its folder path.
type Tab struct {
	Path string `json:"path"`           // folder names and request name joined by "/"
	Pane int    `json:"pane,omitempty"` // selected request pane tab
	Resp int    `json:"resp,omitempty"` // selected response pane tab
}

// State is everything remembered.
type State struct {
	Window    Window               `json:"window"`
	SidebarW  float64              `json:"sidebarW,omitempty"`
	PaneSplit float64              `json:"paneSplit,omitempty"`
	Expanded  []string             `json:"expanded,omitempty"` // folder paths
	Tabs      []Tab                `json:"tabs,omitempty"`
	Active    int                  `json:"active,omitempty"`
	Columns   map[string][]float64 `json:"columns,omitempty"`   // table column weights by table name
	SettingsW int                  `json:"settingsW,omitempty"` // the settings dialog size the user chose; 0 is the default
	SettingsH int                  `json:"settingsH,omitempty"`
}

// Load reads path. A missing file or an empty path yields the empty
// state; a broken file yields the empty state and an error to report.
func Load(path string) (State, error) {
	if path == "" {
		return State{}, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return State{}, nil
	}
	if err != nil {
		return State{}, fmt.Errorf("read state: %w", err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return State{}, fmt.Errorf("parse state %s: %w", filepath.Base(path), err)
	}
	return s, nil
}

// Save writes s to path atomically. An empty path keeps it in memory.
func (s State) Save(path string) error {
	if path == "" {
		return nil
	}
	data, err := json.MarshalIndent(s, "", "  ")
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
	return filepath.Join(dir, "gascoigne-workshop", "state.json")
}
