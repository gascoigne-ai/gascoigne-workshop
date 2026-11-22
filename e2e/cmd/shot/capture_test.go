//go:build !release

package main

import (
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// fakeSender is the driver as the capture's timing sees it.
type fakeSender struct {
	sent       int
	unanswered bool // a send has no outcome yet
	settled    bool
}

func (fake *fakeSender) Send()              { fake.sent++ }
func (fake *fakeSender) ServeSends()        {}
func (fake *fakeSender) SendsPending() bool { return false }
func (fake *fakeSender) AllAnswered() bool  { return !fake.unanswered }
func (fake *fakeSender) Settled() bool      { return fake.settled }

// fakeStep records that it ran; fail makes it return an error.
type fakeStep struct {
	ran  *[]string
	name string
	fail error
}

func (step fakeStep) apply(*scene) error {
	*step.ran = append(*step.ran, step.name)
	return step.fail
}

// fakeDeferred is ready after late tries.
type fakeDeferred struct {
	applied int
	late    int
}

func (step *fakeDeferred) apply(*scene) error { step.applied++; return nil }
func (step *fakeDeferred) ready() bool        { return step.applied >= step.late }

func newTestCapture(sends *fakeSender, sendOnStart bool, steps ...Step) *capture {
	shot := &capture{sends: sends, steps: steps, scene: scene{sendOnStart: sendOnStart}}
	for _, step := range steps {
		if waits, ok := step.(deferred); ok {
			shot.pending = append(shot.pending, waits)
		}
	}
	return shot
}

func TestTheCaptureWaitsForTheSettleFramesThenTheRunEnds(t *testing.T) {
	shot := newTestCapture(&fakeSender{settled: true}, false)
	for frame := 1; frame < settleFrames; frame++ {
		if err := shot.update(); err != nil {
			t.Fatal(err)
		}
		if shot.ready() {
			t.Fatalf("frame %d: not settled yet", frame)
		}
	}
	shot.update()
	if !shot.ready() {
		t.Fatal("must be ready once the frames have settled")
	}

	shot.phase = captureOver // what save records after writing the file
	if err := shot.update(); !errors.Is(err, ebiten.Termination) {
		t.Fatalf("after the capture the next update must end the game, got %v", err)
	}
	if shot.ready() {
		t.Fatal("must not capture twice")
	}
}

func TestTheCaptureGivesUpWaitingAfterTheTimeout(t *testing.T) {
	cases := []struct {
		name   string
		sends  *fakeSender
		onSend bool
	}{
		{"a send that never answers", &fakeSender{settled: true, unanswered: true}, true},
		{"a window that stays unsettled", &fakeSender{settled: false}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			shot := newTestCapture(tc.sends, tc.onSend)
			shot.sent = true
			shot.frames = settleFrames
			if shot.ready() {
				t.Fatal("the capture must wait while the send is open or the window is loading")
			}
			shot.frames = shotTimeout
			if !shot.ready() {
				t.Fatal("after the timeout the capture must go ahead")
			}
		})
	}
}

func TestTheFirstRequestIsSentOnceWhenSendIsAsked(t *testing.T) {
	sends := &fakeSender{settled: true}
	shot := newTestCapture(sends, true)
	for range settleFrames {
		shot.update()
	}
	if sends.sent != 1 {
		t.Fatalf("sent %d times, want once", sends.sent)
	}
	quiet := &fakeSender{settled: true}
	idle := newTestCapture(quiet, false)
	for range settleFrames {
		idle.update()
	}
	if quiet.sent != 0 {
		t.Fatalf("sent %d times without the flag", quiet.sent)
	}
}

// Steps run in the order of the list, so a later step sees what an earlier one
// did, and the first failure stops the rest.
func TestStepsRunInOrderAndAFailureStopsTheRest(t *testing.T) {
	var ran []string
	shot := newTestCapture(&fakeSender{}, false,
		fakeStep{ran: &ran, name: "first"},
		fakeStep{ran: &ran, name: "bad", fail: errors.New("no such dialog")},
		fakeStep{ran: &ran, name: "never"})
	for range setFrame {
		shot.update()
	}
	if got := strings.Join(ran, ","); got != "first,bad" {
		t.Fatalf("ran %q, want the steps up to the failure", got)
	}
	if shot.err == nil || shot.err.Error() != "screenshot: no such dialog" {
		t.Fatalf("err = %v, want the failure with its message", shot.err)
	}
	if err := shot.update(); !errors.Is(err, ebiten.Termination) {
		t.Fatalf("a failed run must end at the next update, got %v", err)
	}
}

// A deferred step does not run at start: it waits for the sends, and the
// capture waits for it until it is ready.
func TestADeferredStepRunsOnceTheSendsAreAnsweredAndTheCaptureWaitsForIt(t *testing.T) {
	step := &fakeDeferred{late: 2}
	sends := &fakeSender{settled: true, unanswered: true}
	shot := newTestCapture(sends, true, step)
	for range settleFrames {
		shot.update()
	}
	if step.applied != 0 || shot.ready() {
		t.Fatalf("applied %d times, ready %v while a send is open", step.applied, shot.ready())
	}
	sends.unanswered = false
	shot.update()
	if step.applied != 1 || shot.ready() {
		t.Fatalf("applied %d times, ready %v, want one try and the capture still waiting", step.applied, shot.ready())
	}
	shot.update()
	if step.applied != 2 || !shot.ready() {
		t.Fatalf("applied %d times, ready %v, want it done and the capture ready", step.applied, shot.ready())
	}
	shot.update()
	if step.applied != 2 {
		t.Errorf("a ready step ran again: applied %d times", step.applied)
	}
}

func TestTheFrameIsWrittenAsAPNGAndTheRunEnds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.png")
	shot := newTestCapture(&fakeSender{settled: true}, false)
	shot.path = path
	shot.save(image.NewRGBA(image.Rect(0, 0, 3, 2)))
	if shot.err != nil || shot.phase != captureOver {
		t.Fatalf("err = %v, phase = %v, want the run over without an error", shot.err, shot.phase)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	img, err := png.Decode(file)
	if err != nil || img.Bounds().Dx() != 3 || img.Bounds().Dy() != 2 {
		t.Fatalf("decoded %v %v", img, err)
	}
}

func TestAFrameThatCannotBeWrittenEndsTheRunWithTheError(t *testing.T) {
	shot := newTestCapture(&fakeSender{settled: true}, false)
	shot.path = filepath.Join(t.TempDir(), "missing", "x.png")
	shot.save(image.NewRGBA(image.Rect(0, 0, 1, 1)))
	if shot.err == nil || !strings.Contains(shot.err.Error(), "write screenshot") || shot.phase != captureOver {
		t.Fatalf("err = %v, phase = %v, want the write error and the run over", shot.err, shot.phase)
	}
}
