package ui

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func shotApp(t *testing.T, opts Options) *App {
	t.Helper()
	opts.SettingsPath = filepath.Join(t.TempDir(), "settings.json")
	a, err := NewApp(opts)
	if err != nil {
		t.Fatal(err)
	}
	a.w, a.h = winW, winH
	return a
}

func TestScreenshotIsOffByDefault(t *testing.T) {
	a := shotApp(t, Options{})
	for range 50 {
		if err := a.shotUpdate(); err != nil {
			t.Fatalf("a normal run must never terminate itself: %v", err)
		}
	}
	if a.shot.ready(a) {
		t.Fatal("no screenshot path, nothing to capture")
	}
	if a.Err() != nil {
		t.Fatalf("Err = %v", a.Err())
	}
}

func TestScreenshotWaitsForTheSettleFramesThenTerminates(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png")})

	for i := 1; i < settleFrames; i++ {
		if err := a.shotUpdate(); err != nil {
			t.Fatal(err)
		}
		if a.shot.ready(a) {
			t.Fatalf("frame %d: not settled yet", i)
		}
	}
	a.shotUpdate()
	if !a.shot.ready(a) {
		t.Fatal("must be ready once the frames have settled")
	}

	a.shot.captured = true // what shotDraw records after writing the file
	if err := a.shotUpdate(); !errors.Is(err, ebiten.Termination) {
		t.Fatalf("after the capture the next update must end the game, got %v", err)
	}
	if a.shot.ready(a) {
		t.Fatal("must not capture twice")
	}
}

func TestScreenshotWithSendWaitsForTheResponse(t *testing.T) {
	srv := jsonServer(t)
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), SendOnStart: true})
	tab := a.cur()
	pointTabAt(tab, srv.URL)

	for range settleFrames + 2 {
		a.shotUpdate()
	}
	if a.shot.ready(a) && tab.resp == nil {
		t.Fatal("with -send the capture must wait for the response")
	}
	if !tab.sending && tab.resp == nil {
		t.Fatal("-send must start the request on the first frame")
	}

	waitForResponse(t, a, tab)
	if !a.shot.ready(a) {
		t.Fatal("ready once the response is in")
	}
}

func TestScreenshotGivesUpWaitingForAResponse(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), SendOnStart: true})
	a.shot.sent = true // pretend the request was sent and never answers
	a.shot.frames = shotTimeout
	if !a.shot.ready(a) {
		t.Fatal("after the timeout the capture must go ahead without a response")
	}
}
