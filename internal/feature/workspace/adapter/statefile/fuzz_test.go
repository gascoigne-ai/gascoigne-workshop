package statefile

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzLoad(fuzz *testing.F) {
	fuzz.Add([]byte(`{"collections":[]}`))
	fuzz.Fuzz(func(t *testing.T, data []byte) {
		path := filepath.Join(t.TempDir(), "state.json")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Skip()
		}
		_, _ = disk.Load(path)
	})
}

// disk is the store under test.
var disk Files
