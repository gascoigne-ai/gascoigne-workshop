package settings

import "errors"

// SettingsStore keeps the settings file.
type SettingsStore interface {
	// Load reads the settings at path. A missing file yields the defaults;
	// a file that does not parse yields the defaults and an error.
	Load(path string) (Settings, error)
	// Save writes what differs from the defaults to path, atomically.
	Save(path string, current Settings) error
	// ReadText reads the file as it is.
	ReadText(path string) ([]byte, error)
	// Stat stamps the file as it is now; a missing file is the zero stamp.
	Stat(path string) FileStamp
	// WriteText replaces the file with text exactly as given, atomically.
	WriteText(path string, text []byte) error
}

// errNewer is every write of a file a newer build wrote.
var errNewer = errors.New("settings.json was written by a newer version of the workshop; it is not changed this session")

// ReadOnly is store that reads and refuses every write, so a session that
// found the file written by a newer build never overwrites it. It is a rule
// about the file, not the window, which is why it lives with the store.
func ReadOnly(store SettingsStore) SettingsStore { return readOnly{store} }

// readOnly reads through the wrapped store and writes nothing.
type readOnly struct{ SettingsStore }

func (readOnly) Save(string, Settings) error    { return errNewer }
func (readOnly) WriteText(string, []byte) error { return errNewer }
