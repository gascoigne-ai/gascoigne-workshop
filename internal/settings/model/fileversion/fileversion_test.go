package fileversion

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// readers is a table of two versions whose readers say which one ran.
var readers = []func([]byte) (string, error){
	func([]byte) (string, error) { return "v0", nil },
	func([]byte) (string, error) { return "v1", nil },
}

func TestReadPicksReaderByVersion(t *testing.T) {
	tests := []struct {
		name        string
		data        string
		wantValue   string
		wantVersion int
		wantNewer   *NewerError // a newer file
		wantErr     bool        // any other error
	}{
		{name: "missing version is v0", data: `{"name":"x"}`, wantValue: "v0", wantVersion: 0},
		{name: "version 0", data: `{"version":0}`, wantValue: "v0", wantVersion: 0},
		{name: "version 1", data: `{"version":1,"name":"x"}`, wantValue: "v1", wantVersion: 1},
		{name: "version 1 after other keys", data: `{"name":"x","version":1}`, wantValue: "v1", wantVersion: 1},
		{name: "newer version", data: `{"version":2}`, wantNewer: &NewerError{Found: 2, Newest: 1}},
		{name: "negative version", data: `{"version":-1}`, wantErr: true},
		{name: "fractional version", data: `{"version":1.5}`, wantErr: true},
		{name: "version as text", data: `{"version":"1"}`, wantErr: true},
		{name: "null version", data: `{"version":null}`, wantErr: true},
		{name: "huge version", data: `{"version":1e400}`, wantErr: true},
		{name: "null document", data: `null`, wantErr: true},
		{name: "array", data: `[1]`, wantErr: true},
		{name: "not JSON", data: `{"version":`, wantErr: true},
		{name: "empty", data: ``, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, version, err := Read([]byte(test.data), readers)
			var newer *NewerError
			switch {
			case test.wantNewer != nil:
				if !errors.As(err, &newer) || *newer != *test.wantNewer {
					t.Fatalf("Read(%s) error = %v, want %+v", test.data, err, *test.wantNewer)
				}
			case test.wantErr:
				if err == nil || errors.As(err, &newer) {
					t.Fatalf("Read(%s) = %q, %d, %v; want an error that is not NewerError", test.data, value, version, err)
				}
			default:
				if err != nil || value != test.wantValue || version != test.wantVersion {
					t.Fatalf("Read(%s) = %q, %d, %v; want %q, %d", test.data, value, version, err, test.wantValue, test.wantVersion)
				}
			}
		})
	}
}

func TestReadPassesTheReadersError(t *testing.T) {
	failing := []func([]byte) (string, error){
		func([]byte) (string, error) { return "", fmt.Errorf("bad layout") },
	}
	if _, _, err := Read([]byte(`{}`), failing); err == nil || !strings.Contains(err.Error(), "bad layout") {
		t.Fatalf("error = %v, want the reader's", err)
	}
}

func TestNewerErrorNamesBothVersions(t *testing.T) {
	err := &NewerError{Found: 3, Newest: 1}
	want := "written by a newer version of the workshop (format 3, this build reads up to 1)"
	if err.Error() != want {
		t.Fatalf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestAccessAfterALoad(t *testing.T) {
	tests := []struct {
		name    string
		loadErr error
		want    Access
	}{
		{"loaded", nil, Writable},
		{"damaged", errors.New("parse: unexpected end"), Writable},
		{"newer", fmt.Errorf("parse state.json: %w", &NewerError{Found: 2, Newest: 1}), ReadOnly},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := AccessAfter(test.loadErr); got != test.want {
				t.Fatalf("AccessAfter(%v) = %v, want %v", test.loadErr, got, test.want)
			}
		})
	}
}

func TestPlainDecodesTheWholeObject(t *testing.T) {
	type sample struct {
		Version int    `json:"version"`
		Name    string `json:"name"`
	}
	got, err := Plain[sample]([]byte(`{"version":1,"name":"x"}`))
	if err != nil || got != (sample{Version: 1, Name: "x"}) {
		t.Fatalf("Plain = %+v, %v", got, err)
	}
	if _, err := Plain[sample]([]byte(`{"name":1}`)); err == nil {
		t.Fatal("Plain of a wrong type gave no error")
	}
}
