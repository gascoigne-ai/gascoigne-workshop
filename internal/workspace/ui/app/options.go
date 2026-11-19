package app

import (
	"cmp"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/collection"
	cport "github.com/gascoigne-ai/gascoigne-workshop/internal/collection/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	reqport "github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
	rport "github.com/gascoigne-ai/gascoigne-workshop/internal/response/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/fileversion"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	sport "github.com/gascoigne-ai/gascoigne-workshop/internal/settings/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/autosave"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/port"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"net/http/cookiejar"
	"time"
)

// Options are the startup switches of the command line.
type Options struct {
	ScreenshotPath   string                                        // save a PNG of the first settled frame and exit
	Scene            []Step                                        // what the screenshot does to the window before the capture, in order
	SendOnStart      bool                                          // send the first request at startup
	SettingsPath     string                                        // settings file; "" keeps settings in memory
	Welcome          bool                                          // open the welcome dialog when this version has not been welcomed yet
	WelcomeEvery     bool                                          // with Welcome, open the dialog at every start, not once per version
	StatePath        string                                        // remembered UI state; "" keeps it in memory
	ThemesDir        string                                        // folder of user theme files (*.json); "" for none
	ScanFonts        bool                                          // look for installed fonts in the background, to apply the saved font families
	Protocols        func(reqport.ProtocolConfig) reqport.Registry // builds the protocols; nil has none
	GRPCMethods      reqport.GRPCMethods                           // lists gRPC methods by reflection; nil cannot
	GRPCSchema       reqport.GRPCSchema                            // describes a gRPC method's request message; nil cannot
	Curl             func(model.Request) string                    // the cURL command of a request; nil copies nothing
	Store            cport.Store                                   // where the collection is saved; nil saves nothing
	States           port.StateStore                               // where the UI state is kept; nil keeps none
	Files            port.FileDialog                               // the desktop file chooser; nil uses the in-app file browser
	Folders          port.Folders                                  // the folders the in-app file browser lists; nil lists none
	History          func(dir string) rport.History                // the history in a folder; nil keeps none
	System           sport.SystemInfo                              // the system and GPU report of the settings; nil has none
	KeyRepeat        sport.KeyRepeat                               // the desktop's key repeat timing; nil keeps the built-in one
	Display          sport.DisplayScale                            // the desktop's display scale when the window system hides it; nil has none
	Fonts            sport.FontStore                               // installed fonts; nil has none
	SettingsFiles    sport.SettingsStore                           // the settings file; nil reads the defaults and cannot save
	Previews         rport.BodyPreview                             // shows a response body in the browser; nil has none
	Clock            port.Clock                                    // the time source; nil uses the wall clock
	RetryDelay       time.Duration                                 // wait before a dropped connection reopens; 0 is two seconds
	HostOS           string                                        // the OS shortcut text follows; "" is the running one
	BaseScale        float64                                       // the platform's interface factor under the user's scale; 0 is 1
	OpenMenu         bool                                          // start with the settings menu open
	WorkspaceDir     string                                        // collection folder; "" uses the remembered one, else the client config, else the sample
	Sample           func() model.Collection                       // the collection shown without a workspace or client config; nil uses collection.GitHubSample
	Clients          cport.ClientConfigs                           // the client config a build offers without a workspace; nil has none
	DefaultWorkspace string                                        // where saving goes when there is no workspace yet
}

// startCollection is the collection shown before a workspace or client
// config replaces it.
func startCollection(sample func() model.Collection) model.Collection {
	if sample == nil {
		return collection.GitHubSample()
	}
	return sample()
}

