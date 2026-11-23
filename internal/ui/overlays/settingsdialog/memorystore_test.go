package settingsdialog

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
)

// memoryFile is one file of the memoryStore: its text and the write that made it.
type memoryFile struct {
	text    []byte
	version int
}

// memoryStore is the settings files in memory: a fake of
// settings.SettingsStore that behaves as the file adapter does, so the dialog
// tests need no disk. Every write, the dialog's or a test's, gets a new stamp,
// as a file's modification time changes.
type memoryStore struct {
	files   map[string]memoryFile
	version int
}

func newMemoryStore() *memoryStore { return &memoryStore{files: map[string]memoryFile{}} }

// text is the file's text, empty when there is no file.
func (store *memoryStore) text(path string) string { return string(store.files[path].text) }

// has tells whether the file exists.
func (store *memoryStore) has(path string) bool {
	_, ok := store.files[path]
	return ok
}

// write puts text in the file as another program would.
func (store *memoryStore) write(path, text string) {
	store.version++
	store.files[path] = memoryFile{text: []byte(text), version: store.version}
}

func (store *memoryStore) Load(path string) (settings.Settings, error) {
	file, ok := store.files[path]
	if !ok {
		return settings.Default(), nil
	}
	loaded, err := settings.Decode(file.text)
	if err != nil {
		return loaded, fmt.Errorf("parse settings %s: %w", filepath.Base(path), err)
	}
	loaded, notes := settings.Clamp(loaded)
	if len(notes) > 0 {
		return loaded, &settings.ClampedError{Notes: notes}
	}
	return loaded, nil
}

func (store *memoryStore) Save(path string, current settings.Settings) error {
	data, err := settings.Encode(current)
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	return store.WriteText(path, data)
}

func (store *memoryStore) ReadText(path string) ([]byte, error) {
	file, ok := store.files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return file.text, nil
}

func (store *memoryStore) Stat(path string) settings.FileStamp {
	file, ok := store.files[path]
	if !ok {
		return settings.FileStamp{}
	}
	return settings.FileStamp{Modified: time.Unix(int64(file.version), 0), Size: int64(len(file.text)), Exists: true}
}

func (store *memoryStore) WriteText(path string, text []byte) error {
	store.write(path, string(text))
	return nil
}

var _ settings.SettingsStore = (*memoryStore)(nil)
