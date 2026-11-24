package app

import (
	"cmp"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/clientconfig"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/sends"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog/formpage/fontlist"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/window"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
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
	Fonts            fontlist.FontStore                            // installed fonts; nil has none
	SettingsFiles    settings.SettingsStore                        // the settings file; nil reads the defaults and cannot save
	Previews         send.BodyPreview                              // shows a response body in the browser; nil has none
	Clock            Clock                                         // the time source; nil uses the wall clock
	RetryDelay       time.Duration                                 // wait before a dropped connection reopens; 0 is two seconds
	HostOS           string                                        // the OS shortcut text follows; "" is the running one
	BaseScale        float64                                       // the platform's interface factor under the user's scale; 0 is 1
	WorkspaceDir     string                                        // collection folder; "" uses the remembered one, else the client config, else the sample
	Sample           func() request.Collection                     // the collection shown without a workspace or client config; nil uses collection.GitHubSample
	Clients          clientconfig.Source                           // the client config a build offers without a workspace; nil has none
	StartWith        collection.StartWith                          // what opens when no workspace folder is given and none is remembered; the zero value is the sample
	DefaultWorkspace string                                        // where saving goes when there is no workspace yet
}

// stores is Options as the window takes it: the three shared stores the app
// builds, the adapters, and the switches.
func (opts Options) stores() window.Stores {
	var now func() time.Time
	if opts.Clock != nil {
		now = opts.Clock.Now
	}
	return window.Stores{
		Tabs: tabs.NewSet(), Sends: sends.New(), Bus: bus.New(),
		Ports: window.Ports{
			Protocols: opts.Protocols, GRPCMethods: opts.GRPCMethods, GRPCSchema: opts.GRPCSchema, Curl: opts.Curl,
			Store: opts.Store, States: opts.States, Clients: opts.Clients, Files: opts.Files, Folders: opts.Folders,
			Now: now, History: opts.History, Fonts: opts.Fonts, System: opts.System,
			SettingsFiles: opts.SettingsFiles, Previews: opts.Previews,
		},
		Start: window.Start{
			SettingsPath: opts.SettingsPath, StatePath: opts.StatePath, ThemesDir: opts.ThemesDir, WorkspaceDir: opts.WorkspaceDir, DefaultWorkspace: opts.DefaultWorkspace, HostOS: opts.HostOS,
			Welcome: opts.Welcome, WelcomeEvery: opts.WelcomeEvery, ScanFonts: opts.ScanFonts, DrawEveryFrame: opts.DrawEveryFrame,
			RetryDelay: opts.RetryDelay, Sample: opts.Sample, StartWith: opts.StartWith,
		},
	}
}

// NewApp builds the window from opts and the app's own state around it: the
// desktop's key repeat and display scale and the platform's base scale.
func NewApp(opts Options) (*App, error) {
	win, err := window.New(opts.stores())
	if err != nil {
		return nil, err
	}
	app := &App{window: win, baseScale: cmp.Or(opts.BaseScale, 1), display: opts.Display}
	app.sampler.HostOS = win.HostOS()
	if opts.KeyRepeat != nil {
		delay, interval := opts.KeyRepeat.Timing()
		app.sampler.Repeat = core.KeyRepeat{Delay: delay, Interval: interval}
	}
	return app, nil
}
