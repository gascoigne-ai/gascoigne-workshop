package app

import (
	"testing"
	"time"
)

type fixedClock time.Time

func (clock fixedClock) Now() time.Time { return time.Time(clock) }

func TestAppReadsTimeFromTheClockPort(t *testing.T) {
	want := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	app, err := NewApp(Options{Clock: fixedClock(want)})
	if err != nil {
		t.Fatal(err)
	}
	if got := app.window.Now(); !got.Equal(want) {
		t.Fatalf("now = %v, want %v", got, want)
	}
}
