package app

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/tree"
	"image"
	"image/png"
	"slices"
	"strconv"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/serial"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/tabs"

	"github.com/hajimehoshi/ebiten/v2"
)

// settleFrames lets fonts and layout warm up before a capture.
const (
	settleFrames = 10
	shotTimeout  = 1200 // frames to wait for a response before giving up
	setFrame     = 5    // the frame at which live settings are applied, early enough to be drawn before the capture
)

// shotPhase is how far a capture has come. It only moves forward:
// waiting, saving, over; a capture that fails before its frame goes
// straight to over.
type shotPhase int

const (
	shotWaiting shotPhase = iota // the frame to capture is not drawn yet
	shotSaving                   // the frame is taken and being written
	shotOver                     // written or failed: the next update ends the run
)

// screenshot saves one frame to a PNG and ends the program. It exists
// so the UI can be checked without a human at the window.
type screenshot struct {
	path        string
	sendOnStart bool
	frames      int
	sent        bool
	waiting     []*reqTab // tabs whose send the capture waits for
	sendNext    *reqTab   // the tab OpenRequest opened, sent once the earlier sends are answered
	phase       shotPhase
	err         error
	steps       []Step     // the scene, run in order at setFrame
	pending     []deferred // the scene's steps that wait for the sends to be answered
}

// newScreenshot builds the capture of a run; a run without a path has none.
func newScreenshot(opts Options) screenshot {
	shot := screenshot{path: opts.ScreenshotPath, sendOnStart: opts.SendOnStart, steps: opts.Scene}
	for _, step := range opts.Scene {
		if waits, ok := step.(deferred); ok {
			shot.pending = append(shot.pending, waits)
		}
	}
	return shot
}

// Err is the error of the capture, if any.
func (a *App) Err() error { return a.shot.err }

func (a *App) shotUpdate() error {
	s := &a.shot
	if s.path == "" {
		return nil
	}
	switch s.phase {
	case shotOver:
		return ebiten.Termination
	case shotSaving:
		return nil // the frame is taken; nothing more is set up for it
	}
	s.frames++
	if s.frames == setFrame && !a.showShotScene() {
		return nil
	}
	if s.sendOnStart && !s.sent {
		s.sent = true
		a.shotSend(a.cur())
	}
	if s.sendNext != nil && s.allAnswered() { // in turn, so the sent log's order is fixed
		next := s.sendNext
		s.sendNext = nil
		a.shotSend(next)
	}
	if s.frames >= setFrame && s.sendsDone() { // after the response, or there is nothing to search
		a.runPending()
	}
	return nil
}

// showShotScene runs the scene's steps in order. The first step that fails
// ends the run with its error and the rest are not run: showing a scene that
// is not the one asked for would let a golden pass for the wrong picture. It
// reports whether the run goes on.
func (a *App) showShotScene() bool {
	for _, step := range a.shot.steps {
		if _, waits := step.(deferred); waits {
			continue
		}
		if err := step.apply(a); err != nil {
			a.failShot(err)
			return false
		}
	}
	return true
}

// runPending tries the deferred steps that are not ready yet.
func (a *App) runPending() {
	for _, step := range a.shot.pending {
		if step.ready(a) {
			continue
		}
		if err := step.apply(a); err != nil {
			a.failShot(err)
			return
		}
	}
}

// failShot ends the capture with err.
func (a *App) failShot(err error) {
	a.shot.err = err
	a.shot.phase = shotOver
}

// sendsDone reports whether every send the capture asked for has its outcome.
func (s *screenshot) sendsDone() bool {
	return !s.sendOnStart || (s.sent && s.sendNext == nil && s.allAnswered())
}

// shotSend sends tab's request and makes the capture wait for its outcome.
func (a *App) shotSend(tab *reqTab) {
	a.shot.waiting = append(a.shot.waiting, tab)
	a.send(tab)
}

// allAnswered reports whether every send of the capture has its outcome.
func (s *screenshot) allAnswered() bool {
	for _, t := range s.waiting {
		if !answered(t) {
			return false
		}
	}
	return true
}

// ready reports whether the frame about to be drawn should be saved.
func (s *screenshot) ready(app *App) bool {
	if s.path == "" || s.phase != shotWaiting || s.frames < settleFrames {
		return false
	}
	if !s.sendsDone() && s.frames < shotTimeout {
		return false
	}
	if !s.pendingReady(app) && s.frames < shotTimeout {
		return false
	}
	if (protoReading(app.cur()) || schemaLoading(app.cur())) && s.frames < shotTimeout {
		return false
	}
	return true
}

// pendingReady reports whether every deferred step of the scene has run.
func (s *screenshot) pendingReady(app *App) bool {
	return !slices.ContainsFunc(s.pending, func(step deferred) bool { return !step.ready(app) })
}

// answered reports whether what -send started has an outcome to show: a
// response, or for a socket an open connection (an MCP server also has
// said who it is) or a failed one.
func answered(tab *reqTab) bool {
	if !isSocket(tab) {
		return tab.Resp != nil && !tab.Sending() // a tab sent again still holds its earlier response
	}
	if tab.State() == tabs.ConnConnecting {
		return false
	}
	if tab.Session() == nil {
		return len(tab.wsView.Lines) > 0 // the error is in the log
	}
	explores := slices.ContainsFunc(viewOf(tab.Req.Protocol).streamPanes, func(panel pane) bool { return panel.kind == paneExplorer })
	return !explores || tab.Server != nil // an explorer waits for the server to say who it is
}

func (a *App) shotDraw(screen *ebiten.Image) {
	s := &a.shot
	if !s.ready(a) {
		return
	}
	b := screen.Bounds()
	img := &image.RGBA{Pix: make([]byte, 4*b.Dx()*b.Dy()), Stride: 4 * b.Dx(), Rect: image.Rect(0, 0, b.Dx(), b.Dy())}
	screen.ReadPixels(img.Pix)
	a.saveShot(img)
}

// saveShot encodes the captured frame as a PNG and writes it to the
// screenshot's path off the UI goroutine; the run ends once it is written.
func (a *App) saveShot(img *image.RGBA) {
	s := &a.shot
	s.phase = shotSaving
	files, path := a.folders, s.path
	a.write("screenshot", serial.Job{
		Run: func() error {
			var encoded bytes.Buffer
			if err := png.Encode(&encoded, img); err != nil {
				return fmt.Errorf("encode screenshot: %w", err)
			}
			if err := files.Write(context.Background(), path, encoded.Bytes()); err != nil {
				return fmt.Errorf("write screenshot: %w", err)
			}
			return nil
		},
		Done: func(err error) {
			s.err, s.phase = err, shotOver
		},
	})
}

// parseSetting reads "key=value": true and false are booleans, whole numbers
// are numbers, anything else is text.
func parseSetting(spec string) (key string, value any, err error) {
	key, text, ok := strings.Cut(spec, "=")
	if !ok || key == "" {
		return "", nil, fmt.Errorf("%q is not key=value", spec)
	}
	switch {
	case text == "true" || text == "false":
		return key, text == "true", nil
	}
	if number, convErr := strconv.Atoi(text); convErr == nil {
		return key, number, nil
	}
	return key, text, nil
}

func (a *App) findRequestByName(name string) *model.Request {
	var found *model.Request
	a.eachRoot(func(root *model.Folder) {
		tree.WalkRequests(root, "", func(_ string, req *model.Request) {
			if found == nil && req.Name == name {
				found = req
			}
		})
	})
	return found
}
