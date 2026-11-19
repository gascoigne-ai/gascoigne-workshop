package widget

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/theming"
)

func TestTextFieldApply(t *testing.T) {
	tests := []struct {
		name       string
		start      string
		cursor     int
		typed      string
		edits      []core.Edit
		want       string
		wantCursor int
		changed    bool
	}{
		{"type at end", "ab", 2, "c", nil, "abc", 3, true},
		{"type in middle", "ac", 1, "b", nil, "abc", 2, true},
		{"backspace", "abc", 3, "", []core.Edit{core.EditBackspace}, "ab", 2, true},
		{"backspace at start does nothing", "abc", 0, "", []core.Edit{core.EditBackspace}, "abc", 0, false},
		{"delete", "abc", 0, "", []core.Edit{core.EditDelete}, "bc", 0, true},
		{"delete at end does nothing", "abc", 3, "", []core.Edit{core.EditDelete}, "abc", 3, false},
		{"type then home", "bc", 2, "a", []core.Edit{core.EditHome}, "bca", 0, true},
		{"left right end", "abc", 3, "", []core.Edit{core.EditLeft, core.EditLeft, core.EditRight, core.EditEnd}, "abc", 3, false},
		{"control runes ignored", "a", 1, "\n\t\x7f", nil, "a", 1, false},
		{"multibyte runes", "żó", 2, "ł", []core.Edit{core.EditBackspace, core.EditBackspace}, "ż", 1, true},
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
		run      func(field *TextField)
		wantText string
		wantSel  string
	}{
		{"shift+right selects forward", func(field *TextField) {
			field.Cursor = 1
			field.Apply(nil, []core.Edit{core.EditRight, core.EditRight}, true)
		}, "hello", "el"},
		{"shift+home selects to start", func(field *TextField) {
			field.Cursor = 3
			field.Apply(nil, []core.Edit{core.EditHome}, true)
		}, "hello", "hel"},
		{"select all", func(field *TextField) { field.SelectAll() }, "hello", "hello"},
		{"left collapses to the low edge", func(field *TextField) {
			field.Cursor = 1
			field.Apply(nil, []core.Edit{core.EditRight, core.EditRight}, true)
			field.Apply(nil, []core.Edit{core.EditLeft}, false)
		}, "hello", ""},
		{"typing replaces the selection", func(field *TextField) {
			field.SelectAll()
			field.Apply([]rune("x"), nil, false)
		}, "x", ""},
		{"backspace deletes the selection only", func(field *TextField) {
			field.Cursor = 1
			field.Apply(nil, []core.Edit{core.EditRight, core.EditRight}, true)
			field.Apply(nil, []core.Edit{core.EditBackspace}, false)
		}, "hlo", ""},
		{"paste replaces the selection and strips line breaks", func(field *TextField) {
			field.Cursor = 1
			field.Apply(nil, []core.Edit{core.EditRight}, true)
			field.Insert("A\nB\x00")
		}, "hA Bllo", ""},
		{"drag selects between press and pointer", func(field *TextField) {
			field.Press(1, false)
			field.Drag(4)
		}, "hello", "ell"},
		{"shift+click extends the selection", func(field *TextField) {
			field.Cursor = 1
			field.Press(4, true)
		}, "hello", "ell"},
		{"press without drag selects nothing", func(field *TextField) { field.Press(2, false) }, "hello", ""},
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

// A word is a run of letters (any script, with their combining marks),
// digits and underscores; everything else, punctuation included, is
// skipped like a space, and a line break is crossed.
func TestWordBoundariesAcrossCharacterClasses(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		from      int
		wantLeft  int
		wantRight int
	}{
		{"punctuation inside a name separates words", "user.name", 4, 0, 9},
		{"a run of operators is skipped", "a == b", 1, 0, 6},
		{"url segments", "https://h/v1/users", 18, 13, 18},
		{"url from inside the scheme", "https://h/v1/users", 2, 0, 5},
		{"digits and underscores belong to the word", "v1.2_3", 2, 0, 6},
		{"precomposed accents are letters", "café x", 4, 0, 6},
		{"combining marks stay with their letter", "cafe\u0301 x", 0, 0, 5},
		{"cjk letters form a word", "日本語 テキスト", 8, 4, 8},
		{"whitespace runs are skipped", "one   two", 6, 0, 9},
		{"a line break is crossed", "one\n  two", 6, 0, 9},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			text := []rune(test.text)
			if got := wordLeft(text, test.from); got != test.wantLeft {
				t.Errorf("wordLeft(%d) = %d, want %d", test.from, got, test.wantLeft)
			}
			if got := wordRight(text, test.from); got != test.wantRight {
				t.Errorf("wordRight(%d) = %d, want %d", test.from, got, test.wantRight)
			}
		})
	}
}

