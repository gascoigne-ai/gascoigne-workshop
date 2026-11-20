// Package fileversion reads the version of a saved JSON file and picks the
// reader of that version from a table. Every file the workshop writes
// carries "version" as its first key; a file without one is version 0, the
// layout written before versions existed (decision 22).
package fileversion

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// Key is the name of the version in every saved object.
const Key = "version"

// NewerError is a file written by a newer build: its layout is unknown to
// this one, so it is neither read nor overwritten.
type NewerError struct {
	Found  int // the version in the file
	Newest int // the newest version this build reads
}

func (err *NewerError) Error() string {
	return fmt.Sprintf("written by a newer version of the workshop (format %d, this build reads up to %d)", err.Found, err.Newest)
}

// Read decodes data with the reader of its version: readers[0] reads
// version 0 (no version key), readers[i] version i. It returns the value and
// the version found. A version above the last reader is a *NewerError; a
// version that is not a whole number from 0 up, or data that is not a JSON
// object, is an error too.
//
// The readers are a slice indexed by version because a version is a small
// dense integer read once per file and every version up to the newest has a
// reader: a map adds nothing and a switch per format is the sprawl the table
// replaces. Each reader decodes its whole version rather than chaining
// upgrades from version 0, since no layout has changed yet. Revisit at
// version 2: the old reader then decodes its own wire type and converts it,
// and a chain of upgrade steps pays off from three layouts on.
func Read[T any](data []byte, readers []func([]byte) (T, error)) (T, int, error) {
	var zero T
	version, err := Of(data)
	if err != nil {
		return zero, 0, err
	}
	if newest := len(readers) - 1; version > newest {
		return zero, version, &NewerError{Found: version, Newest: newest}
	}
	value, err := readers[version](data)
	if err != nil {
		return zero, version, err
	}
	return value, version, nil
}

// Of is the version of the JSON object in data: 0 when it has none.
func Of(data []byte) (int, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return 0, errors.New("not a JSON object")
	}
	var probe struct {
		Version json.RawMessage `json:"version"`
	}
	if err := json.Unmarshal(trimmed, &probe); err != nil {
		return 0, err
	}
	if probe.Version == nil {
		return 0, nil
	}
	return Parse(probe.Version)
}

// Parse reads raw, the JSON value of a version: a whole number from 0 up.
func Parse(raw json.RawMessage) (int, error) {
	version, err := strconv.Atoi(string(bytes.TrimSpace(raw)))
	if err != nil || version < 0 {
		return 0, fmt.Errorf("%s %s is not a whole number from 0 up", Key, raw)
	}
	return version, nil
}

// Plain decodes data into T as it is: the reader of a version whose layout
// is the wire type itself.
func Plain[T any](data []byte) (T, error) {
	var value T
	err := json.Unmarshal(data, &value)
	return value, err
}

// Access is whether a file the program loaded once may be written this
// session.
type Access int

const (
	Writable Access = iota // missing, read, or damaged and backed up
	ReadOnly               // written by a newer build: left as it is
)

// AccessAfter is the access to a file whose load ended with loadErr.
func AccessAfter(loadErr error) Access {
	var newer *NewerError
	if errors.As(loadErr, &newer) {
		return ReadOnly
	}
	return Writable
}
