package app

import (
	"cmp"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/fileversion"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/autosave"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"net/http"
	"net/http/cookiejar"
	"time"
)

// Options are the startup switches of the command line.
type Options struct {
	DrawEveryFrame   bool                                          // build and draw every frame, even an idle one: a capture reads the frame it just drew
	SettingsPath     string                                        // settings file; "" keeps settings in memory
	Welcome          bool                                          // open the welcome dialog when this version has not been welcomed yet
	WelcomeEvery     bool                                          // with Welcome, open the dialog at every start, not once per version
	StatePath        string                                        // remembered UI state; "" keeps it in memory
	ThemesDir        string                                        // folder of user theme files (*.json); "" for none
	ScanFonts        bool                                          // look for installed fonts in the background, to apply the saved font families
	Protocols        func(request.ProtocolConfig) request.Registry // builds the protocols; nil has none
	GRPCMethods      request.GRPCMethods                           // lists gRPC methods by reflection; nil cannot
	GRPCSchema       request.GRPCSchema                            // describes a gRPC method's request message; nil cannot
	Curl             func(request.Request) string                  // the cURL command of a request; nil copies nothing
	Store            collection.Store                              // where the collection is saved; nil saves nothing
	States           workspace.StateStore                          // where the UI state is kept; nil keeps none
	Files            workspace.FileDialog                          // the desktop file chooser; nil uses the in-app file browser
	Folders          workspace.Folders                             // the folders the in-app file browser lists; nil lists none
	History          func(dir string) response.History             // the history in a folder; nil keeps none
	System           settings.SystemInfo                           // the system and GPU report of the settings; nil has none
	KeyRepeat        settings.KeyRepeat                            // the desktop's key repeat timing; nil keeps the built-in one
	Display          settings.DisplayScale                         // the desktop's display scale when the window system hides it; nil has none
	Fonts            settings.FontStore                            // installed fonts; nil has none
	SettingsFiles    settings.SettingsStore                        // the settings file; nil reads the defaults and cannot save
	Previews         response.BodyPreview                          // shows a response body in the browser; nil has none
	Clock            workspace.Clock                               // the time source; nil uses the wall clock
	RetryDelay       time.Duration                                 // wait before a dropped connection reopens; 0 is two seconds
	HostOS           string                                        // the OS shortcut text follows; "" is the running one
	BaseScale        float64                                       // the platform's interface factor under the user's scale; 0 is 1
	WorkspaceDir     string                                        // collection folder; "" uses the remembered one, else the client config, else the sample
	Sample           func() request.Collection                     // the collection shown without a workspace or client config; nil uses collection.GitHubSample
	Clients          collection.ClientConfigs                      // the client config a build offers without a workspace; nil has none
	DefaultWorkspace string                                        // where saving goes when there is no workspace yet
}

// startCollection is the collection shown before a workspace or client
// config replaces it.
func startCollection(sample func() request.Collection) request.Collection {
	if sample == nil {
		return collection.GitHubSample()
	}
	return sample()
}

// withDefaults fills what a caller may leave out: null ports for the
// adapters the app always calls, the retry delay and the base scale.
func (opts Options) withDefaults() Options {
	opts.SettingsFiles = cmp.Or[settings.SettingsStore](opts.SettingsFiles, noSettingFiles{})
	opts.States = cmp.Or[workspace.StateStore](opts.States, noStates{})
	opts.Store = cmp.Or[collection.Store](opts.Store, noStore{})
	opts.Fonts = cmp.Or[settings.FontStore](opts.Fonts, noFonts{})
	opts.Files = cmp.Or[workspace.FileDialog](opts.Files, noFileDialog{})
	opts.Folders = cmp.Or[workspace.Folders](opts.Folders, noFolders{})
	opts.Previews = cmp.Or[response.BodyPreview](opts.Previews, noBrowser{})
	opts.RetryDelay = cmp.Or(opts.RetryDelay, defaultRetryDelay)
	opts.BaseScale = cmp.Or(opts.BaseScale, 1)
	return opts
}

