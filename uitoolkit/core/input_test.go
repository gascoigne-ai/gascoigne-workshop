package core

import (
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
