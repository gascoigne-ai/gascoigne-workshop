// Package history keeps the last responses of each request between runs,
// one small JSON file per request in a folder.
package history

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

// Limit is how many responses are kept per request.
const Limit = 10

// MaxBody is the largest body kept: a bigger response is shown once and
// not written to disk.
const MaxBody = 256 << 10

// Entry is one remembered response.
type Entry struct {
	At         time.Time
	Status     int
	StatusText string
	Duration   time.Duration
	Headers    []model.KV
	Body       []byte
}

// file is what is written: the key is inside, so a file is only shown
// for the request it belongs to.
type file struct {
	Key     string
	Entries []Entry
}

// path is the file of key; the name is a hash, since a request path has
// characters a file name cannot hold.
func path(dir, key string) string {
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(dir, hex.EncodeToString(sum[:8])+".json")
}

// Load reads the entries saved for key (a request's path), newest first.
// Nothing saved is no entries and no error.
func Load(dir, key string) ([]Entry, error) {
	data, err := os.ReadFile(path(dir, key))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read history of %s: %w", key, err)
	}
	var saved file
	if err := json.Unmarshal(data, &saved); err != nil {
		return nil, fmt.Errorf("parse history of %s: %w", key, err)
	}
	if saved.Key != key {
		return nil, nil
	}
	return saved.Entries, nil
}

// Save replaces the entries of key with the first Limit of entries whose
// body is at most MaxBody; none removes the file. The file is written
// whole and renamed into place, so a crash leaves the old one.
func Save(dir, key string, entries []Entry) error {
	var keep []Entry
	for _, entry := range entries {
		if len(entry.Body) <= MaxBody && len(keep) < Limit {
			keep = append(keep, entry)
		}
	}
	target := path(dir, key)
	if len(keep) == 0 {
		if err := os.Remove(target); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("remove history of %s: %w", key, err)
		}
		return nil
	}
	data, err := json.Marshal(file{Key: key, Entries: keep})
	if err != nil {
		return fmt.Errorf("encode history of %s: %w", key, err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	temp := target + ".tmp"
	if err := os.WriteFile(temp, data, 0o600); err != nil {
		return fmt.Errorf("write history of %s: %w", key, err)
	}
	if err := os.Rename(temp, target); err != nil {
		return fmt.Errorf("replace history of %s: %w", key, err)
	}
	return nil
}
