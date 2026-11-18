package app

import (
	"path/filepath"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

func TestAStepThatCannotRunEndsTheCaptureWithItsMessage(t *testing.T) {
	cases := []struct {
		name string
		step Step
		want string
	}{
		{"unknown request", OpenRequest("Nope"), `screenshot request "Nope" is not in the collection`},
		{"unknown pane", ShowPane("Nope"), `screenshot pane "Nope" is not a pane of the open request`},
		{"unknown response pane", ShowResponsePane("Nope"), `screenshot response pane "Nope" is not a response pane of the open request`},
		{"unknown body type", ChooseBody("Nope"), `screenshot body type "Nope" is not one of the body chips`},
		{"unknown shortcut action", Record("nope"), `screenshot record: "nope" is not a shortcut action`},
		{"unknown dialog", OpenDialog("nope"), `screenshot dialog "nope" is not one of rename, delete, move, copy`},
		{"setting without a value", Set("nonsense"), `"nonsense" is not key=value`},
		{"folder without a collection", AddCollection(""), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := shotApp(t, withAdapters(Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), Scene: []Step{tc.step}}))
			for range setFrame {
				a.shotUpdate()
			}
			if a.Err() == nil || a.shot.phase != shotOver {
				t.Fatalf("err = %v, phase = %v, want the run ended with an error", a.Err(), a.shot.phase)
			}
			if tc.want != "" && a.Err().Error() != tc.want {
				t.Errorf("err = %q, want %q", a.Err(), tc.want)
			}
		})
	}
}

func TestTypeBodyNeedsATextBody(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), Scene: []Step{ChooseBody(string(model.BodyFile)), TypeBody("x")}})
	for range setFrame {
		a.shotUpdate()
	}
	want := `screenshot body text needs a JSON or Text body, the open request's is "file"`
	if a.Err() == nil || a.Err().Error() != want {
		t.Fatalf("err = %v, want %q", a.Err(), want)
	}
}

// Steps run in the order of the list, so a later step sees what an earlier one
// did, and the first failure stops the rest.
func TestStepsRunInOrderAndAFailureStopsTheRest(t *testing.T) {
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), Scene: []Step{ShowToast("first"), OpenDialog("nope"), ShowToast("never")}})
	for range setFrame {
		a.shotUpdate()
	}
	if a.Err() == nil {
		t.Fatal("the bad dialog did not fail the run")
	}
	if len(a.toasts) != 1 || a.toasts[0].Text != "first" {
		t.Fatalf("toasts %+v, want only the one before the failure", a.toasts)
	}
}

// A deferred step does not run at start: it waits for the sends, and the
// capture waits for it until it is ready.
func TestADeferredStepRunsOnceTheSendsAreAnsweredAndTheCaptureWaitsForIt(t *testing.T) {
	step := &fakeDeferred{late: 2}
	a := shotApp(t, Options{ScreenshotPath: filepath.Join(t.TempDir(), "x.png"), SendOnStart: true, Scene: []Step{step}})
	a.shot.sent = true
	a.shot.waiting = []*reqTab{a.cur()} // a send that is not answered yet
	for range settleFrames {
		a.shotUpdate()
	}
	if step.applied != 0 || a.shot.ready(a) {
		t.Fatalf("applied %d times, ready %v while a send is open", step.applied, a.shot.ready(a))
	}
	a.shot.waiting = nil // the send is answered
	a.shotUpdate()
	if step.applied != 1 || a.shot.ready(a) {
		t.Fatalf("applied %d times, ready %v, want one try and the capture still waiting", step.applied, a.shot.ready(a))
	}
	a.shotUpdate()
	if step.applied != 2 || !a.shot.ready(a) {
		t.Fatalf("applied %d times, ready %v, want it done and the capture ready", step.applied, a.shot.ready(a))
	}
	a.shotUpdate()
	if step.applied != 2 {
		t.Errorf("a ready step ran again: applied %d times", step.applied)
	}
}

type fakeDeferred struct {
	applied int
	late    int // tries before it is ready
}

func (step *fakeDeferred) apply(*App) error { step.applied++; return nil }
func (step *fakeDeferred) ready(*App) bool  { return step.applied >= step.late }
