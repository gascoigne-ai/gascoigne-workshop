package settings

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
