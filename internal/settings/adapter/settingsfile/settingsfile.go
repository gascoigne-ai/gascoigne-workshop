// Package settingsfile reads and writes the settings file in the user
// config directory.
package settingsfile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/port"
)

// Load reads path. A missing file is not an error and yields the
// defaults; fields missing from the file keep their defaults.
func Load(path string) (settings.Settings, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return settings.Default(), nil
	}
	if err != nil {
		return settings.Default(), fmt.Errorf("read settings: %w", err)
	}
	loaded, err := settings.Decode(data)
	if err != nil {
		return loaded, fmt.Errorf("parse settings %s: %w", filepath.Base(path), err)
	}
	return loaded, nil
}

// Save writes current to path, creating its directory. Only what differs
// from the defaults is written. The file is replaced atomically so a crash
// cannot leave half a file behind.
func Save(path string, current settings.Settings) error {
	data, err := settings.Encode(current)
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	return WriteText(path, data)
}

// WriteText replaces the settings file with data exactly as given, creating
// its directory. The file is replaced atomically.
func WriteText(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create settings dir: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace settings: %w", err)
	}
	return nil
}

// ThemesDir is where the user's theme files live, or "" when there is no
// config directory.
func ThemesDir() string { return inConfigDir("themes") }

// DefaultWorkspace is where the collection is saved when the user saves
// without having chosen a folder, or "" when there is no config directory.
func DefaultWorkspace() string { return inConfigDir("workspace") }

// DefaultPath is where settings live for this user, or "" when the
// platform has no config directory (settings then stay in memory).
func DefaultPath() string { return inConfigDir("settings.json") }

func inConfigDir(name string) string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "gascoigne-workshop", name)
}

// Files is the settings store of the file system.
type Files struct{}

// Load implements port.SettingsStore.
func (Files) Load(path string) (settings.Settings, error) { return Load(path) }

// Save implements port.SettingsStore.
func (Files) Save(path string, current settings.Settings) error { return Save(path, current) }

// WriteText implements port.SettingsStore.
func (Files) WriteText(path string, text []byte) error { return WriteText(path, text) }

var _ port.SettingsStore = Files{}
