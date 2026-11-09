package port

import "github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"

// SettingsStore keeps the settings file.
type SettingsStore interface {
	// Load reads the settings at path. A missing file yields the defaults;
	// a file that does not parse yields the defaults and an error.
	Load(path string) (settings.Settings, error)
	// Save writes what differs from the defaults to path, atomically.
	Save(path string, current settings.Settings) error
	// WriteText replaces the file with text exactly as given, atomically.
	WriteText(path string, text []byte) error
}
