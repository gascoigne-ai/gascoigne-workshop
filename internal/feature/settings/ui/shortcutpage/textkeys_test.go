package shortcutpage

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
)

func TestEveryTextKeyIsRefused(t *testing.T) {
	if len(textKeys) != 6 {
		t.Fatalf("%d text keys, want 6", len(textKeys))
	}
	for _, entry := range textKeys {
		for _, keys := range entry.keys {
			binding, err := engine.ParseBinding(keys)
			if err != nil {
				t.Fatalf("%s: %v", keys, err)
			}
			if Refuse(binding) == nil {
				t.Errorf("%s (%s) can be given to an action", keys, entry.action)
			}
		}
	}
}

func TestRedoShowsBothKeysJoinedWithOr(t *testing.T) {
	tests := []struct{ goos, want string }{
		{"linux", "Ctrl+Y or Ctrl+Shift+Z"},
		{"darwin", "Cmd+Y or Cmd+Shift+Z"},
	}
	for _, test := range tests {
		t.Run(test.goos, func(t *testing.T) {
			for _, row := range textKeyRows(test.goos) {
				if row.Action == "Redo" {
					if row.Keys != test.want {
						t.Fatalf("Redo shows %q, want %q", row.Keys, test.want)
					}
					return
				}
			}
			t.Fatal("no Redo row")
		})
	}
}
