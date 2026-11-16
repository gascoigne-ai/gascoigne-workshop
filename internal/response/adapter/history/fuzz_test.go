package history

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzLoad(f *testing.F) {
	f.Add([]byte(`{"key":"a","entries":[{"status":200}]}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		if err := os.WriteFile(path(dir, "a"), data, 0o600); err != nil {
			t.Skip()
		}
		_, _ = Load(dir, "a")
	})
}

func FuzzLoadSent(f *testing.F) {
	f.Add([]byte("{\"path\":\"a\"}\nnot json\n{\"path\":\"b\"}\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, sentFile), data, 0o600); err != nil {
			t.Skip()
		}
		_, _ = LoadSent(dir)
	})
}
