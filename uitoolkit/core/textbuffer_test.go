package core

import "testing"

func TestTextBufferApply(t *testing.T) {
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
			f := TextBuffer{Text: tt.start, Cursor: tt.cursor}
			changed := f.Apply([]rune(tt.typed), tt.edits, false)
			if f.Text != tt.want || f.Cursor != tt.wantCursor || changed != tt.changed {
				t.Fatalf("got %q cursor %d changed %v; want %q cursor %d changed %v",
					f.Text, f.Cursor, changed, tt.want, tt.wantCursor, tt.changed)
			}
		})
	}
}

func TestTextBufferSelection(t *testing.T) {
	tests := []struct {
		name     string
		run      func(f *TextBuffer)
		wantText string
		wantSel  string
	}{
		{"shift+right selects forward", func(f *TextBuffer) {
			f.Cursor = 1
			f.Apply(nil, []Edit{EditRight, EditRight}, true)
		}, "hello", "el"},
		{"shift+home selects to start", func(f *TextBuffer) {
			f.Cursor = 3
			f.Apply(nil, []Edit{EditHome}, true)
		}, "hello", "hel"},
		{"select all", func(f *TextBuffer) { f.SelectAll() }, "hello", "hello"},
		{"left collapses to the low edge", func(f *TextBuffer) {
			f.Cursor = 1
			f.Apply(nil, []Edit{EditRight, EditRight}, true)
			f.Apply(nil, []Edit{EditLeft}, false)
		}, "hello", ""},
		{"typing replaces the selection", func(f *TextBuffer) {
			f.SelectAll()
			f.Apply([]rune("x"), nil, false)
		}, "x", ""},
		{"backspace deletes the selection only", func(f *TextBuffer) {
			f.Cursor = 1
			f.Apply(nil, []Edit{EditRight, EditRight}, true)
			f.Apply(nil, []Edit{EditBackspace}, false)
		}, "hlo", ""},
		{"paste replaces the selection and strips line breaks", func(f *TextBuffer) {
			f.Cursor = 1
			f.Apply(nil, []Edit{EditRight}, true)
			f.Insert("A\nB\x00")
		}, "hA Bllo", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewTextBuffer("hello")
			tt.run(&f)
			if f.Text != tt.wantText || f.SelectedText() != tt.wantSel {
				t.Fatalf("text %q selection %q; want %q and %q", f.Text, f.SelectedText(), tt.wantText, tt.wantSel)
			}
		})
	}
}

func TestTextBufferCutAndCopyText(t *testing.T) {
	f := NewTextBuffer("hello world")
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

func TestWordLeftAndRightStopAtWordBoundaries(t *testing.T) {
	text := []rune("hello, wörld_1 /x")
	//              01234567890123456
	left := map[int]int{0: 0, 5: 0, 6: 0, 7: 0, 8: 7, 13: 7, 14: 7, 16: 7, 17: 16}
	for from, want := range left {
		if got := wordLeft(text, from); got != want {
			t.Errorf("wordLeft(%d) = %d, want %d", from, got, want)
		}
	}
	right := map[int]int{0: 5, 5: 14, 6: 14, 13: 14, 14: 17, 17: 17}
	for from, want := range right {
		if got := wordRight(text, from); got != want {
			t.Errorf("wordRight(%d) = %d, want %d", from, got, want)
		}
	}
}

func TestWordWiseMovementAndDeletionActFromTheCursor(t *testing.T) {
	tests := []struct {
		name       string
		cursor     int
		edit       Edit
		extend     bool
		wantText   string
		wantCursor int
		wantSel    string
	}{
		{"word left", 11, EditWordLeft, false, "one two three", 8, ""},
		{"word right", 4, EditWordRight, false, "one two three", 7, ""},
		{"shift+word left selects", 13, EditWordLeft, true, "one two three", 8, "three"},
		{"delete word left", 7, EditDeleteWordLeft, false, "one  three", 4, ""},
		{"delete word right", 4, EditDeleteWordRight, false, "one  three", 4, ""},
		{"delete word at start does nothing", 0, EditDeleteWordLeft, false, "one two three", 0, ""},
		{"delete word with a selection deletes the selection only", 0, EditDeleteWordRight, false, "one two three", 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewTextBuffer("one two three")
			f.Cursor = tt.cursor
			if tt.name == "delete word with a selection deletes the selection only" {
				f.Anchor, f.Cursor, f.HasSel = 0, 3, true
				tt.wantText, tt.wantCursor = " two three", 0
			}
			f.Apply(nil, []Edit{tt.edit}, tt.extend)
			if f.Text != tt.wantText || f.Cursor != tt.wantCursor || f.SelectedText() != tt.wantSel {
				t.Fatalf("text %q cursor %d sel %q; want %q %d %q", f.Text, f.Cursor, f.SelectedText(), tt.wantText, tt.wantCursor, tt.wantSel)
			}
		})
	}
}

