package view

import "testing"

const poem = "abcdef\nab\nabcdef"

func area(text string) *TextField {
	f := NewTextField(text)
	f.Multiline = true
	f.ViewLines = 2
	return &f
}

func TestTripleClickSelectsTheLineInAnArea(t *testing.T) {
	f := area(poem)
	f.PressN(8, false, 3)
	if f.SelectedText() != "ab\n" && f.SelectedText() != "ab" {
		t.Fatalf("selection %q", f.SelectedText())
	}
	g := area(poem)
	g.PressN(8, false, 2)
	if g.SelectedText() != "ab" {
		t.Fatalf("double click: %q", g.SelectedText())
	}
}
