package view

import "testing"

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

func TestTextFieldPointerSelection(t *testing.T) {
	tests := []struct {
		name     string
		run      func(f *TextField)
		wantText string
		wantSel  string
	}{
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
