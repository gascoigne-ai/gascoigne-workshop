package core

import (
	"math"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestSamplerCountsClicksByItsOwnClock(t *testing.T) {
	now := time.Unix(100, 0)
	s := &Sampler{Now: func() time.Time { return now }}
	if got := s.now(); !got.Equal(now) {
		t.Fatalf("now = %v, want the injected clock %v", got, now)
	}
}

func TestInputConsumeRemovesTheKeysFromEveryForm(t *testing.T) {
	in := Input{
		Enter: true, Escape: true, Tab: true,
		Edits: []Edit{EditUp, EditDown, EditLeft},
		Keys:  []ebiten.Key{ebiten.KeyEnter, ebiten.KeyArrowUp, ebiten.KeyArrowDown, ebiten.KeyEscape, ebiten.KeyTab, ebiten.KeyA},
	}
	in.Consume(ebiten.KeyEnter, ebiten.KeyArrowUp, ebiten.KeyArrowDown, ebiten.KeyEscape, ebiten.KeyTab)
	if in.Enter || in.Escape || in.Tab {
		t.Errorf("flags left: enter %v escape %v tab %v", in.Enter, in.Escape, in.Tab)
	}
	if len(in.Edits) != 1 || in.Edits[0] != EditLeft {
		t.Errorf("edits = %v, want only EditLeft", in.Edits)
	}
	if len(in.Keys) != 1 || in.Keys[0] != ebiten.KeyA {
		t.Errorf("keys = %v, want only A", in.Keys)
	}
}

// A browser that goes up a folder on Backspace must take the Backspace
// from the path field under it, as it takes Up and Down.
func TestInputConsumeRemovesEveryEditingKeysEdit(t *testing.T) {
	in := Input{
		Edits: []Edit{EditBackspace, EditDelete, EditHome, EditPageDown, EditLeft},
		Keys:  []ebiten.Key{ebiten.KeyBackspace},
	}
	in.Consume(ebiten.KeyBackspace, ebiten.KeyDelete, ebiten.KeyHome, ebiten.KeyPageDown)
	if len(in.Edits) != 1 || in.Edits[0] != EditLeft {
		t.Errorf("edits = %v, want only EditLeft", in.Edits)
	}
	if len(in.Keys) != 0 {
		t.Errorf("keys = %v, want none", in.Keys)
	}
}

func TestInputConsumeLeavesOtherKeys(t *testing.T) {
	in := Input{Enter: true, Escape: true, Edits: []Edit{EditUp}, Keys: []ebiten.Key{ebiten.KeyEnter}}
	in.Consume(ebiten.KeyEscape)
	if !in.Enter || in.Escape || len(in.Edits) != 1 || len(in.Keys) != 1 {
		t.Fatalf("Consume(Escape) changed more than Escape: %+v", in)
	}
}

func TestInputPressed(t *testing.T) {
	in := Input{Keys: []ebiten.Key{ebiten.KeyArrowDown}}
	if !in.Pressed(ebiten.KeyArrowDown) || in.Pressed(ebiten.KeyArrowUp) {
		t.Fatal("Pressed must report only keys that went down")
	}
}

func TestInputRestartsCaret(t *testing.T) {
	tests := []struct {
		name  string
		input Input
		want  bool
	}{
		{"typing", Input{Typed: []rune("a")}, true},
		{"key down", Input{Keys: []ebiten.Key{ebiten.KeyA}}, true},
		{"editing key", Input{Edits: []Edit{EditLeft}}, true},
		{"click", Input{Click: true}, true},
		{"enter", Input{Enter: true}, true},
		{"paste", Input{Paste: true}, true},
		{"pointer moving only", Input{X: 10, Y: 20}, false},
		{"wheel only", Input{Wheel: 1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.input.RestartsCaret(); got != tt.want {
				t.Fatalf("RestartsCaret = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKeyRepeatFiresAfterTheDelayThenEveryInterval(t *testing.T) {
	// At 60 ticks a second, 600 ms is 36 ticks and 25 ms is 1.5, rounded to 2.
	repeat := KeyRepeat{Delay: 600 * time.Millisecond, Interval: 25 * time.Millisecond}
	tests := []struct {
		name string
		held int
		want bool
	}{
		{"first tick", 1, true},
		{"held a little", 2, false},
		{"just before the delay", 35, false},
		{"at the delay", 36, true},
		{"between repeats", 37, false},
		{"next repeat", 38, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := repeat.Fires(tt.held, 60); got != tt.want {
				t.Errorf("Fires(%d) = %v, want %v", tt.held, got, tt.want)
			}
		})
	}
}

func TestKeyRepeatLeftUnsetKeepsTheBuiltInTiming(t *testing.T) {
	var unset KeyRepeat
	// The built-in timing: after 25 ticks, then every 3.
	if !unset.Fires(25, 60) || unset.Fires(26, 60) || !unset.Fires(28, 60) {
		t.Fatal("an unset KeyRepeat must repeat after 25 ticks and then every 3")
	}
}

// Scaled divided by the scale, so 0 gave int(Inf) pointer positions.
func TestScaledIgnoresAScaleThatIsNotPositive(t *testing.T) {
	in := Input{X: 40, Y: 20}
	for _, scale := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if got := in.Scaled(scale); got.X != 40 || got.Y != 20 {
			t.Errorf("Scaled(%v) = %d,%d, want 40,20", scale, got.X, got.Y)
		}
	}
	if got := in.Scaled(2); got.X != 20 || got.Y != 10 {
		t.Errorf("Scaled(2) = %d,%d, want 20,10", got.X, got.Y)
	}
}
