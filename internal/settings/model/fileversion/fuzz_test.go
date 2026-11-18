package fileversion

import "testing"

// FuzzRead feeds Read any bytes: it returns an error or a version within
// the table, never panics.
func FuzzRead(f *testing.F) {
	for _, seed := range []string{`{}`, `{"version":1}`, `{"version":2}`, `{"version":-1}`, `{"version":1.5}`, `{"version":"1"}`, `null`, `[]`, ``} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		_, version, err := Read(data, readers)
		if err == nil && (version < 0 || version >= len(readers)) {
			t.Fatalf("Read(%q) gave version %d with no error", data, version)
		}
	})
}
