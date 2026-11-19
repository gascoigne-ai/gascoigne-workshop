// Command workshop is a GPU-rendered desktop client for REST, WebSocket,
// MCP and other API calls.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/adapter/store"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/collection"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/adapter/proto"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/adapter/history"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/adapter/preview"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/adapter/displayscale"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/adapter/fonts"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/adapter/keyrepeat"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/adapter/dockicon"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/adapter/filedialog"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/adapter/folders"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/adapter/systemclock"
	"image"
	"image/png"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/adapter/settingsfile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/adapter/systeminfo"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/adapter/statefile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/ui/app"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/ui/appicon"
)

const (
	windowWidth  = 1280
	windowHeight = 800
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

// previewLifetime is how long a response stays served for the browser.
const previewLifetime = 5 * time.Minute

// launch is what the command line asks for: the app's options, the window
// size and the one-shot icon export.
type launch struct {
	opts             app.Options
	scene            *sceneFlags
	windowW, windowH int
	iconPNG          *string
}

func main() {
	run := defineFlags(flag.CommandLine)
	adapters(&run.opts)
	flag.Parse()
	run.opts.Scene = run.scene.steps()
	if *run.iconPNG != "" {
		if err := writeIcon(*run.iconPNG); err != nil {
			log.Fatal(err)
		}
		return
	}
	// A normal run opens the welcome dialog once per version; a screenshot
	// only when asked, so the other scenes show the window itself.
	if run.opts.ScreenshotPath == "" {
		run.opts.Welcome = true
		run.opts.WelcomeEvery = welcomeEveryStart
	}
	isolateScreenshot(&run.opts)

	window, err := app.NewApp(run.opts)
	if err != nil {
		log.Fatal(err)
	}
	if err := openWindow(window, run.windowW, run.windowH); err != nil {
		log.Fatal(err)
	}
	if err := window.FlushWorkspace(); err != nil {
		log.Printf("saving the collection: %v", err)
	}
	if err := window.SaveState(); err != nil {
		log.Printf("saving state: %v", err)
	}
	if err := window.Err(); err != nil {
		log.Fatal(err)
	}
}

// defineFlags declares the command line on set.
func defineFlags(set *flag.FlagSet) *launch {
	run := &launch{windowW: windowWidth, windowH: windowHeight}
	opts := &run.opts
	set.StringVar(&opts.ScreenshotPath, "screenshot", "", "save a PNG of the window to this path and exit")
	set.BoolVar(&opts.SendOnStart, "send", false, "send the first request at startup (for screenshots)")
	set.StringVar(&opts.SettingsPath, "settings", settingsfile.DefaultPath(), "settings file; empty keeps settings in memory, as a screenshot does unless this flag is given")
	set.StringVar(&opts.StatePath, "state", statefile.DefaultPath(), "remembered window and tabs; empty keeps them in memory")
	set.StringVar(&opts.ThemesDir, "themes", settingsfile.ThemesDir(), "folder of theme files (*.json) that extend the dark or light theme; empty for none")
	set.StringVar(&opts.WorkspaceDir, "workspace", "", "folder to load and save the collection; empty uses the last one, else the client config (release build) or the example server (default build)")
	run.iconPNG = set.String("icon-png", "", "write the program icon (256x256 PNG) to this path and exit, for packaging")
	set.BoolVar(&opts.Welcome, "welcome", false, "open the welcome dialog before the screenshot; a normal run opens it once per version")
	set.Func("window", "WIDTHxHEIGHT: open the window at this size, for screenshots", func(spec string) error {
		var err error
		run.windowW, run.windowH, err = parseWindowSize(spec)
		return err
	})
	set.BoolVar(&opts.OpenMenu, "menu", false, "start with the settings menu open (for screenshots)")
	run.scene = newSceneFlags(set)
	return run
}

// adapters gives opts the real implementation of every port.
func adapters(opts *app.Options) {
	opts.ScanFonts = true
	opts.Protocols = proto.Configured
	opts.GRPCMethods = proto.ListGRPCMethods
	opts.GRPCSchema = proto.GRPCInputSchema
	opts.Curl = proto.Curl
	opts.Store = store.Files{}
	opts.Sample = collection.Sample // which sample is a build decision, see clients_*.go
	opts.Clients = clientConfigs()
	opts.States = statefile.Files{}
	opts.Files = filedialog.System{}
	opts.Folders = folders.Disk{}
	opts.History = func(dir string) port.History { return history.Dir(dir) }
	home, err := os.UserHomeDir()
	if err != nil { // the user's own font folders are then left out of the scan
		log.Printf("fonts: no home folder, scanning the system folders only: %v", err)
	}
	opts.Fonts = fonts.System{Home: home}
	opts.Display = displayscale.New()
	opts.KeyRepeat = keyrepeat.New()
	opts.Previews = preview.Browser{Lifetime: previewLifetime}
	opts.Clock = systemclock.Clock{}
	opts.SettingsFiles = settingsfile.Files{}
	opts.System = systeminfo.Info{Graphics: systeminfo.Ebiten}
	opts.DefaultWorkspace = settingsfile.DefaultWorkspace()
}

// isolateScreenshot keeps a screenshot from the real state and settings:
// -set and -record save settings, and a screenshot taken by hand would
// overwrite the user's file, so only a path given on the command line is
// used. A screenshot is also the same on every platform, so it skips the
// platform's base scale.
func isolateScreenshot(opts *app.Options) {
	if opts.ScreenshotPath == "" {
		opts.BaseScale = settings.BaseScale(runtime.GOOS)
		return
	}
	if !flagSet("state") {
		opts.StatePath = ""
	}
	if !flagSet("settings") {
		opts.SettingsPath = ""
	}
}

// openWindow sizes the window, restores its last place and runs the game
// until the window closes.
func openWindow(window *app.App, width, height int) error {
	ebiten.SetWindowSize(width, height)
	if mw, mh := ebiten.Monitor().Size(); mw > 0 && mh > 0 {
		if restored := window.RestoredWindow(mw, mh); restored.W > 0 {
			ebiten.SetWindowSize(restored.W, restored.H)
			ebiten.SetWindowPosition(restored.X, restored.Y)
		}
	}
	ebiten.SetWindowIcon([]image.Image{appicon.Image(32), appicon.Image(64), appicon.Image(128)})
	var dock bytes.Buffer
	if png.Encode(&dock, appicon.Image(512)) == nil {
		dockicon.Set(dock.Bytes()) // the Dock ignores the window icon
	}
	ebiten.SetWindowTitle("Gascoigne Workshop")
	ebiten.SetWindowSizeLimits(720, 480, -1, -1) // below this the panes cannot show their controls
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetScreenClearedEveryFrame(false) // frames without changes are skipped in App.Draw
	if err := ebiten.RunGame(window); err != nil && !errors.Is(err, ebiten.Termination) {
		return err
	}
	return nil
}

// flagSet reports whether the flag was given on the command line.
func flagSet(name string) bool {
	set := false
	flag.Visit(func(visited *flag.Flag) { set = set || visited.Name == name })
	return set
}

// writeIcon saves the program icon as a PNG file.
func writeIcon(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, appicon.Image(256)); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