// wallClock is the time source when no Clock port is given.
func (opts Options) now() func() time.Time {
	if opts.Clock == nil {
		return time.Now
	}
	return opts.Clock.Now
}

// loadSettings reads the settings file named by opts, the defaults when it
// names none, and the user themes beside it.
func loadSettings(opts Options) (settings.Settings, error, []string) {
	st, loadErr := settings.Default(), error(nil)
	if opts.SettingsPath != "" {
		st, loadErr = opts.SettingsFiles.Load(opts.SettingsPath)
	}
	themes, themeProblems := theme.LoadUserThemes(opts.ThemesDir)
	theme.SetUserThemes(themes)
	return st, loadErr, themeProblems
}

// NewApp builds the window state from opts: settings, user themes and the start collection.
func NewApp(opts Options) (*App, error) {
	opts = opts.withDefaults()
	st, loadErr, themeProblems := loadSettings(opts)
	ctx, err := core.NewCtx(theme.ThemeWith(st))
	if err != nil {
		return nil, err
	}
	a := &App{
		ctx:        ctx,
		baseScale:  opts.BaseScale,
		retryDelay: opts.RetryDelay,
		sample:     opts.Sample,
		widgets:    map[*tabs.Tab]*reqTab{},
		bus:        bus.New(),
		status:     "Ready",
		jar:        newJar(),
		list:       tabs.List{Drag: -1},
		ports: ports{
			protocols: opts.Protocols, grpcMethods: opts.GRPCMethods, grpcSchema: opts.GRPCSchema, curl: opts.Curl,
			store: opts.Store, clients: opts.Clients, states: opts.States, files: opts.Files, folders: opts.Folders,
			now: opts.now(), histories: opts.History, fonts: opts.Fonts, display: opts.Display, system: opts.System,
			settingFiles: opts.SettingsFiles, previews: opts.Previews,
		},
		treeState:    treeState{expanded: map[*request.Folder]bool{}},
		frameState:   frameState{dirty: true, drawEveryFrame: opts.DrawEveryFrame},
		persistState: persistState{saves: map[*request.Collection]*autosave.Debounce{}, defaultWorkspace: opts.DefaultWorkspace, statePath: opts.StatePath},
		settingsState: settingsState{
			settings: st, settingsPath: opts.SettingsPath, settingsAccess: fileversion.AccessAfter(loadErr),
		},
	}
	a.driver.app = a
	if loadErr != nil {
		a.status = loadErr.Error()
	}
	a.prefs.Stamp(a, a.settingsPath)
	a.reg = a.newRegistry()
	ctx.Clipboard = core.ReportClipboard(ctx.Clipboard, a.clipboardNote)
	if opts.HostOS != "" {
		ctx.HostOS = opts.HostOS
	}
	a.sampler.HostOS = ctx.HostOS
	ctx.Now = func() time.Time { return a.now() } // the caret blinks by the app clock
	a.noteReadOnly(a.settingsAccess, loadErr)
	if opts.KeyRepeat != nil {
		delay, interval := opts.KeyRepeat.Timing()
		a.sampler.Repeat = core.KeyRepeat{Delay: delay, Interval: interval}
	}
	for _, s := range shortcuts {
		a.actionIDs = append(a.actionIDs, s.id)
	}
	if opts.ScanFonts {
		a.prefs.ScanFonts(a)
	}
	a.openWorkspace(opts.WorkspaceDir)
	stateErr := a.loadState(opts.StatePath)
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

// loadState reads the remembered state at path and applies it; the error is
// the reason it could not be read, if any.
func (a *App) loadState(path string) error {
	stored, err := a.states.Load(path)
	a.stateAccess = fileversion.AccessAfter(err)
	a.noteReadOnly(a.stateAccess, err)
	a.initial = stored.Window
	a.restore(stored)
	a.welcomed = stored.Welcomed
	return err
}

// newJar is the cookie jar of a session; with no options it cannot fail.
func newJar() http.CookieJar {
	jar, _ := cookiejar.New(nil)
	return jar
}
