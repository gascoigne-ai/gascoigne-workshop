// Package clientfile reads the .workshop client config from the user
// config directory (global) and from the working directory (local).
package clientfile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/clientconfig"
)

// Files reads client configs from two directories; an empty directory name
// is skipped.
type Files struct {
	GlobalDir string
	LocalDir  string
}

// DefaultGlobalDir is the settings directory in the user config directory,
// or "" when the system has none.
func DefaultGlobalDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "gascoigne-workshop")
}

// Load returns the global config merged with the local one, which wins for
// entries of the same name. A directory without a config file adds nothing.
func (files Files) Load() (clientconfig.Config, error) {
	global, err := loadDir(files.GlobalDir)
	if err != nil {
		return clientconfig.Config{}, err
	}
	local, err := loadDir(files.LocalDir)
	if err != nil {
		return clientconfig.Config{}, err
	}
	return clientconfig.Merge(global, local), nil
}

func loadDir(dir string) (clientconfig.Config, error) {
	if dir == "" {
		return clientconfig.Config{}, nil
	}
	path, err := findFile(dir)
	if err != nil || path == "" {
		return clientconfig.Config{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return clientconfig.Config{}, fmt.Errorf("read %s: %w", path, err)
	}
	config, err := clientconfig.Parse(path, data)
	if err != nil {
		return clientconfig.Config{}, err
	}
	return config, nil
}

// findFile returns the one config file in dir, or "" when there is none.
// Two files in different formats are ambiguous, so they are an error.
func findFile(dir string) (string, error) {
	var found []string
	for _, name := range clientconfig.FileNames() {
		path := filepath.Join(dir, name)
		_, err := os.Stat(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("look for client config: %w", err)
		}
		found = append(found, path)
	}
	if len(found) > 1 {
		return "", fmt.Errorf("%s holds more than one client config (%s); keep one", dir, strings.Join(baseNames(found), ", "))
	}
	if len(found) == 0 {
		return "", nil
	}
	return found[0], nil
}

func baseNames(paths []string) []string {
	names := make([]string, len(paths))
	for i, path := range paths {
		names[i] = filepath.Base(path)
	}
	return names
}
