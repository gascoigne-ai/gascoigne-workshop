package app

import (
	"cmp"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/clientconfig"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/sends"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/autosave"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/serial"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/storage"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/assertions"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/auth"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/body"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/graphql"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/grpc"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/mcp"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/urlbar"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/ws"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog/formpage"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/response"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sidebar"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/statusbar"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/tabstrip"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"net/http"
	"net/http/cookiejar"
	"path/filepath"
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
	Files            FileDialog                                    // the desktop file chooser; nil uses the in-app file browser
	Folders          workspace.Folders                             // the folders the in-app file browser lists; nil lists none
	History          func(dir string) history.History              // the history in a folder; nil keeps none
	System           SystemInfo                                    // the system and GPU report of the settings; nil has none
	KeyRepeat        KeyRepeat                                     // the desktop's key repeat timing; nil keeps the built-in one
	Display          DisplayScale                                  // the desktop's display scale when the window system hides it; nil has none
	Fonts            formpage.FontStore                            // installed fonts; nil has none
	SettingsFiles    settings.SettingsStore                        // the settings file; nil reads the defaults and cannot save
	Previews         send.BodyPreview                              // shows a response body in the browser; nil has none
	Clock            Clock                                         // the time source; nil uses the wall clock
	RetryDelay       time.Duration                                 // wait before a dropped connection reopens; 0 is two seconds
	HostOS           string                                        // the OS shortcut text follows; "" is the running one
	BaseScale        float64                                       // the platform's interface factor under the user's scale; 0 is 1
	WorkspaceDir     string                                        // collection folder; "" uses the remembered one, else the client config, else the sample
	Sample           func() request.Collection                     // the collection shown without a workspace or client config; nil uses collection.GitHubSample
	Clients          clientconfig.Source                           // the client config a build offers without a workspace; nil has none
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
	opts.Fonts = cmp.Or[formpage.FontStore](opts.Fonts, noFonts{})
	opts.Files = cmp.Or[FileDialog](opts.Files, noFileDialog{})
	opts.Folders = cmp.Or[workspace.Folders](opts.Folders, noFolders{})
	opts.Previews = cmp.Or[send.BodyPreview](opts.Previews, noBrowser{})
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
		jar:        newJar(),
		tabs:       tabs.NewSet(),
		sends:      sends.New(),
		ports: ports{
			protocols: opts.Protocols, grpcMethods: opts.GRPCMethods, grpcSchema: opts.GRPCSchema, curl: opts.Curl,
			store: opts.Store, clients: opts.Clients, states: opts.States, files: opts.Files, folders: opts.Folders,
			now: opts.now(), histories: opts.History, fonts: opts.Fonts, display: opts.Display, system: opts.System,
			settingFiles: opts.SettingsFiles, previews: opts.Previews,
		},
		frameState:    frameState{dirty: true, drawEveryFrame: opts.DrawEveryFrame},
		persistState:  persistState{saves: map[*request.Collection]*autosave.Debounce{}, defaultWorkspace: opts.DefaultWorkspace, statePath: opts.StatePath},
		settingsState: settingsState{settings: st},
	}
	a.writes = serial.NewWriter(a.bus, a.drain)
	a.keys = autosave.NewKeys(keysHost{a})
	a.cols = collections.NewLibrary(a.store, collectionsDir(opts.DefaultWorkspace), collections.Deps{Settings: librarySettings{a}, Saves: librarySaves{a}})
	a.driver.app = a
	a.notices.SetStatus("Ready")
	settingsAccess := storage.AccessAfter(loadErr)
	over := overlaysHost{a}
	a.over = overlays.New(overlays.Deps{
		Ctx: ctx, Window: over, Clock: over, Desktop: over, Folders: over, Bus: over,
		Settings: settingsdialog.Deps{
			Settings: settingsHost{a}, Window: settingsHost{a}, Files: settingsHost{a}, Dialogs: settingsHost{a},
			Bus: settingsHost{a}, Looks: settingsHost{a}, Keys: settingsHost{a},
			Path: opts.SettingsPath, Store: opts.SettingsFiles, Access: settingsAccess, Fonts: opts.Fonts, System: opts.System,
		},
	})
	a.side = sidebar.New(sidebar.Deps{
		Collections: sidebarHost{a}, Requests: sidebarHost{a}, Nodes: sidebarHost{a}, Drops: sidebarHost{a},
		Folders: sidebarHost{a}, Extras: sidebarHost{a}, Collection: collectionHost{a}, Workspace: sidebarHost{a},
		Menus: sidebarHost{a}, Sent: a,
	})
	a.kv = kvtable.New(kvtable.Deps{Tables: kvtableHost{a}, Tabs: kvtableHost{a}, Menus: kvtableHost{a}, Secrets: kvtableHost{a}, Styles: kvtableHost{a}})
	a.urlbar = urlbar.New(urlbar.Deps{Tabs: urlbarHost{a}, Sends: urlbarHost{a}, Connections: urlbarHost{a}, GRPC: urlbarHost{a}, Protocols: urlbarHost{a}, Styles: urlbarHost{a}, Hints: urlbarHost{a}})
	a.body = body.New(body.Deps{Tabs: bodyHost{a}, Slots: bodyHost{a}, Vars: bodyHost{a}, Files: bodyHost{a}, Schemas: bodyHost{a}, Window: bodyHost{a}})
	a.graphql = graphql.New(graphql.Deps{Tabs: graphqlHost{a}, Query: a.body, Sends: graphqlHost{a}, Window: graphqlHost{a}})
	a.auth = auth.New(auth.Deps{Tabs: authHost{a}, Vars: authHost{a}, Trust: authHost{a}, Cookies: authHost{a}, Window: authHost{a}})
	a.assertions = assertions.New(assertions.Deps{Tabs: assertionsHost{a}, Vars: assertionsHost{a}, Status: assertionsHost{a}, Defaults: assertionsHost{a}})
	a.grpc = grpc.New(grpc.Deps{Tabs: grpcHost{a}, Methods: grpcHost{a}, Vars: grpcHost{a}, Files: grpcHost{a}, Connections: grpcHost{a}, Window: grpcHost{a}})
	a.mcp = mcp.New(mcp.Deps{Tabs: mcpHost{a}, Bus: mcpHost{a}, Composer: mcpHost{a}, Dialogs: mcpHost{a}, Looks: mcpHost{a}})
	a.ws = ws.New(ws.Deps{Tabs: wsHost{a}, Outbox: wsHost{a}, Tools: wsHost{a}, Window: wsHost{a}, Composers: wsHost{a}})
	a.resp = response.New(response.Deps{Tabs: responseHost{a}, Slots: responseHost{a}, Headers: responseHost{a}, Files: responseHost{a}, Exports: responseHost{a}, Window: responseHost{a}, Hints: responseHost{a}, Defaults: responseHost{a}})
	a.strip = tabstrip.New(tabstrip.Deps{Tabs: tabstripHost{a}, Marks: tabstripHost{a}, Menus: tabstripHost{a}, Requests: tabstripHost{a}, Hints: tabstripHost{a}})
	a.bar = statusbar.New(statusbar.Deps{Status: &a.notices, Log: a.over.Calls(), Welcome: a, Version: version})
	if loadErr != nil {
		a.notices.SetStatus(loadErr.Error())
	}
	a.reg = a.newRegistry()
	ctx.Clipboard = core.ReportClipboard(ctx.Clipboard, a.clipboardNote)
	if opts.HostOS != "" {
		ctx.HostOS = opts.HostOS
	}
	a.sampler.HostOS = ctx.HostOS
	ctx.Now = func() time.Time { return a.now() } // the caret blinks by the app clock
	a.noteReadOnly(settingsAccess, loadErr)
	if opts.KeyRepeat != nil {
		delay, interval := opts.KeyRepeat.Timing()
		a.sampler.Repeat = core.KeyRepeat{Delay: delay, Interval: interval}
	}
	if opts.ScanFonts {
		a.over.Settings.ScanFonts()
	}
	a.openWorkspace(opts.WorkspaceDir)
	stateErr := a.loadState(opts.StatePath)
	if opts.Welcome && (opts.WelcomeEvery || a.welcomed != version) {
		a.OpenWelcome()
	}
	a.side.Sent().Load(a)
	a.notices.SetStatus(stateNote(a.notices.Status(), stateErr))
	if len(themeProblems) > 0 {
		a.notices.SetStatus(stateNote(a.notices.Status(), fmt.Errorf("%s", themeProblems[0])))
	}
	a.stateSave = autosave.New(stateDebounce, stateKey(a.snapshot()))
	return a, nil
}

// collectionsDir is the folder new collections are made in: beside the
// default workspace, never inside it, since a folder with a folder.json
// below a collection is one of its folders. Empty when the program has no
// config directory.
func collectionsDir(defaultWorkspace string) string {
	if defaultWorkspace == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(defaultWorkspace), "collections")
}

// loadState reads the remembered state at path and applies it; the error is
// the reason it could not be read, if any.
func (a *App) loadState(path string) error {
	stored, err := a.states.Load(path)
	a.stateAccess = storage.AccessAfter(err)
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

// newRegistry builds the protocols from the cookie jar, the network
// settings (proxy, certificates) and the app's sampling dialog.
func (a *App) newRegistry() request.Registry {
	if a.protocols == nil {
		return request.Registry{}
	}
	s := a.settings
	return a.protocols(request.ProtocolConfig{Jar: a.jar, Proxy: s.Proxy, CACert: s.CACert, ClientCert: s.ClientCert, ClientKey: s.ClientKey, Sampler: a.mcp.Sample})
}
