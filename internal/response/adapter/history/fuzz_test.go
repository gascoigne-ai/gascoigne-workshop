package history

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzLoad(fuzz *testing.F) {
	fuzz.Add([]byte(`{"key":"a","entries":[{"status":200}]}`))
	fuzz.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		if err := os.WriteFile(path(dir, "a"), data, 0o600); err != nil {
			t.Skip()
		}
		_, _ = Load(dir, "a")
	})
}

func FuzzLoadSent(fuzz *testing.F) {
	fuzz.Add([]byte("{\"path\":\"a\"}\nnot json\n{\"path\":\"b\"}\n"))
	fuzz.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, sentFile), data, 0o600); err != nil {
			t.Skip()
		}
		_, _ = LoadSent(dir)
	})
}
