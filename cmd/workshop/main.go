// Command workshop is a GPU-rendered desktop client for REST, WebSocket,
// MCP and other API calls.
package main

import (
	"bytes"
	"errors"
	"flag"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/adapter/collectionstore"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history/adapter/historyfile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/adapter/preview"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/adapter/protocol"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/adapter/folders"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/displayscale"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/fonts"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/keyrepeat"
	"image"
	"image/png"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/app"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/adapter/settingsfile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/adapter/statefile"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/appicon"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/systeminfo"
)

const (
	windowWidth  = 1280
	windowHeight = 800
)

// previewLifetime is how long a response stays served for the browser.
const previewLifetime = 5 * time.Minute

// launch is what the command line asks for: the app's options and the
// one-shot icon export.
type launch struct {
	opts    app.Options
	iconPNG *string
	cpuProf *string
}

func main() {
	run := defineFlags(flag.CommandLine)
	adapters(&run.opts)
	flag.Parse()
	if *run.iconPNG != "" {
		if err := writeIcon(*run.iconPNG); err != nil {
			log.Fatal(err)
		}
		return
	}
	// A normal run opens the welcome dialog once per version.
	stopProfile, err := startProfile(*run.cpuProf)
	if err != nil {
		log.Fatal(err)
	}
	defer stopProfile()
	run.opts.Welcome = true
	run.opts.WelcomeEvery = welcomeEveryStart
	run.opts.BaseScale = settings.BaseScale(runtime.GOOS)

	window, err := app.NewApp(run.opts)
	if err != nil {
		log.Fatal(err)
	}
	if err := openWindow(window, windowWidth, windowHeight); err != nil {
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
	run := &launch{}
	opts := &run.opts
	set.StringVar(&opts.SettingsPath, "settings", settingsfile.DefaultPath(), "settings file; empty keeps settings in memory")
	set.StringVar(&opts.StatePath, "state", statefile.DefaultPath(), "remembered window and tabs; empty keeps them in memory")
	set.StringVar(&opts.ThemesDir, "themes", settingsfile.ThemesDir(), "folder of theme files (*.json) that extend the dark or light theme; empty for none")
	set.StringVar(&opts.WorkspaceDir, "workspace", "", "folder to load and save the collection; empty uses the remembered collections, else the example server (default build) or none (release build)")
	run.cpuProf = set.String("cpuprofile", "", "write a CPU profile of the run to this file, for `go tool pprof`")
	run.iconPNG = set.String("icon-png", "", "write the program icon (256x256 PNG) to this path and exit, for packaging")
	return run
}

// adapters gives opts the real implementation of every response.
func adapters(opts *app.Options) {
	opts.ScanFonts = true
	opts.Protocols = protocol.Configured
	opts.GRPCMethods = protocol.ListGRPCMethods
	opts.GRPCSchema = protocol.GRPCInputSchema
	opts.Curl = protocol.Curl
	opts.Store = collectionstore.Files{}
	opts.Sample = collection.Sample // which start is a build decision, see clients_*.go
	opts.StartWith = startWith
	opts.States = statefile.Files{}
	opts.Files = filedialog.System{}
	opts.Folders = folders.Disk{}
	opts.History = func(dir string) history.History { return historyfile.Dir(dir) }
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
		desktop.SetDockIcon(dock.Bytes()) // the Dock ignores the window icon
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
