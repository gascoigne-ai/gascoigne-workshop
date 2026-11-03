package ui

import "testing"

func TestTextFieldApply(t *testing.T) {
	tests := []struct {
		name       string
		start      string
		cursor     int
		typed      string
		edits      []Edit
		want       string
		wantCursor int
		changed    bool
	}{
		{"type at end", "ab", 2, "c", nil, "abc", 3, true},
		{"type in middle", "ac", 1, "b", nil, "abc", 2, true},
		{"backspace", "abc", 3, "", []Edit{EditBackspace}, "ab", 2, true},
		{"backspace at start does nothing", "abc", 0, "", []Edit{EditBackspace}, "abc", 0, false},
		{"delete", "abc", 0, "", []Edit{EditDelete}, "bc", 0, true},
		{"delete at end does nothing", "abc", 3, "", []Edit{EditDelete}, "abc", 3, false},
		{"type then home", "bc", 2, "a", []Edit{EditHome}, "bca", 0, true},
		{"left right end", "abc", 3, "", []Edit{EditLeft, EditLeft, EditRight, EditEnd}, "abc", 3, false},
		{"control runes ignored", "a", 1, "\n\t\x7f", nil, "a", 1, false},
		{"multibyte runes", "żó", 2, "ł", []Edit{EditBackspace, EditBackspace}, "ż", 1, true},
		{"stale cursor clamped", "ab", 99, "c", nil, "abc", 3, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := TextField{Text: tt.start, Cursor: tt.cursor}
			changed := f.Apply([]rune(tt.typed), tt.edits)
			if f.Text != tt.want || f.Cursor != tt.wantCursor || changed != tt.changed {
				t.Fatalf("got %q cursor %d changed %v; want %q cursor %d changed %v",
					f.Text, f.Cursor, changed, tt.want, tt.wantCursor, tt.changed)
			}
		})
	}
}