// NewApp builds the window state from opts: settings, user themes and the start collection.
func NewApp(opts Options) (*App, error) {
	settingFiles := opts.SettingsFiles
	if settingFiles == nil {
		settingFiles = noSettingFiles{}
	}
	st, loadErr := settings.Default(), error(nil)
	if opts.SettingsPath != "" {
		st, loadErr = settingFiles.Load(opts.SettingsPath)
	}
	themes, themeProblems := appearance.LoadUserThemes(opts.ThemesDir)
	appearance.SetUserThemes(themes)
	th := appearance.ThemeWith(st)
	ctx, err := core.NewCtx(th)
	if err != nil {
		return nil, err
	}
	a := &App{
		ctx:              ctx,
		baseScale:        cmp.Or(opts.BaseScale, 1),
		sample:           opts.Sample,
		saves:            map[*model.Collection]*autosave.Debounce{},
		clients:          opts.Clients,
		system:           opts.System,
		defaultWorkspace: opts.DefaultWorkspace,
		expanded:         map[*model.Folder]bool{},
		widgets:          map[*tabs.Tab]*reqTab{},
		bus:              bus.New(),
		status:           "Ready",
		shot:             newScreenshot(opts),

		settings:       st,
		settingFiles:   settingFiles,
		settingsPath:   opts.SettingsPath,
		settingsAccess: fileversion.AccessAfter(loadErr),
		statePath:      opts.StatePath,
		menu:           opts.OpenMenu,
	}
	if loadErr != nil {
		a.status = loadErr.Error()
	}
	a.prefs.Stamp(a, a.settingsPath)
	a.jar, _ = cookiejar.New(nil)
	a.protocols, a.grpcMethods, a.grpcSchema, a.curl = opts.Protocols, opts.GRPCMethods, opts.GRPCSchema, opts.Curl
	a.reg = a.newRegistry()
	ctx.Clipboard = core.ReportClipboard(ctx.Clipboard, a.clipboardNote)
	a.list.Drag = -1
	a.dirty = true
	a.now = time.Now // the fallback when no Clock port is given
	if opts.Clock != nil {
		a.now = opts.Clock.Now
	}
	if opts.HostOS != "" {
		ctx.HostOS = opts.HostOS
	}
	a.sampler.HostOS = ctx.HostOS
	a.retryDelay = cmp.Or(opts.RetryDelay, defaultRetryDelay)
	ctx.Now = func() time.Time { return a.now() } // the caret blinks by the app clock
	a.noteReadOnly(a.settingsAccess, loadErr)
	a.store, a.states = opts.Store, opts.States
	if a.states == nil {
		a.states = noStates{}
	}
	if a.store == nil {
		a.store = noStore{}
	}
	a.display = opts.Display
	if opts.KeyRepeat != nil {
		delay, interval := opts.KeyRepeat.Timing()
		a.sampler.Repeat = core.KeyRepeat{Delay: delay, Interval: interval}
	}
	a.files, a.previews, a.histories, a.fonts = opts.Files, opts.Previews, opts.History, opts.Fonts
	if a.fonts == nil {
		a.fonts = noFonts{}
	}
	if a.files == nil {
		a.files = noFileDialog{}
	}
	a.folders = opts.Folders
	if a.folders == nil {
		a.folders = noFolders{}
	}
	if a.previews == nil {
		a.previews = noBrowser{}
	}
	for _, s := range shortcuts {
		a.actionIDs = append(a.actionIDs, s.id)
	}
	if opts.ScanFonts {
		a.prefs.ScanFonts(a)
	}
	a.openWorkspace(opts.WorkspaceDir)
	stored, stateErr := a.states.Load(opts.StatePath)
	a.stateAccess = fileversion.AccessAfter(stateErr)
	a.noteReadOnly(a.stateAccess, stateErr)
	a.initial = stored.Window
	a.restore(stored)
	a.welcomed = stored.Welcomed
	if opts.Welcome && (opts.WelcomeEvery || a.welcomed != version) {
		a.openWelcome()
	}
	a.sent.Load(a)
	a.status = stateNote(a.status, stateErr)
	if len(themeProblems) > 0 {
		a.status = stateNote(a.status, fmt.Errorf("%s", themeProblems[0]))
	}
	a.stateSave = autosave.New(stateDebounce, stateKey(a.snapshot()))
	return a, nil
}
