package store

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzLoad(fuzz *testing.F) {
	fuzz.Add([]byte(`{"name":"a","order":["a.req.json"]}`), []byte(`{"method":"GET","url":"https://x.test"}`), []byte(`{}`))
	fuzz.Fuzz(func(t *testing.T, folder, request, secrets []byte) {
		dir := t.TempDir()
		files := map[string][]byte{
			folderFile:                      folder,
			"a" + reqSuffix:                 request,
			secretsFile:                     secrets,
			filepath.Join(envDir, "e.json"): request,
		}
		for name, data := range files {
			path := filepath.Join(dir, name)
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Skip()
			}
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Skip()
			}
		}
		_, _ = disk.Load(dir)
	})
}
