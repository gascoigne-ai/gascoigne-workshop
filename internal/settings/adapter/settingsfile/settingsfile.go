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
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/userdir"
)

// Load reads path. A missing file is not an error and yields the
// defaults; fields missing from the file keep their defaults. Values
// outside their range are clamped and reported as a *settings.ClampedError
// with the usable settings.
func (Files) Load(path string) (settings.Settings, error) {
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
	loaded, notes := settings.Clamp(loaded)
	if len(notes) > 0 {
		return loaded, &settings.ClampedError{Notes: notes}
	}
	return loaded, nil
}

// Save writes current to path, creating its directory. Only what differs
// from the defaults is written. The file is replaced atomically so a crash
// cannot leave half a file behind.
func (files Files) Save(path string, current settings.Settings) error {
	data, err := settings.Encode(current)
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	return files.WriteText(path, data)
}

// WriteText replaces the settings file with data exactly as given, creating
// its directory. The file is replaced atomically. An existing file that
// does not parse is first copied to path+".bak", so replacing it never
// destroys what the user wrote.
func (Files) WriteText(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create settings dir: %w", err)
	}
	if err := backUpBroken(path); err != nil {
		return err
	}
	if err := userdir.WriteFileAtomic(path, data, 0o644); err != nil {
		return fmt.Errorf("write settings: %w", err)
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
	return userdir.Path(name)
}

// Files is the settings store of the file system.
type Files struct{}

// ReadText implements port.SettingsStore.
func (Files) ReadText(path string) ([]byte, error) { return os.ReadFile(path) }

// Stat implements port.SettingsStore.
func (Files) Stat(path string) settings.FileStamp {
	info, err := os.Stat(path)
	if err != nil {
		return settings.FileStamp{}
	}
	return settings.FileStamp{Modified: info.ModTime(), Size: info.Size(), Exists: true}
}

var _ port.SettingsStore = Files{}

func backUpBroken(path string) error {
	old, err := os.ReadFile(path)
	if err != nil {
		return nil // nothing there to lose
	}
	if _, err := settings.Decode(old); err == nil {
		return nil
	}
	if err := os.WriteFile(path+".bak", old, 0o644); err != nil {
		return fmt.Errorf("back up broken settings: %w", err)
	}
	return nil
}
