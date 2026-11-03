// Command workshop is a GPU-rendered desktop client for REST, WebSocket,
// MCP and other API calls.
package main

import (
	"errors"
	"flag"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui"
)

const (
	windowWidth  = 1280
	windowHeight = 800
)

func main() {
	var opts ui.Options
	flag.StringVar(&opts.ScreenshotPath, "screenshot", "", "save a PNG of the window to this path and exit")
	flag.BoolVar(&opts.SendOnStart, "send", false, "send the first request at startup (for screenshots)")
	flag.Parse()

	app, err := ui.NewApp(opts)
	if err != nil {
		log.Fatal(err)
	}
	ebiten.SetWindowSize(windowWidth, windowHeight)
	ebiten.SetWindowTitle("Gascoigne Workshop")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(app); err != nil && !errors.Is(err, ebiten.Termination) {
		log.Fatal(err)
	}
	if err := app.Err(); err != nil {
		log.Fatal(err)
	}
}
