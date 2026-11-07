package engine

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestAFrameIsDrawnWhenSomethingChangedAndAtLeastMinFPSTimesASecond(t *testing.T) {
	cases := []struct {
		name           string
		tick, tps      int
		active, always bool
		want           bool
	}{
		{"input at any tick", 1, ActiveTPS, true, false, true},
		{"always (screenshots)", 1, ActiveTPS, false, true, true},
		{"idle, off the floor tick at 60 TPS", 5, 60, false, false, false},
		{"idle, on the floor tick at 60 TPS (every 6th is 10 FPS)", 6, 60, false, false, true},
		{"idle, off the floor tick at 30 TPS", 2, 30, false, false, false},
		{"idle, on the floor tick at 30 TPS (every 3rd is 10 FPS)", 3, 30, false, false, true},
		{"a rate below the floor draws every frame", 1, 5, false, false, true},
	}
	for _, c := range cases {
		if got := MustDraw(c.tick, c.tps, c.active, c.always); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestIdleFramesNeverComeLessOftenThanMinFPS(t *testing.T) {
	for _, tps := range []int{ActiveTPS, IdleTPS} {
		drawn := 0
		for tick := 1; tick <= tps; tick++ { // one second of idle frames
			if MustDraw(tick, tps, false, false) {
				drawn++
			}
		}
		if drawn < MinFPS {
			t.Errorf("%d TPS: %d frames drawn in a second, want at least %d", tps, drawn, MinFPS)
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

func TestInputIsActivityOnlyWhenItDiffersFromTheLastFrame(t *testing.T) {
	var still core.Input
	if InputActive(still, still) {
		t.Fatal("identical empty input is not activity")
	}
	moved := core.Input{X: 3}
	if !InputActive(still, moved) {
		t.Fatal("pointer move is activity")
	}
	if !InputActive(still, core.Input{Enter: true}) {
		t.Fatal("Enter is activity")
	}
}
