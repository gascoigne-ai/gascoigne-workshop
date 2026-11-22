// Package history keeps the last responses of each request between runs,
// one small JSON file per request in a folder.
package history

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/userdir"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/fileversion"
)

// Limit is how many responses are kept per request.
const Limit = 10

// MaxBody is the largest body kept: a bigger response is shown once and
// not written to disk.
const MaxBody = 256 << 10

// file is what is written: the key is inside, so a file is only shown
// for the request it belongs to.
type file struct {
	Version int `json:"version"`
	Key     string
	Entries []entryFile
}

// entryFile is one saved response. Its keys are spelled out so a rename in
// response.Entry cannot change the file.
type entryFile struct {
	At         time.Time     `json:"At"`
	Status     int           `json:"Status"`
	StatusText string        `json:"StatusText"`
	Duration   time.Duration `json:"Duration"`
	Headers    []request.KV  `json:"Headers"`
	Body       []byte        `json:"Body"`
}

// readers read a history file by version; versions 0 and 1 share a layout.
var readers = []func([]byte) (file, error){fileversion.Plain[file], fileversion.Plain[file]}

// current is the version Save writes.
var current = len(readers) - 1

// path is the file of key; the name is a hash, since a request path has
// characters a file name cannot hold.
func path(dir, key string) string {
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(dir, hex.EncodeToString(sum[:8])+".json")
}

// Dir is the history kept in one folder.
type Dir string

// Load reads the entries saved for key (a request's path), newest first.
// Nothing saved is no entries and no error.
func (dir Dir) Load(key string) ([]response.Entry, error) {
	data, err := os.ReadFile(path(string(dir), key))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read history of %s: %w", key, err)
	}
	saved, _, err := fileversion.Read(data, readers)
	if err != nil {
		return nil, fmt.Errorf("parse history of %s: %w", key, err)
	}
	if saved.Key != key {
		return nil, nil
	}
	entries := make([]response.Entry, len(saved.Entries))
	for i, entry := range saved.Entries {
		entries[i] = response.Entry(entry)
	}
	return entries, nil
}

// Save replaces the entries of key with the first Limit of entries whose
// body is at most MaxBody; none removes the file. The file is written
// whole and renamed into place, so a crash leaves the old one.
func (dir Dir) Save(key string, entries []response.Entry) error {
	var keep []entryFile
	for _, entry := range entries {
		if len(entry.Body) <= MaxBody && len(keep) < Limit {
			keep = append(keep, entryFile(entry))
		}
	}
	target := path(string(dir), key)
	if len(keep) == 0 {
		if err := os.Remove(target); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("remove history of %s: %w", key, err)
		}
		return nil
	}
	data, err := json.Marshal(file{Version: current, Key: key, Entries: keep})
	if err != nil {
		return fmt.Errorf("encode history of %s: %w", key, err)
	}
	if err := os.MkdirAll(string(dir), 0o700); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	if err := userdir.WriteFileAtomic(target, data, 0o600); err != nil {
		return fmt.Errorf("write history of %s: %w", key, err)
	}
	return nil
}

// Sent implements response.History.
func (dir Dir) Sent() response.SentStore { return SentLog{Dir: string(dir)} }

var _ response.History = Dir("")
