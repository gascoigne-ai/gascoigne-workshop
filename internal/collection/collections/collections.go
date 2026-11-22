// Package collections holds the rules for managing whole collections, each a
// folder on disk: what a name may be, what the delete question says, which
// collection opens when the open one is deleted, and the Set of open
// collections (order, folder, active environment, which collection a request
// belongs to). It has no I/O; the store port does the work.
package collections

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// MaxNameLen is the longest collection name, in characters. The name becomes
// a folder name, which file systems limit.
const MaxNameLen = 80

// CleanName trims a name typed by the user and rejects one that cannot name
// a collection.
func CleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return "", errors.New("the name is empty")
	case utf8.RuneCountInString(name) > MaxNameLen:
		return "", errors.New("the name is too long")
	}
	return name, nil
}

// DeletePrompt is the title and the question of the confirmation dialog
// that comes before a collection is deleted.
func DeletePrompt(name string) (title, body string) {
	return "Delete " + name + "?",
		"Its requests, folders and environments are removed from the disk. This cannot be undone."
}

// ClosePrompt is the title and the question of the confirmation dialog
// that comes before an edited collection without a folder is closed.
func ClosePrompt(name string) (title, body string) {
	return "Close " + name + "?",
		"It is not saved in a folder, so its changes are lost. Save it first to keep them."
}

// NextAfterDelete is the collection to open once deleted is gone: the one
// after it in dirs, else the one before it, else "" when none is left. dirs
// is the order the store listed them in, deleted included or not.
func NextAfterDelete(dirs []string, deleted string) string {
	var rest []string
	at := -1
	for _, dir := range dirs {
		if dir == deleted {
			at = len(rest)
			continue
		}
		rest = append(rest, dir)
	}
	switch {
	case len(rest) == 0:
		return ""
	case at < 0 || at >= len(rest):
		return rest[len(rest)-1]
	}
	return rest[at]
}
