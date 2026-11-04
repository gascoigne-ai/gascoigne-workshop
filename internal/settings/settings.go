// Package settings stores the user's preferences as a small JSON file in
// the user config directory. Unknown values are kept as written; the UI
// decides how to fall back.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Settings are the user preferences.
type Settings struct {
	Theme    string `json:"theme"`    // "dark" or "light"
	TextSize string `json:"textSize"` // "small", "medium" or "large"

	// Layout places the request and response panes: "side" by side or "stacked".
	Layout string `json:"layout"`

	// Contrast is "normal" or "high"; high strengthens the selection,
	// find and accent colors.
	Contrast string `json:"contrast"`

	// Colors are the user's color overrides: theme id, then token name,
	// then "#rrggbb".
	Colors map[string]map[string]string `json:"colors,omitempty"`

	// Shortcuts are the user's key bindings: action id to a key string such
	// as "Ctrl+Shift+Tab"; "" means unbound. Missing actions keep their default.
	Shortcuts map[string]string `json:"shortcuts,omitempty"`

	// Corners is how round controls are: "square", "soft" (the theme's own
	// radii) or "round".
	Corners string `json:"corners"`

	// Flat turns every gradient into a flat fill.
	Flat bool `json:"flat"`

	// UIScalePct is the interface scale in percent, on top of the
	// display's own scale factor; 0 means the default.
	UIScalePct int `json:"uiScale"`

	// TimeoutSec is the default request timeout in seconds.
	TimeoutSec int `json:"timeoutSec"`
}

// DefaultUIScalePct is the display's own scale; MinUIScalePct and
// MaxUIScalePct bound what is accepted.
const (
	DefaultUIScalePct = 100
	MinUIScalePct     = 50
	MaxUIScalePct     = 300
)

// UIScale is the interface scale as a factor, always within bounds.
func (s Settings) UIScale() float64 {
	pct := s.UIScalePct
	if pct == 0 {
		pct = DefaultUIScalePct
	}
	return float64(max(MinUIScalePct, min(pct, MaxUIScalePct))) / 100
}

// Default is what a first run uses.
func Default() Settings {
	return Settings{Theme: "dark", TextSize: "medium", Layout: "side", Contrast: "normal", TimeoutSec: 60, UIScalePct: DefaultUIScalePct, Corners: "soft"}
}

// Load reads path. A missing file is not an error and yields Default;
// fields missing from the file keep their defaults.
func Load(path string) (Settings, error) {
	s := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, fmt.Errorf("read settings: %w", err)
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return Default(), fmt.Errorf("parse settings %s: %w", filepath.Base(path), err)
	}
	return s, nil
}

// Save writes s to path, creating its directory. The file is replaced
// atomically so a crash cannot leave half a file behind.
func (s Settings) Save(path string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create settings dir: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace settings: %w", err)
	}
	return nil
}

// ThemesDir is where the user's theme files live, or "" when there is no
// config directory.
func ThemesDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "gascoigne-workshop", "themes")
}

// DefaultPath is where settings live for this user, or "" when the
// platform has no config directory (settings then stay in memory).
func DefaultPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "gascoigne-workshop", "settings.json")
}
