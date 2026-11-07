package ui

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"strconv"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"

	"github.com/hajimehoshi/ebiten/v2"
)

// settleFrames lets fonts and layout warm up before a capture.
const (
	settleFrames = 10
	shotTimeout  = 1200 // frames to wait for a response before giving up
	setFrame     = 5    // the frame at which live settings are applied, early enough to be drawn before the capture
)

// screenshot saves one frame to a PNG and ends the program. It exists
// so the UI can be checked without a human at the window.
type screenshot struct {
	path        string
	sendOnStart bool
	frames      int
	sent        bool
	captured    bool
	err         error
	set         []string // key=value settings applied live before the capture, as the dialog would
}

// Err is the error of the capture, if any.
func (a *App) Err() error { return a.shot.err }

func (a *App) shotUpdate() error {
	s := &a.shot
	if s.path == "" {
		return nil
	}
	if s.captured {
		return ebiten.Termination
	}
	s.frames++
	if s.frames == setFrame {
		for _, spec := range s.set {
			a.applyShotSetting(spec)
		}
	}
	if s.sendOnStart && !s.sent {
		s.sent = true
		a.send(a.cur())
	}
	return nil
}

// ready reports whether the frame about to be drawn should be saved.
func (s *screenshot) ready(a *App) bool {
	if s.path == "" || s.captured || s.frames < settleFrames {
		return false
	}
	if s.sendOnStart && a.cur().resp == nil && s.frames < shotTimeout {
		return false
	}
	return true
}

func (a *App) shotDraw(screen *ebiten.Image) {
	s := &a.shot
	if !s.ready(a) {
		return
	}
	s.captured = true
	b := screen.Bounds()
	img := &image.RGBA{Pix: make([]byte, 4*b.Dx()*b.Dy()), Stride: 4 * b.Dx(), Rect: image.Rect(0, 0, b.Dx(), b.Dy())}
	screen.ReadPixels(img.Pix)
	f, err := os.Create(s.path)
	if err != nil {
		s.err = fmt.Errorf("create screenshot: %w", err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		s.err = fmt.Errorf("encode screenshot: %w", err)
	}
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

// applyShotSetting changes one setting through the path the settings dialog
// uses, so a screenshot shows a live change and not a start-up one.
func (a *App) applyShotSetting(spec string) {
	key, value, err := parseSetting(spec)
	if err != nil {
		a.shot.err = err
		return
	}
	a.Change(func(s *settings.Settings) { *s = settings.With(*s, key, value) })
}
