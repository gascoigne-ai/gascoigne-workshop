package ui

import (
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

// settleFrames lets fonts and layout warm up before a capture.
const (
	settleFrames = 10
	shotTimeout  = 1200 // frames to wait for a response before giving up
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