func TestUndoRedoCoalescesTyping(t *testing.T) {
	f := NewTextBuffer("")
	for _, r := range "abc" {
		f.Apply([]rune{r}, nil, false)
	}
	f.Apply(nil, []Edit{EditLeft}, false) // moving starts a new group
	f.Apply([]rune("X"), nil, false)
	if f.Text != "abXc" {
		t.Fatalf("text = %q", f.Text)
	}

	if !f.Undo() || f.Text != "abc" {
		t.Fatalf("first undo: %q", f.Text)
	}
	if !f.Undo() || f.Text != "" {
		t.Fatalf("second undo must remove the whole typed word, got %q", f.Text)
	}
	if f.Undo() {
		t.Fatal("nothing left to undo")
	}
	if !f.Redo() || f.Text != "abc" || !f.Redo() || f.Text != "abXc" {
		t.Fatalf("redo: %q", f.Text)
	}
	if f.Redo() {
		t.Fatal("nothing left to redo")
	}
}

func TestUndoGroupsBackspacesPasteAndCutSeparately(t *testing.T) {
	f := NewTextBuffer("hello world")
	f.Apply(nil, []Edit{EditBackspace, EditBackspace}, false) // "hello wor"
	f.Insert("XYZ")                                           // paste: its own step
	f.SelectAll()
	f.DeleteSelection() // cut
	if f.Text != "" {
		t.Fatalf("text = %q", f.Text)
	}
	f.Undo()
	if f.Text != "hello worXYZ" {
		t.Fatalf("undo cut: %q", f.Text)
	}
	f.Undo()
	if f.Text != "hello wor" {
		t.Fatalf("undo paste: %q", f.Text)
	}
	f.Undo()
	if f.Text != "hello world" {
		t.Fatalf("undo both backspaces: %q", f.Text)
	}
}

func TestEditAfterUndoClearsRedoAndRestoresCursor(t *testing.T) {
	f := NewTextBuffer("ab")
	f.Apply([]rune("c"), nil, false)
	f.Undo()
	if f.Cursor != 2 {
		t.Fatalf("cursor after undo = %d, want 2", f.Cursor)
	}
	f.Apply([]rune("d"), nil, false)
	if f.Redo() {
		t.Fatal("a new edit must clear the redo stack")
	}
}

func TestUndoHistoryIsBounded(t *testing.T) {
	f := NewTextBuffer("")
	for i := 0; i < historyLimit+50; i++ {
		f.Insert("x") // paste-style edits never coalesce
	}
	n := 0
	for f.Undo() {
		n++
	}
	if n != historyLimit {
		t.Fatalf("undo steps = %d, want %d", n, historyLimit)
	}
}

func TestMaskedFieldHidesItsTextAndNeverCopiesIt(t *testing.T) {
	f := NewTextBuffer("s3cret")
	f.Masked = true
	if got := f.Display(); got != "••••••" {
		t.Fatalf("Display = %q", got)
	}
	plain := NewTextBuffer("abc")
	if plain.Display() != "abc" {
		t.Fatalf("an unmasked field displays its text, got %q", plain.Display())
	}

	c, err := NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	clip := &fakeClipboard{text: "before"}
	c.Clipboard = clip
	f.SelectAll()
	c.In = Input{Copy: true}
	f.RunShortcuts(c)
	c.In = Input{Cut: true}
	f.RunShortcuts(c)
	if clip.text != "before" {
		t.Fatalf("a masked field must not put its text on the clipboard, got %q", clip.text)
	}
	if f.Text != "s3cret" {
		t.Fatalf("Cut on a masked field must not delete either, text %q", f.Text)
	}

	clip.text = "pasted"
	f.SelectAll()
	c.In = Input{Paste: true}
	f.RunShortcuts(c)
	if f.Text != "pasted" {
		t.Fatalf("pasting into a masked field works, got %q", f.Text)
	}
}

func TestCanUndoAndCanRedoFollowTheHistory(t *testing.T) {
	f := NewTextBuffer("")
	if f.CanUndo() || f.CanRedo() {
		t.Fatal("a fresh field has no history")
	}
	f.Apply([]rune("a"), nil, false)
	if !f.CanUndo() || f.CanRedo() {
		t.Fatal("after typing: undo yes, redo no")
	}
	f.Undo()
	if f.CanUndo() || !f.CanRedo() {
		t.Fatal("after undo: redo yes, undo no")
	}
}
