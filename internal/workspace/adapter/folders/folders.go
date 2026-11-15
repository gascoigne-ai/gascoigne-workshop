// Package folders reads folders from the disk for the in-app file browser
// (port.Folders).
package folders

import (
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

// Start implements port.Folders.
func (Disk) Start() (string, error) { return os.Getwd() }

// Home implements port.Folders.
func (Disk) Home() (string, error) { return os.UserHomeDir() }

var _ port.Folders = Disk{}
