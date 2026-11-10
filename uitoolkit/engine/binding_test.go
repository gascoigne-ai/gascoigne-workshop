package engine

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
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

func TestBindingMatchesOnlyTheExactModifiers(t *testing.T) {
	ctrlTab := Binding{Key: ebiten.KeyTab, Ctrl: true, Bound: true}
	tests := []struct {
		name    string
		binding Binding
		input   core.Input
		key     ebiten.Key
		want    bool
	}{
		{"same key and modifiers", ctrlTab, core.Input{Ctrl: true}, ebiten.KeyTab, true},
		{"extra shift", ctrlTab, core.Input{Ctrl: true, Shift: true}, ebiten.KeyTab, false},
		{"missing ctrl", ctrlTab, core.Input{}, ebiten.KeyTab, false},
		{"other key", ctrlTab, core.Input{Ctrl: true}, ebiten.KeyEnter, false},
		{"unbound never matches", Binding{Key: ebiten.KeyTab}, core.Input{}, ebiten.KeyTab, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.binding.Matches(tt.input, tt.key); got != tt.want {
				t.Fatalf("Matches = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBindingFromReadsTheCombinationPressedThisFrame(t *testing.T) {
	tests := []struct {
		name   string
		input  core.Input
		want   Binding
		wantOK bool
	}{
		{"key with modifiers", core.Input{Keys: []ebiten.Key{ebiten.KeyTab}, Ctrl: true, Shift: true}, Binding{Key: ebiten.KeyTab, Ctrl: true, Shift: true, Bound: true}, true},
		{"modifier keys alone are waited out", core.Input{Keys: []ebiten.Key{ebiten.KeyControlLeft}, Ctrl: true}, Binding{}, false},
		{"modifier then key takes the key", core.Input{Keys: []ebiten.Key{ebiten.KeyShiftLeft, ebiten.KeyA}, Shift: true}, Binding{Key: ebiten.KeyA, Shift: true, Bound: true}, true},
		{"escape alone", core.Input{Escape: true}, Binding{Key: ebiten.KeyEscape, Bound: true}, true},
		{"nothing pressed", core.Input{}, Binding{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := BindingFrom(tt.input)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("BindingFrom = %+v, %v; want %+v, %v", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestBindingTypesText(t *testing.T) {
	tests := []struct {
		name    string
		binding Binding
		want    bool
	}{
		{"plain letter", Binding{Key: ebiten.KeyA, Bound: true}, true},
		{"shift letter", Binding{Key: ebiten.KeyA, Shift: true, Bound: true}, true},
		{"ctrl letter", Binding{Key: ebiten.KeyA, Ctrl: true, Bound: true}, false},
		{"alt arrow", Binding{Key: ebiten.KeyArrowUp, Alt: true, Bound: true}, false},
		{"function key", Binding{Key: ebiten.KeyF5, Bound: true}, false},
		{"delete", Binding{Key: ebiten.KeyDelete, Bound: true}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.binding.TypesText(); got != tt.want {
				t.Fatalf("TypesText = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsFunctionKey(t *testing.T) {
	for key, want := range map[ebiten.Key]bool{ebiten.KeyF1: true, ebiten.KeyF12: true, ebiten.KeyA: false, ebiten.KeyEnter: false} {
		if got := IsFunctionKey(key); got != want {
			t.Errorf("IsFunctionKey(%v) = %v, want %v", key, got, want)
		}
	}
}
