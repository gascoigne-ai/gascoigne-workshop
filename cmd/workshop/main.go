// Command workshop is a GPU-rendered desktop client for REST, WebSocket,
// MCP and other API calls.
package main

import (
	"errors"
	"flag"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/adapter/store"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/adapter/history"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/adapter/preview"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/adapter/fonts"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/adapter/filedialog"
	"image"
	"image/png"
	"log"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/adapter/statefile"
	ui "github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/ui/app"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/ui/appicon"
)

const (
	windowWidth  = 1280
	windowHeight = 800
)

// previewLifetime is how long a response stays served for the browser.
const previewLifetime = 5 * time.Minute

func main() {
	var opts ui.Options
	flag.StringVar(&opts.ScreenshotPath, "screenshot", "", "save a PNG of the window to this path and exit")
	flag.Func("set", "key=value: change a setting live before the screenshot, as the settings dialog would (repeatable, for screenshots)", func(spec string) error {
		opts.ScreenshotSet = append(opts.ScreenshotSet, spec)
		return nil
	})
	flag.BoolVar(&opts.SendOnStart, "send", false, "send the first request at startup (for screenshots)")
	flag.StringVar(&opts.SettingsPath, "settings", settings.DefaultPath(), "settings file; empty keeps settings in memory")
	flag.StringVar(&opts.StatePath, "state", statefile.DefaultPath(), "remembered window and tabs; empty keeps them in memory")
	opts.ScanFonts = true
	opts.Store = store.Files{}
	opts.States = statefile.Files{}
	opts.Files = filedialog.System{}
	opts.History = func(dir string) port.History { return history.Dir(dir) }
	home, _ := os.UserHomeDir()
	opts.Fonts = fonts.System{Home: home}
	opts.Previews = preview.Browser{Lifetime: previewLifetime}
	flag.StringVar(&opts.ThemesDir, "themes", settings.ThemesDir(), "folder of theme files (*.json) that extend the dark or light theme; empty for none")
	flag.StringVar(&opts.WorkspaceDir, "workspace", "", "folder to load and save the collection; empty uses the last one, else the built-in sample")
	opts.DefaultWorkspace = settings.DefaultWorkspace()
	iconPNG := flag.String("icon-png", "", "write the program icon (256x256 PNG) to this path and exit, for packaging")
	flag.BoolVar(&opts.OpenMenu, "menu", false, "start with the settings menu open (for screenshots)")
	flag.Parse()
	if *iconPNG != "" {
		if err := writeIcon(*iconPNG); err != nil {
			log.Fatal(err)
		}
		return
	}
	if opts.ScreenshotPath != "" && !flagSet("state") {
		opts.StatePath = "" // screenshots must not read or write the real state
	}

	app, err := ui.NewApp(opts)
	if err != nil {
		log.Fatal(err)
	}
	ebiten.SetWindowSize(windowWidth, windowHeight)
	if mw, mh := ebiten.Monitor().Size(); mw > 0 && mh > 0 {
		if w := app.RestoredWindow(mw, mh); w.W > 0 {
			ebiten.SetWindowSize(w.W, w.H)
			ebiten.SetWindowPosition(w.X, w.Y)
		}
	}
	ebiten.SetWindowIcon([]image.Image{appicon.Image(32), appicon.Image(64), appicon.Image(128)})
	ebiten.SetWindowTitle("Gascoigne Workshop")
	ebiten.SetWindowSizeLimits(720, 480, -1, -1) // below this the panes cannot show their controls
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetScreenClearedEveryFrame(false) // frames without changes are skipped in App.Draw
	if err := ebiten.RunGame(app); err != nil && !errors.Is(err, ebiten.Termination) {
		log.Fatal(err)
	}
	if err := app.FlushWorkspace(); err != nil {
		log.Printf("saving the collection: %v", err)
	}
	if err := app.SaveState(); err != nil {
		log.Printf("saving state: %v", err)
	}
	if err := app.Err(); err != nil {
		log.Fatal(err)
	}
}

// flagSet reports whether the flag was given on the command line.
func flagSet(name string) bool {
	set := false
	flag.Visit(func(f *flag.Flag) { set = set || f.Name == name })
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
