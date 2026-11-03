// Command workshop is a GPU-rendered desktop client for REST, WebSocket,
// MCP and other API calls.
package main

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	windowWidth  = 1280
	windowHeight = 800
)

var background = color.RGBA{R: 0x1b, G: 0x1b, B: 0x1b, A: 0xff}

type app struct{}

func (app) Update() error { return nil }

func (app) Draw(screen *ebiten.Image) { screen.Fill(background) }

func (app) Layout(w, h int) (int, int) { return w, h }

func main() {
	ebiten.SetWindowSize(windowWidth, windowHeight)
	ebiten.SetWindowTitle("Gascoigne Workshop")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(app{}); err != nil {
		log.Fatal(err)
	}
}
