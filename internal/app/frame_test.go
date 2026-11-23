package app

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
)

func TestFrameSizeAppliesTheInterfaceScaleOnTopOfTheDisplayScale(t *testing.T) {
	cases := []struct {
		name         string
		w, h         int
		dsf, ui      float64
		lw, lh       int
		scale        float64
		screenW, scH int
	}{
		{"no scaling", 1000, 800, 1, 1, 1000, 800, 1, 1000, 800},
		{"ten percent bigger", 1100, 880, 1, 1.1, 1000, 800, 1.1, 1100, 880},
		{"hidpi and bigger", 1000, 600, 2, 1.25, 800, 480, 2.5, 2000, 1200},
	}
	for _, c := range cases {
		got := frameSize(c.w, c.h, c.dsf, c.dsf, c.ui)
		if got.LW != c.lw || got.LH != c.lh || got.Scale != c.scale || got.SW != c.screenW || got.SH != c.scH {
			t.Errorf("%s: got %+v", c.name, got)
		}
	}
}

// A compositor that scales the display itself (Hyprland with Xwayland)
// leaves Ebitengine at 1: the window is in real pixels, so the logical
// size shrinks by the true scale and the screen stays the window size.
func TestFrameSizeUsesAScaleTheWindowSystemDoesNotReport(t *testing.T) {
	got := frameSize(2000, 1200, 1, 2, 1.25)
	want := frame{LW: 800, LH: 480, Scale: 2.5, SW: 2000, SH: 1200, WindowScale: 2.5}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestDisplayScaleTrustsTheWindowSystemUnlessItReportsOne(t *testing.T) {
	cases := []struct {
		name     string
		reported float64
		display  DisplayScale
		want     float64
	}{
		{"reported wins when above one", 2, fakeDisplay(1.5), 2},
		{"desktop fills in a reported one", 1, fakeDisplay(2), 2},
		{"fractional desktop scale", 1, fakeDisplay(1.25), 1.25},
		{"unknown desktop scale", 1, fakeDisplay(0), 1},
		{"no desktop source", 1, nil, 1},
	}
	for _, c := range cases {
		app := &App{display: c.display}
		if got := app.displayScale(c.reported); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

type fakeDisplay float64

func (display fakeDisplay) Factor() float64 { return float64(display) }

// The platform's base factor multiplies the user's scale, so the slider can
// read 100% while the interface is as big as the platform needs.
func TestUIScaleIncludesTheBaseScale(t *testing.T) {
	plain, err := NewApp(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := plain.uiScale(); got != 1 {
		t.Fatalf("no base: %v, want 1", got)
	}
	scaled, err := NewApp(Options{
		BaseScale: 1.1, SettingsPath: "settings.json",
		SettingsFiles: fixedSettings{current: settings.Settings{UIScalePct: 150}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := scaled.uiScale(); got < 1.649 || got > 1.651 {
		t.Fatalf("base 1.1 and 150%%: %v, want 1.65", got)
	}
}

// fixedSettings is a settings store that reads one value.
type fixedSettings struct {
	settings.SettingsStore
	current settings.Settings
}

func (store fixedSettings) Stat(string) settings.FileStamp { return settings.FileStamp{} }

func (store fixedSettings) Load(string) (settings.Settings, error) { return store.current, nil }
