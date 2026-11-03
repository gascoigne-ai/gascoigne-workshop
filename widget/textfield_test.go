package widget

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
			changed := f.Apply([]rune(tt.typed), tt.edits, false)
			if f.Text != tt.want || f.Cursor != tt.wantCursor || changed != tt.changed {
				t.Fatalf("got %q cursor %d changed %v; want %q cursor %d changed %v",
					f.Text, f.Cursor, changed, tt.want, tt.wantCursor, tt.changed)
			}
		})
	}
}

func TestTextFieldSelection(t *testing.T) {
	tests := []struct {
		name     string
		run      func(f *TextField)
		wantText string
		wantSel  string
	}{
		{"shift+right selects forward", func(f *TextField) {
			f.Cursor = 1
			f.Apply(nil, []Edit{EditRight, EditRight}, true)
		}, "hello", "el"},
		{"shift+home selects to start", func(f *TextField) {
			f.Cursor = 3
			f.Apply(nil, []Edit{EditHome}, true)
		}, "hello", "hel"},
		{"select all", func(f *TextField) { f.SelectAll() }, "hello", "hello"},
		{"left collapses to the low edge", func(f *TextField) {
			f.Cursor = 1
			f.Apply(nil, []Edit{EditRight, EditRight}, true)
			f.Apply(nil, []Edit{EditLeft}, false)
		}, "hello", ""},
		{"typing replaces the selection", func(f *TextField) {
			f.SelectAll()
			f.Apply([]rune("x"), nil, false)
		}, "x", ""},
		{"backspace deletes the selection only", func(f *TextField) {
			f.Cursor = 1
			f.Apply(nil, []Edit{EditRight, EditRight}, true)
			f.Apply(nil, []Edit{EditBackspace}, false)
		}, "hlo", ""},
		{"paste replaces the selection and strips line breaks", func(f *TextField) {
			f.Cursor = 1
			f.Apply(nil, []Edit{EditRight}, true)
			f.Insert("A\nB\x00")
		}, "hA Bllo", ""},
		{"drag selects between press and pointer", func(f *TextField) {
			f.Press(1, false)
			f.Drag(4)
		}, "hello", "ell"},
		{"shift+click extends the selection", func(f *TextField) {
			f.Cursor = 1
			f.Press(4, true)
		}, "hello", "ell"},
		{"press without drag selects nothing", func(f *TextField) { f.Press(2, false) }, "hello", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewTextField("hello")
			tt.run(&f)
			if f.Text != tt.wantText || f.SelectedText() != tt.wantSel {
				t.Fatalf("text %q selection %q; want %q and %q", f.Text, f.SelectedText(), tt.wantText, tt.wantSel)
			}
		})
	}
}

func TestTextFieldCutAndCopyText(t *testing.T) {
	f := NewTextField("hello world")
	f.Anchor, f.Cursor, f.HasSel = 6, 11, true
	if got := f.SelectedText(); got != "world" {
		t.Fatalf("SelectedText = %q", got)
	}
	if !f.DeleteSelection() || f.Text != "hello " || f.Cursor != 6 {
		t.Fatalf("after cut: %q cursor %d", f.Text, f.Cursor)
	}
	if f.DeleteSelection() {
		t.Fatal("deleting with no selection must report false")
	}
}
