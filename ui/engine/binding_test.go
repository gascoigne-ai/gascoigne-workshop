package engine

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestParseAndFormatBindings(t *testing.T) {
	tests := []struct {
		in   string
		want Binding
		out  string // the canonical text; "" for the same as in
	}{
		{"Ctrl+Enter", Binding{Key: ebiten.KeyEnter, Ctrl: true, Bound: true}, ""},
		{"Ctrl+Shift+Tab", Binding{Key: ebiten.KeyTab, Ctrl: true, Shift: true, Bound: true}, ""},
		{"shift+ctrl+tab", Binding{Key: ebiten.KeyTab, Ctrl: true, Shift: true, Bound: true}, "Ctrl+Shift+Tab"},
		{"Alt+Up", Binding{Key: ebiten.KeyArrowUp, Alt: true, Bound: true}, ""},
		{"Alt+ArrowDown", Binding{Key: ebiten.KeyArrowDown, Alt: true, Bound: true}, "Alt+Down"},
		{"Ctrl+,", Binding{Key: ebiten.KeyComma, Ctrl: true, Bound: true}, ""},
		{"F5", Binding{Key: ebiten.KeyF5, Bound: true}, ""},
		{"Delete", Binding{Key: ebiten.KeyDelete, Bound: true}, ""},
		{"Ctrl+Alt+Shift+K", Binding{Key: ebiten.KeyK, Ctrl: true, Alt: true, Shift: true, Bound: true}, ""},
		{"ctrl+q", Binding{Key: ebiten.KeyQ, Ctrl: true, Bound: true}, "Ctrl+Q"},
		{"", Binding{}, ""},
	}
	for _, tt := range tests {
		got, err := ParseBinding(tt.in)
		if err != nil || got != tt.want {
			t.Errorf("ParseBinding(%q) = %+v, %v; want %+v", tt.in, got, err, tt.want)
			continue
		}
		want := tt.out
		if want == "" {
			want = tt.in
		}
		if got.String() != want {
			t.Errorf("%q formats as %q, want %q", tt.in, got.String(), want)
		}
	}
}

func TestParseBindingRejectsNonsense(t *testing.T) {
	for _, in := range []string{"Ctrl+", "Ctrl", "Shift+Alt", "Banana", "Ctrl+Ctrl+T", "Ctrl+T+Y", "+", "Ctrl++"} {
		if b, err := ParseBinding(in); err == nil {
			t.Errorf("ParseBinding(%q) = %+v, want an error", in, b)
		}
	}
}
