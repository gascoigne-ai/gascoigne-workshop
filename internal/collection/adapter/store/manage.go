package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
)

// siblings is the lower-case names taken in parent, but not skip's.
func siblings(parent, skip string) (map[string]bool, error) {
	taken := map[string]bool{}
	entries, err := os.ReadDir(parent)
	if errors.Is(err, fs.ErrNotExist) {
		return taken, nil
	}
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.Name() != skip {
			taken[strings.ToLower(e.Name())] = true
		}
	}
	return taken, nil
}

// isCollection tells a folder that holds a collection from any other.
func isCollection(dir string) bool { return fileExists(filepath.Join(dir, folderFile)) }

// Create makes an empty collection called name in a new folder below
// parent, named after the name and made unique, and returns the folder.
func (files Files) Create(parent, name string) (string, error) {
	taken, err := siblings(parent, "")
	if err != nil {
		return "", err
	}
	dir := filepath.Join(parent, unique(slug(name), taken))
	if err := files.Save(dir, request.Collection{Folder: request.Folder{Name: name}}); err != nil {
		return "", err
	}
	return dir, nil
}

// Rename gives the collection in dir a new name and moves it to a folder
// named after it, which is made unique among its neighbors; the folder it
// ends up in is returned. A folder that is not a collection is not touched.
func (Files) Rename(dir, name string) (string, error) {
	if !isCollection(dir) {
		return "", fmt.Errorf("%s holds no collection", dir)
	}
	meta, err := readJSON(filepath.Join(dir, folderFile), folderReaders)
	if err != nil {
		return "", err
	}
	meta.Version, meta.Name = folderVersion, name
	if err := writeJSON(filepath.Join(dir, folderFile), meta, false); err != nil {
		return "", err
	}
	parent := filepath.Dir(dir)
	taken, err := siblings(parent, filepath.Base(dir))
	if err != nil {
		return "", err
	}
	want := unique(slug(name), taken)
	if strings.EqualFold(want, filepath.Base(dir)) {
		return dir, nil
	}
	moved := filepath.Join(parent, want)
	if err := os.Rename(dir, moved); err != nil {
		return "", err
	}
	return moved, nil
}

// Delete removes the collection in dir: the files this package writes, as
// Save removes them, then the folder itself when nothing else is in it.
// Files of the user's own stay, and so does the folder with them. A folder
// that is not a collection is refused.
func (Files) Delete(dir string) error {
	if !isCollection(dir) {
		return fmt.Errorf("%s holds no collection", dir)
	}
	if err := prune(dir, map[string]bool{}, reqSuffix); err != nil {
		return err
	}
	ed := filepath.Join(dir, envDir)
	if err := prune(ed, map[string]bool{}, envSuffix); err != nil {
		return err
	}
	ignore := filepath.Join(dir, ".gitignore")
	if old, err := os.ReadFile(ignore); err == nil && string(old) == secretsFile+"\n" {
		if err := os.Remove(ignore); err != nil {
			return err // ours alone: a .gitignore with other lines is the user's
		}
	}
	for _, p := range []string{ed, filepath.Join(dir, folderFile), filepath.Join(dir, secretsFile)} {
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) && !isNotEmpty(err) {
			return err
		}
	}
	if err := os.Remove(dir); err != nil && !isNotEmpty(err) {
		return err
	}
	return nil
}

// isNotEmpty is the error of removing a directory that still holds files.
func isNotEmpty(err error) bool {
	return errors.Is(err, fs.ErrExist) || strings.Contains(err.Error(), "not empty")
}

// List is the folders below parent that hold a collection, by name.
func (Files) List(parent string) ([]string, error) {
	entries, err := os.ReadDir(parent)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, e := range entries {
		if dir := filepath.Join(parent, e.Name()); e.IsDir() && isCollection(dir) {
			dirs = append(dirs, dir)
		}
	}
	sort.Strings(dirs)
	return dirs, nil
}
