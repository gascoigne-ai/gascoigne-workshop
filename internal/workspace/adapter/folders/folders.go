// Package folders is the user's file system on the disk (port.Folders):
// folder listings for the in-app file browser and the files the user
// names.
package folders

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/port"
)

// Disk is the file system of this machine.
type Disk struct{}

// List implements port.Folders. A link counts as a folder when it points
// at one, so the browser opens it; a broken link is listed as a file.
func (Disk) List(dir string) ([]port.FolderEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", dir, err)
	}
	listed := make([]port.FolderEntry, 0, len(entries))
	for _, entry := range entries {
		isDir := entry.IsDir()
		if entry.Type()&fs.ModeSymlink != 0 {
			info, statErr := os.Stat(filepath.Join(dir, entry.Name()))
			isDir = statErr == nil && info.IsDir()
		}
		listed = append(listed, port.FolderEntry{Name: entry.Name(), Dir: isDir})
	}
	return listed, nil
}

// Home implements port.Folders.
func (Disk) Home() (string, error) { return os.UserHomeDir() }

// userFileMode is the mode of a file the user names: readable by others,
// as the files an editor saves.
const userFileMode = 0o644

// Write implements port.Folders.
func (Disk) Write(ctx context.Context, path string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return os.WriteFile(path, data, userFileMode)
}

// IsFile implements port.Folders. A path that cannot be looked at is
// answered as no file, which is what the file body shows for it.
func (Disk) IsFile(ctx context.Context, path string) bool {
	if ctx.Err() != nil || path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

var _ port.Folders = Disk{}