func TestWordWiseMovementAndDeletionActFromTheCursor(t *testing.T) {
	tests := []struct {
		name       string
		cursor     int
		edit       core.Edit
		extend     bool
		wantText   string
		wantCursor int
		wantSel    string
	}{
		{"word left", 11, core.EditWordLeft, false, "one two three", 8, ""},
		{"word right", 4, core.EditWordRight, false, "one two three", 7, ""},
		{"shift+word left selects", 13, core.EditWordLeft, true, "one two three", 8, "three"},
		{"delete word left", 7, core.EditDeleteWordLeft, false, "one  three", 4, ""},
		{"delete word right", 4, core.EditDeleteWordRight, false, "one  three", 4, ""},
		{"delete word at start does nothing", 0, core.EditDeleteWordLeft, false, "one two three", 0, ""},
		{"delete word with a selection deletes the selection only", 0, core.EditDeleteWordRight, false, "one two three", 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewTextField("one two three")
			f.Cursor = tt.cursor
			if tt.name == "delete word with a selection deletes the selection only" {
				f.Anchor, f.Cursor, f.HasSel = 0, 3, true
				tt.wantText, tt.wantCursor = " two three", 0
			}
			f.Apply(nil, []core.Edit{tt.edit}, tt.extend)
			if f.Text != tt.wantText || f.Cursor != tt.wantCursor || f.SelectedText() != tt.wantSel {
				t.Fatalf("text %q cursor %d sel %q; want %q %d %q", f.Text, f.Cursor, f.SelectedText(), tt.wantText, tt.wantCursor, tt.wantSel)
			}
		})
	}
}

// Deleting to a line edge (Cmd+Backspace and Cmd+Delete on macOS) stays on
// the cursor's line; at the edge itself it takes the line break, so the
// lines join as in the macOS editors instead of the key doing nothing.
func TestDeleteToLineEdgesStaysOnTheLine(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		cursor     int
		edit       core.Edit
		wantText   string
		wantCursor int
	}{
		{"to the start of a single line", "one two", 4, core.EditDeleteToLineStart, "two", 0},
		{"to the end of a single line", "one two", 3, core.EditDeleteToLineEnd, "one", 3},
		{"to the start of the cursor's line only", "ab\ncd ef\ngh", 6, core.EditDeleteToLineStart, "ab\nef\ngh", 3},
		{"to the end of the cursor's line only", "ab\ncd ef\ngh", 5, core.EditDeleteToLineEnd, "ab\ncd\ngh", 5},
		{"at a line start the line joins the previous one", "ab\ncd", 3, core.EditDeleteToLineStart, "abcd", 2},
		{"at a line end the next line joins it", "ab\ncd", 2, core.EditDeleteToLineEnd, "abcd", 2},
		{"at the text start nothing happens", "ab", 0, core.EditDeleteToLineStart, "ab", 0},
		{"at the text end nothing happens", "ab", 2, core.EditDeleteToLineEnd, "ab", 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newMultiline(test.text)
			f.Cursor = test.cursor
			f.Apply(nil, []core.Edit{test.edit}, false)
			if f.Text != test.wantText || f.Cursor != test.wantCursor {
				t.Fatalf("text %q cursor %d; want %q %d", f.Text, f.Cursor, test.wantText, test.wantCursor)
			}
		})
	}
}

func TestDeleteToLineEdgeTakesASelectionAndIsItsOwnUndoStep(t *testing.T) {
	f := NewTextField("one two")
	f.Cursor = 7
	f.Apply([]rune("x"), nil, false)
	f.Apply(nil, []core.Edit{core.EditDeleteToLineStart}, false)
	if f.Text != "" {
		t.Fatalf("after delete to line start: %q", f.Text)
	}
	f.Undo()
	if f.Text != "one twox" {
		t.Fatalf("undo restored %q, want the typed text back", f.Text)
	}

	f.Anchor, f.Cursor, f.HasSel = 0, 3, true
	f.Apply(nil, []core.Edit{core.EditDeleteToLineEnd}, false)
	if f.Text != " twox" {
		t.Fatalf("with a selection: %q, want only the selection deleted", f.Text)
	}
}

func TestPressNSelectsWordOrAll(t *testing.T) {
	tests := []struct {
		name    string
		pos     int
		clicks  int
		wantSel string
	}{
		{"double click at a word start", 7, 2, "wörld_1"},
		{"double click inside a word", 10, 2, "wörld_1"},
		{"double click on spaces selects the run", 5, 2, ", "},
		{"triple click selects everything", 7, 3, "hello, wörld_1"},
		{"single click selects nothing", 7, 1, ""},
	}
	for _, tt := range tests {
		f := NewTextField("hello, wörld_1")
		f.PressN(tt.pos, false, tt.clicks)
		if f.SelectedText() != tt.wantSel {
			t.Errorf("%s: selection %q, want %q", tt.name, f.SelectedText(), tt.wantSel)
		}
	}
}

func TestRevealKeepsTheCursorInView(t *testing.T) {
	tests := []struct {
		name           string
		scroll, cursor int
		view           int
		want           int
	}{
		{"cursor inside", 0, 50, 100, 0},
		{"past the right edge", 0, 130, 100, 130 - 100 + revealMargin},
		{"before the left edge", 200, 150, 100, 150 - revealMargin},
		{"near the start never scrolls negative", 5, 2, 100, 0},
		{"text shorter than the view resets", 40, 30, 100, 0},
	}
	for _, tt := range tests {
		f := TextField{ScrollX: tt.scroll}
		f.Reveal(tt.cursor, tt.view)
		if f.ScrollX != tt.want {
			t.Errorf("%s: ScrollX = %d, want %d", tt.name, f.ScrollX, tt.want)
		}
	}
}

