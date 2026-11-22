//go:build !release

package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"slices"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/ui/app"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	settleFrames = 10   // lets fonts and layout warm up before the capture
	shotTimeout  = 1200 // frames to wait for a response before giving up
	setFrame     = 5    // the frame at which the scene is applied, early enough to be drawn before the capture
)

// capturePhase is how far a capture has come. It only moves forward:
// a run that failed or wrote its file ends at the next update.
type capturePhase int

const (
	captureWaiting capturePhase = iota // the frame to capture is not drawn yet
	captureOver                        // written or failed: the next update ends the run
)

// sender is the part of the app's driver the capture's timing depends on: the
// sends it waits for and whether the window shows anything still loading.
type sender interface {
	Send()
	ServeSends()
	SendsPending() bool
	AllAnswered() bool
	Settled() bool
}

// capture is an ebiten.Game around the app that runs a scene on it, waits
// until the window has settled and saves one frame as a PNG. It lives here,
// not in the app, so the shipped program carries no screenshot code.
type capture struct {
	window  *app.App
	sends   sender
	scene   scene
	path    string
	frames  int
	sent    bool
	phase   capturePhase
	err     error
	steps   []Step
	pending []deferred // the scene's steps that wait for the sends to be answered
}

// newCapture builds the capture of the scene steps on window, saved to path.
func newCapture(window *app.App, path string, sendOnStart bool, steps []Step) *capture {
	shot := &capture{
		window: window, sends: window.Driver(), path: path, steps: steps,
		scene: scene{driver: window.Driver(), sendOnStart: sendOnStart},
	}
	for _, step := range steps {
		if waits, ok := step.(deferred); ok {
			shot.pending = append(shot.pending, waits)
		}
	}
	return shot
}

// Layout implements ebiten.Game.
func (shot *capture) Layout(width, height int) (int, int) { return shot.window.Layout(width, height) }

// Update implements ebiten.Game: it runs the app's frame, then moves the
// capture along.
func (shot *capture) Update() error {
	if err := shot.window.Update(); err != nil {
		return err
	}
	return shot.update()
}

// update counts the frame, applies the scene at setFrame and starts the sends
// the capture waits for.
func (shot *capture) update() error {
	if shot.phase == captureOver {
		return ebiten.Termination
	}
	shot.frames++
	if shot.frames == setFrame && !shot.applyScene() {
		return nil
	}
	if shot.scene.sendOnStart && !shot.sent {
		shot.sent = true
		shot.sends.Send()
	}
	shot.sends.ServeSends()                          // in turn, so the sent log's order is fixed
	if shot.frames >= setFrame && shot.sendsDone() { // after the response, or there is nothing to search
		shot.runPending()
	}
	return nil
}

// applyScene runs the scene's steps in order. The first step that fails ends
// the capture and the later ones do not run.
func (shot *capture) applyScene() bool {
	for _, step := range shot.steps {
		if _, waits := step.(deferred); waits {
			continue
		}
		if err := step.apply(&shot.scene); err != nil {
			shot.fail(err)
			return false
		}
	}
	return true
}

// runPending tries the deferred steps that are not ready yet.
func (shot *capture) runPending() {
	for _, step := range shot.pending {
		if step.ready() {
			continue
		}
		if err := step.apply(&shot.scene); err != nil {
			shot.fail(err)
			return
		}
	}
}

// fail ends the capture with err.
func (shot *capture) fail(err error) {
	shot.err = fmt.Errorf("screenshot: %w", err)
	shot.phase = captureOver
}

// sendsDone reports whether every send the capture asked for has its outcome.
func (shot *capture) sendsDone() bool {
	return !shot.scene.sendOnStart || (shot.sent && !shot.sends.SendsPending() && shot.sends.AllAnswered())
}

// pendingReady reports whether every deferred step of the scene has run.
func (shot *capture) pendingReady() bool {
	return !slices.ContainsFunc(shot.pending, func(step deferred) bool { return !step.ready() })
}

// ready reports whether the frame about to be drawn should be saved.
func (shot *capture) ready() bool {
	if shot.phase != captureWaiting || shot.frames < settleFrames {
		return false
	}
	if !shot.sendsDone() && shot.frames < shotTimeout {
		return false
	}
	if !shot.pendingReady() && shot.frames < shotTimeout {
		return false
	}
	return shot.sends.Settled() || shot.frames >= shotTimeout
}

// Draw implements ebiten.Game: it draws the app, then saves the frame when
// the capture is ready.
func (shot *capture) Draw(screen *ebiten.Image) {
	shot.window.Draw(screen)
	if !shot.ready() {
		return
	}
	bounds := screen.Bounds()
	frame := &image.RGBA{Pix: make([]byte, 4*bounds.Dx()*bounds.Dy()), Stride: 4 * bounds.Dx(), Rect: image.Rect(0, 0, bounds.Dx(), bounds.Dy())}
	screen.ReadPixels(frame.Pix)
	shot.save(frame)
}

// save writes the frame as a PNG to the capture's path.
func (shot *capture) save(frame *image.RGBA) {
	shot.phase = captureOver
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, frame); err != nil {
		shot.err = fmt.Errorf("encode screenshot: %w", err)
		return
	}
	if err := os.WriteFile(shot.path, encoded.Bytes(), 0o644); err != nil {
		shot.err = fmt.Errorf("write screenshot: %w", err)
	}
}
