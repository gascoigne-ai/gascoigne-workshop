package userdir

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Path is the file or folder name inside the program's directory in the
// user config directory, or "" when the platform has none (what is kept
// there then stays in memory). An empty name is the directory itself.
func Path(name string) string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, Name, name)
}

// WriteFileAtomic replaces path with data so a crash cannot leave half a
// file behind: it writes a temp file in the same directory, then renames it
// over path. The temp file is removed when any step fails. The helper lives
// in this model package, not in an adapter, because the layer rules let
// every domain import a model but keep adapters apart; it uses only the
// standard library. The directory of path must exist. Five adapters had
// copied it. Rejected (the user's choice): an adapter package shared across
// domains, which needs a layer exception, and a copy per domain.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file for %s: %w", path, err)
	}
	_, writeErr := temp.Write(data)
	closeErr := temp.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		os.Remove(temp.Name())
		return fmt.Errorf("write temp file for %s: %w", path, err)
	}
	if err := os.Chmod(temp.Name(), perm); err != nil {
		os.Remove(temp.Name())
		return fmt.Errorf("set mode of temp file for %s: %w", path, err)
	}
	if err := os.Rename(temp.Name(), path); err != nil {
		os.Remove(temp.Name())
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}