func TestUndoRedoCoalescesTyping(t *testing.T) {
	f := NewTextField("")
	for _, r := range "abc" {
		f.Apply([]rune{r}, nil, false)
	}
	f.Apply(nil, []core.Edit{core.EditLeft}, false) // moving starts a new group
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
	f := NewTextField("hello world")
	f.Apply(nil, []core.Edit{core.EditBackspace, core.EditBackspace}, false) // "hello wor"
	f.Insert("XYZ")                                                          // paste: its own step
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
	f := NewTextField("ab")
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
	f := NewTextField("")
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
	f := NewTextField("s3cret")
	f.Masked = true
	if got := f.Display(); got != "••••••" {
		t.Fatalf("Display = %q", got)
	}
	plain := NewTextField("abc")
	if plain.Display() != "abc" {
		t.Fatalf("an unmasked field displays its text, got %q", plain.Display())
	}

	c, err := core.NewCtx(theming.Dark())
	if err != nil {
		t.Fatal(err)
	}
	clip := &memClip{text: "before"}
	c.Clipboard = clip
	f.SelectAll()
	c.In = core.Input{Copy: true}
	f.RunShortcuts(c)
	c.In = core.Input{Cut: true}
	f.RunShortcuts(c)
	if clip.text != "before" {
		t.Fatalf("a masked field must not put its text on the clipboard, got %q", clip.text)
	}
	if f.Text != "s3cret" {
		t.Fatalf("Cut on a masked field must not delete either, text %q", f.Text)
	}

	clip.text = "pasted"
	f.SelectAll()
	c.In = core.Input{Paste: true}
	f.RunShortcuts(c)
	if f.Text != "pasted" {
		t.Fatalf("pasting into a masked field works, got %q", f.Text)
	}
}

func TestCanUndoAndCanRedoFollowTheHistory(t *testing.T) {
	f := NewTextField("")
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

// A selection kept from a longer text (the text was replaced from outside,
// as when a response or a format step rewrites the payload) reached past the
// end and made deleting panic with "slice bounds out of range".
func TestStaleSelectionPastTheTextEndIsClampedWhenDeleting(t *testing.T) {
	tests := []struct {
		name string
		edit core.Edit
	}{
		{"backspace", core.EditBackspace},
		{"delete", core.EditDelete},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := &TextField{Text: "abcd", Anchor: 9, Cursor: 2, HasSel: true}
			field.Apply(nil, []core.Edit{tt.edit}, false)
			if field.Text != "ab" {
				t.Errorf("Text = %q, want %q", field.Text, "ab")
			}
		})
	}
}

// FuzzEdit: any typed text, pasted text and run of edit keys leaves the
// cursor and selection inside the text and never panics, for a single
// line field and a multi-line one.
func FuzzEdit(fuzz *testing.F) {
	fuzz.Add("hello wörld", "typed ż", []byte{0, 1, 2, 3, 6, 7, 8, 9})
	fuzz.Add("a\r\nb\tc", "\x00\xff‮", []byte{16, 17, 18, 24, 25})
	fuzz.Add("", "", []byte{})
	fuzz.Fuzz(func(t *testing.T, start, typed string, keys []byte) {
		var edits []core.Edit
		for _, key := range keys {
			edits = append(edits, core.Edit(key%26))
		}
		field := NewTextField(start)
		field.Apply([]rune(typed), edits, true)
		field.Insert(typed)
		field.SelectAll()
		field.DeleteSelection()
		field.Undo()
		field.Redo()
		area := NewTextArea(start)
		area.Apply([]rune(typed), edits, false)
		area.Insert(typed)
		if lo, hi, ok := field.Selection(); ok && (lo < 0 || hi > len([]rune(field.Text))) {
			t.Fatalf("selection %d..%d outside %q", lo, hi, field.Text)
		}
	})
}

// Every edit kept a snapshot of the whole text for up to 100 steps, so a
// 1 MiB text held 100 MiB of history. The oldest snapshots go first once
// the total passes historyBytes.
func TestUndoHistoryIsBoundedByBytesOfLargeText(t *testing.T) {
	big := strings.Repeat("x", 1<<20)
	field := NewTextField(big)
	for range 60 {
		field.Cursor = len([]rune(field.Text))
		field.Insert("y")
		field.last = kindNone // each insert is its own undo step
	}
	total := 0
	for _, snapshot := range field.undo {
		total += len(snapshot.text)
	}
	if total > historyBytes+len(big) { // the newest snapshot may stand alone above the limit
		t.Fatalf("undo history holds %d bytes in %d snapshots, limit %d", total, len(field.undo), historyBytes)
	}
	if len(field.undo) == 0 {
		t.Fatal("the latest step must stay undoable")
	}
	if !field.Undo() || len(field.Text) != len(big)+59 {
		t.Errorf("undo of the latest insert: %d bytes, want %d", len(field.Text), len(big)+59)
	}
}
