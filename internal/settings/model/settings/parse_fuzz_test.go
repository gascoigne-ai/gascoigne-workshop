package settings

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Settings text with problems leaves the settings in effect unchanged, and
// no text panics.
func FuzzParse(f *testing.F) {
	for _, file := range []string{"settings-v0.json", "settings-v1.json"} {
		data, err := os.ReadFile(filepath.Join("testdata", file))
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data)
	}
	last := Default()
	f.Fuzz(func(t *testing.T, data []byte) {
		parsed, problems := Parse(data, last)
		if len(problems) > 0 && !reflect.DeepEqual(parsed, last) {
			t.Fatalf("text with problems changed the settings: %v", problems)
		}
	})
}
