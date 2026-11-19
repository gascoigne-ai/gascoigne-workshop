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

func main() {
	var opts app.Options
	flag.StringVar(&opts.ScreenshotPath, "screenshot", "", "save a PNG of the window to this path and exit")
	flag.BoolVar(&opts.SendOnStart, "send", false, "send the first request at startup (for screenshots)")
	flag.StringVar(&opts.SettingsPath, "settings", settingsfile.DefaultPath(), "settings file; empty keeps settings in memory, as a screenshot does unless this flag is given")
	flag.StringVar(&opts.StatePath, "state", statefile.DefaultPath(), "remembered window and tabs; empty keeps them in memory")
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
	flag.StringVar(&opts.ThemesDir, "themes", settingsfile.ThemesDir(), "folder of theme files (*.json) that extend the dark or light theme; empty for none")
	flag.StringVar(&opts.WorkspaceDir, "workspace", "", "folder to load and save the collection; empty uses the last one, else the client config (release build) or the example server (default build)")
	opts.DefaultWorkspace = settingsfile.DefaultWorkspace()
	iconPNG := flag.String("icon-png", "", "write the program icon (256x256 PNG) to this path and exit, for packaging")
	flag.BoolVar(&opts.Welcome, "welcome", false, "open the welcome dialog before the screenshot; a normal run opens it once per version")
	windowW, windowH := windowWidth, windowHeight
	flag.Func("window", "WIDTHxHEIGHT: open the window at this size, for screenshots", func(spec string) error {
		var err error
		windowW, windowH, err = parseWindowSize(spec)
		return err
	})
	flag.BoolVar(&opts.OpenMenu, "menu", false, "start with the settings menu open (for screenshots)")
	scene := newSceneFlags(flag.CommandLine)
	flag.Parse()
	opts.Scene = scene.steps()
	if *iconPNG != "" {
		if err := writeIcon(*iconPNG); err != nil {
			log.Fatal(err)
		}
		return
	}
	// A normal run opens the welcome dialog once per version; a screenshot
	// only when asked, so the other scenes show the window itself.
	if opts.ScreenshotPath == "" {
		opts.Welcome = true
		opts.WelcomeEvery = welcomeEveryStart
	}
	// Screenshots must not read or write the real state or settings: -set and
	// -record save settings, and a screenshot taken by hand would overwrite
	// the user's file. Only a path given on the command line is used.
	if opts.ScreenshotPath != "" && !flagSet("state") {
		opts.StatePath = ""
	}
	if opts.ScreenshotPath != "" && !flagSet("settings") {
		opts.SettingsPath = ""
	}

	// A screenshot is the same on every platform, so it skips the platform's
	// base scale.
	if opts.ScreenshotPath == "" {
		opts.BaseScale = settings.BaseScale(runtime.GOOS)
	}

	window, err := app.NewApp(opts)
	if err != nil {
		log.Fatal(err)
	}
	ebiten.SetWindowSize(windowW, windowH)
	if mw, mh := ebiten.Monitor().Size(); mw > 0 && mh > 0 {
		if w := window.RestoredWindow(mw, mh); w.W > 0 {
			ebiten.SetWindowSize(w.W, w.H)
			ebiten.SetWindowPosition(w.X, w.Y)
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

// flagSet reports whether the flag was given on the command line.
func flagSet(name string) bool {
	set := false
	flag.Visit(func(flag *flag.Flag) { set = set || flag.Name == name })
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
