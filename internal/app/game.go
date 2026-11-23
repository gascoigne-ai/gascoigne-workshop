package app

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// cursorShapes maps the cursor widgets ask for to the window system's.
var cursorShapes = map[core.Cursor]ebiten.CursorShapeType{
	core.CursorDefault:    ebiten.CursorShapeDefault,
	core.CursorResizeH:    ebiten.CursorShapeEWResize,
	core.CursorResizeV:    ebiten.CursorShapeNSResize,
	core.CursorMove:       ebiten.CursorShapeMove,
	core.CursorResizeNWSE: ebiten.CursorShapeNWSEResize,
	core.CursorResizeNESW: ebiten.CursorShapeNESWResize,
}

// Layout implements ebiten.Game. The UI is laid out in logical pixels
// (width by height); the screen is that size times the monitor's scale factor so
// text and lines stay sharp on HiDPI displays.
func (a *App) Layout(width, height int) (int, int) {
	reported := ebiten.Monitor().DeviceScaleFactor()
	f := frameSize(width, height, reported, a.displayScale(reported), a.uiScale())
	a.window.Resize(f.LW, f.LH, f.Scale, f.WindowScale)
	return f.SW, f.SH
}

// Update implements ebiten.Game: it samples input and runs one frame.
func (a *App) Update() error {
	a.window.SetPosition(ebiten.WindowPosition())
	a.window.Step(a.sampler.Read().Scaled(a.window.Scale()))
	if title := a.window.Title(); title != a.title {
		a.title = title
		ebiten.SetWindowTitle(title)
	}
	ebiten.SetCursorShape(cursorShapes[a.window.Cursor()])
	if tps := a.window.TPS(); tps != a.tps {
		ebiten.SetTPS(tps)
		a.tps = tps
	}
	return nil
}

// Draw implements ebiten.Game. It paints the tree of the last step, built
// again only when something changed after that tree was built.
func (a *App) Draw(screen *ebiten.Image) { a.window.Draw(screen) }
