package core

import (
	"testing"
	"time"
)

func TestTheCaretBlinksOnAWallClockAndRestartsOnActivity(t *testing.T) {
	ctx, err := NewCtx(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	ctx.Now = func() time.Time { return now }
	ctx.RestartCaret()
	steps := []struct {
		after time.Duration
		want  bool
	}{
		{0, true}, {100 * time.Millisecond, true}, {499 * time.Millisecond, true},
		{500 * time.Millisecond, false}, {900 * time.Millisecond, false},
		{1000 * time.Millisecond, true}, {1500 * time.Millisecond, false},
	}
	start := now
	for _, step := range steps {
		now = start.Add(step.after)
		if got := ctx.CaretVisible(); got != step.want {
			t.Errorf("after %v: visible = %v, want %v", step.after, got, step.want)
		}
	}
	now = start.Add(700 * time.Millisecond) // in an off phase
	ctx.RestartCaret()
	if !ctx.CaretVisible() {
		t.Fatal("activity must show the caret again")
	}
}

func TestTheBlinkRateDoesNotDependOnTheUpdateRate(t *testing.T) {
	// the phase is a function of time only: the same instant gives the same
	// answer however many ticks have passed
	ctx, _ := NewCtx(testTheme())
	start := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	ctx.Now = func() time.Time { return start.Add(600 * time.Millisecond) }
	ctx.CaretSince = start
	ctx.Tick = 7
	first := ctx.CaretVisible()
	ctx.Tick = 7000
	if ctx.CaretVisible() != first {
		t.Fatal("the caret phase changed with the tick count")
	}
}
