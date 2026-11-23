package app

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// fakeKeyRepeat is a desktop with a fixed key repeat setting.
type fakeKeyRepeat struct{ delay, interval time.Duration }

func (fake fakeKeyRepeat) Timing() (time.Duration, time.Duration) { return fake.delay, fake.interval }

func TestTheWindowRepeatsKeysAtTheDesktopsSpeed(t *testing.T) {
	want := core.KeyRepeat{Delay: 600 * time.Millisecond, Interval: 25 * time.Millisecond}
	app, err := NewApp(Options{
		SettingsPath: filepath.Join(t.TempDir(), "settings.json"),
		KeyRepeat:    fakeKeyRepeat{want.Delay, want.Interval},
	})
	if err != nil {
		t.Fatal(err)
	}
	if app.sampler.Repeat != want {
		t.Fatalf("key repeat = %+v, want %+v", app.sampler.Repeat, want)
	}
}

// The app's one job is to hand the program's options to the window; what the
// window shows of them tells that none was dropped on the way.
func TestNewAppHandsTheOptionsToTheWindow(t *testing.T) {
	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	app, err := NewApp(Options{SettingsPath: settingsPath, HostOS: "windows"})
	if err != nil {
		t.Fatal(err)
	}
	if got := app.SettingsPath(); got != settingsPath {
		t.Errorf("settings path = %q, want %q", got, settingsPath)
	}
	if app.sampler.HostOS != "windows" {
		t.Errorf("host OS of the sampler = %q, want windows", app.sampler.HostOS)
	}
}
