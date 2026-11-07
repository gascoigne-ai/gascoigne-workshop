package view

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// What the window system delivers for dead keys (a dead accent then a
// letter) and for an input method is the composed text as ordinary typed
// runes; the editors must take them as one character each. The in-progress
// composition itself is not delivered by Ebitengine and cannot be shown.
func TestComposedTextIsInsertedAsWholeCharacters(t *testing.T) {
	tests := []struct {
		name       string
		typed      string
		want       string
		wantCursor int
	}{
		{"dead key accent", "é", "é", 1},
		{"dead key tilde", "ñ", "ñ", 1},
		{"decomposed accent stays two runes", "é", "é", 2},
		{"an input method result", "日本語", "日本語", 3},
		{"a character outside the basic plane", "😀", "😀", 1},
		{"control characters are dropped", "a\x07b", "ab", 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			field := NewTextField("")
			field.Apply([]rune(test.typed), nil, false)
			if field.Text != test.want || field.Cursor != test.wantCursor {
				t.Fatalf("text %q cursor %d, want %q %d", field.Text, field.Cursor, test.want, test.wantCursor)
			}
		})
	}
}

func TestBackspaceRemovesOneComposedCharacter(t *testing.T) {
	field := NewTextField("")
	field.Apply([]rune("日本語"), nil, false)
	field.Apply(nil, []core.Edit{core.EditBackspace}, false)
	if field.Text != "日本" {
		t.Fatalf("text %q, want %q", field.Text, "日本")
	}
}

func TestComposedTextReachesAMultilineEditor(t *testing.T) {
	area := NewTextArea("")
	area.Apply([]rune("ça va\n"), nil, false)
	area.Apply([]rune("日本"), nil, false)
	if area.Text != "ça va日本" { // the line break comes from Enter, not from typed text
		t.Fatalf("text %q", area.Text)
	}
}
