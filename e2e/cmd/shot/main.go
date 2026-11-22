//go:build !release

// Command shot starts the workshop, puts it into the state its flags
// describe, saves one frame as a PNG and exits. The end-to-end tests draw
// their golden scenes with it, and `make shot` renders the window with it.
// It wires the app like cmd/workshop, with in-memory settings and state
// unless a flag names a file, and runs the app inside a capture.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/app"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/adapter/store"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history"
	historyadapter "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history/adapter"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/adapter/preview"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/proto"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/adapter/settingsfile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/adapter/folders"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/adapter/statefile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/platform/fonts"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/platform/systeminfo"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/displayscale"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/keyrepeat"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	windowWidth  = 1280
	windowHeight = 800
	// previewLifetime is how long a response stays served for the browser.
	previewLifetime = 5 * time.Minute
)

// parseWindowSize reads "WIDTHxHEIGHT" for the -window flag.
func parseWindowSize(spec string) (width, height int, err error) {
	widthText, heightText, ok := strings.Cut(spec, "x")
	if !ok {
		return 0, 0, fmt.Errorf("%q is not WIDTHxHEIGHT", spec)
	}
	width, widthErr := strconv.Atoi(widthText)
	height, heightErr := strconv.Atoi(heightText)
	if widthErr != nil || heightErr != nil || width <= 0 || height <= 0 {
		return 0, 0, fmt.Errorf("%q is not a positive WIDTHxHEIGHT", spec)
	}
	return width, height, nil
}

// launch is what the command line asks for.
type launch struct {
	opts             app.Options
	scene            *sceneFlags
	path             string
	sendOnStart      bool
	openMenu         bool
	windowW, windowH int
}

func main() {
	run := defineFlags(flag.CommandLine)
	adapters(&run.opts)
	flag.Parse()
	if run.path == "" {
		log.Fatal("-screenshot PATH is required")
	}
	run.opts.DrawEveryFrame = true
	window, err := app.NewApp(run.opts)
	if err != nil {
		log.Fatal(err)
	}
	if run.openMenu {
		window.Driver().OpenSettingsMenu()
	}
	shot := newCapture(window, run.path, run.sendOnStart, run.scene.steps())
	if err := openWindow(shot, run.windowW, run.windowH); err != nil {
		log.Fatal(err)
	}
	if err := window.FlushWorkspace(); err != nil {
		log.Printf("saving the collection: %v", err)
	}
	if err := window.SaveState(); err != nil {
		log.Printf("saving state: %v", err)
	}
	for _, err := range []error{shot.err, window.Err()} {
		if err != nil {
			log.Fatal(err)
		}
	}
}

// defineFlags declares the command line on set.
func defineFlags(set *flag.FlagSet) *launch {
	run := &launch{windowW: windowWidth, windowH: windowHeight}
	opts := &run.opts
	set.StringVar(&run.path, "screenshot", "", "save a PNG of the window to this path and exit")
	set.BoolVar(&run.sendOnStart, "send", false, "send the first request at startup")
	set.StringVar(&opts.SettingsPath, "settings", "", "settings file; empty keeps settings in memory")
	set.StringVar(&opts.StatePath, "state", "", "remembered window and tabs; empty keeps them in memory")
	set.StringVar(&opts.ThemesDir, "themes", settingsfile.ThemesDir(), "folder of theme files (*.json) that extend the dark or light theme; empty for none")
	set.StringVar(&opts.WorkspaceDir, "workspace", "", "folder to load and save the collection; empty uses the last one, else the example server's")
	set.BoolVar(&opts.Welcome, "welcome", false, "open the welcome dialog before the screenshot")
	set.Func("window", "WIDTHxHEIGHT: open the window at this size", func(spec string) error {
		var err error
		run.windowW, run.windowH, err = parseWindowSize(spec)
		return err
	})
	set.BoolVar(&run.openMenu, "menu", false, "start with the settings menu open")
	run.scene = newSceneFlags(set)
	return run
}

// adapters gives opts the real implementation of every port, as the
// program's composition root does. The client config port stays nil: the
// default build starts from the sample, which carries the example server's
// endpoints.
func adapters(opts *app.Options) {
	opts.ScanFonts = true
	opts.Protocols = proto.Configured
	opts.GRPCMethods = proto.ListGRPCMethods
	opts.GRPCSchema = proto.GRPCInputSchema
	opts.Curl = proto.Curl
	opts.Store = store.Files{}
	opts.Sample = collection.Sample
	opts.States = statefile.Files{}
	opts.Files = filedialog.System{}
	opts.Folders = folders.Disk{}
	opts.History = func(dir string) history.History { return historyadapter.Dir(dir) }
	home, err := os.UserHomeDir()
	if err != nil { // the user's own font folders are then left out of the scan
		log.Printf("fonts: no home folder, scanning the system folders only: %v", err)
	}
	opts.Fonts = fonts.System{Home: home}
	opts.Display = displayscale.New()
	opts.KeyRepeat = keyrepeat.New()
	opts.Previews = preview.Browser{Lifetime: previewLifetime}
	opts.SettingsFiles = settingsfile.Files{}
	opts.System = systeminfo.Info{Graphics: systeminfo.Ebiten}
	opts.DefaultWorkspace = settingsfile.DefaultWorkspace()
}

// openWindow sizes the window and runs the game until it ends.
func openWindow(game ebiten.Game, width, height int) error {
	ebiten.SetWindowSize(width, height)
	ebiten.SetWindowTitle("Gascoigne Workshop")
	ebiten.SetWindowSizeLimits(720, 480, -1, -1) // below this the panes cannot show their controls
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetScreenClearedEveryFrame(false) // frames without changes are skipped in App.Draw
	if err := ebiten.RunGame(game); err != nil && !errors.Is(err, ebiten.Termination) {
		return err
	}
	return nil
}
