package engine

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/ui/widget"
)

func TestMustDraw(t *testing.T) {
	cases := []struct {
		name                 string
		tick                 int
		active, caret, alway bool
		want                 bool
	}{
		{"idle", 1, false, false, false, false},
		{"input", 1, true, false, false, true},
		{"caret on blink tick", BlinkTicks, false, true, false, true},
		{"caret off blink tick", BlinkTicks + 1, false, true, false, false},
		{"blink tick without caret", BlinkTicks, false, false, false, false},
		{"heartbeat", HeartbeatTicks, false, false, false, true},
		{"always", 1, false, false, true, true},
	}
	for _, c := range cases {
		if got := MustDraw(c.tick, c.active, c.caret, c.alway); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestPacerSlowsDownWhenIdle(t *testing.T) {
	var p Pacer
	if p.TPS() != ActiveTPS {
		t.Fatalf("start at %d", p.TPS())
	}
	for range IdleAfterTicks + 1 {
		p.NoteIdle(false)
	}
	if p.TPS() != IdleTPS {
		t.Fatalf("idle rate %d", p.TPS())
	}
	p.NoteIdle(true)
	if p.TPS() != ActiveTPS {
		t.Fatalf("activity restores the rate: %d", p.TPS())
	}
}

func TestInputActive(t *testing.T) {
	var still widget.Input
	if InputActive(still, still) {
		t.Fatal("identical empty input is not activity")
	}
	moved := widget.Input{X: 3}
	if !InputActive(still, moved) {
		t.Fatal("pointer move is activity")
	}
	if !InputActive(still, widget.Input{Enter: true}) {
		t.Fatal("Enter is activity")
	}
}
