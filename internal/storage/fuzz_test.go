package storage

import "testing"

// FuzzRead feeds Read any bytes: it returns an error or a version within
// the table, never panics.
func FuzzRead(fuzz *testing.F) {
	for _, seed := range []string{`{}`, `{"version":1}`, `{"version":2}`, `{"version":-1}`, `{"version":1.5}`, `{"version":"1"}`, `null`, `[]`, ``} {
		fuzz.Add([]byte(seed))
	}
	fuzz.Fuzz(func(t *testing.T, data []byte) {
		_, version, err := Read(data, readers)
		if err == nil && (version < 0 || version >= len(readers)) {
			t.Fatalf("Read(%q) gave version %d with no error", data, version)
		}
	})
}
