package window

import (
	"context"
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
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog/formpage"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/systeminfo"
)

// Stores is everything New builds the window from: the three stores the
// program shares between its goroutines and the window (the open tabs, the
// requests in flight, the bus that carries what a goroutine reports back),
// the adapters, and the startup switches. The caller makes the three stores
// because it owns their lifetime; the window makes the stores it is the host
// of (the library, the state file, the autosave, the write queue).
type Stores struct {
	Tabs  *tabs.Set
	Sends *sends.Sends
	Bus   *bus.Bus
	Ports Ports
	Start Start
}

// Ports are the adapters the window talks to. A nil port is replaced by a
// null one (nulls.go) or, for the optional ones, kept nil and checked where
// it is used; New says which.
type Ports struct {
	Protocols     func(request.ProtocolConfig) request.Registry // builds the registry; nil has no protocols
	GRPCMethods   request.GRPCMethods
	GRPCSchema    request.GRPCSchema
	Curl          func(request.Request) string
	Store         collection.Store
	States        workspace.StateStore
	Clients       clientconfig.Source              // the build's client config; nil has none
	Files         FileDialog                       // the desktop file chooser
	Folders       workspace.Folders                // what the in-app file browser lists
	Now           func() time.Time                 // the clock; nil is the wall clock
	History       func(dir string) history.History // nil keeps no history
	Fonts         formpage.FontStore
	System        SystemInfo // what the System page of the settings shows; nil has no report
	SettingsFiles settings.SettingsStore
	Previews      send.BodyPreview // shows a response body in the browser
}

// Start are the startup switches: where the files are, what to open and how
// the frame is paced.
type Start struct {
	SettingsPath     string                    // settings file; "" keeps settings in memory
	StatePath        string                    // remembered UI state; "" keeps it in memory
	ThemesDir        string                    // folder of user theme files (*.json); "" for none
	WorkspaceDir     string                    // collection folder; "" uses the remembered one, else the client config, else the sample
	DefaultWorkspace string                    // where saving goes when there is no workspace yet
	HostOS           string                    // the OS shortcut text follows; "" is the running one
	Welcome          bool                      // open the welcome dialog when this version has not been welcomed yet
	WelcomeEvery     bool                      // with Welcome, open the dialog at every start, not once per version
	ScanFonts        bool                      // look for installed fonts in the background, to apply the saved font families
	DrawEveryFrame   bool                      // build and draw every frame, even an idle one: a capture reads the frame it just drew
	RetryDelay       time.Duration             // wait before a dropped connection reopens; 0 is two seconds
	Sample           func() request.Collection // the collection shown without a workspace or client config; nil uses collection.GitHubSample
}

// FileDialog asks the desktop for a file name. The toolkit's
// desktop/filedialog package implements it; a test fake stands in elsewhere.
type FileDialog interface {
	// Available reports whether there is a dialog to show; without one the
	// caller opens its in-app file browser.
	Available() bool
	// Suggested is the program to install for native dialogs on this
	// desktop ("zenity", "kdialog", "zenity or kdialog"), "" when there is
	// nothing to suggest.
	Suggested() string
	// Choose shows the dialog and returns the chosen path, or
	// filedialog.ErrCancelled. Cancelling ctx closes it. SaveFile asks for a
	// new file named name; OpenFile ignores name.
	Choose(ctx context.Context, title string, mode filedialog.Mode, name string) (string, error)
}

// SystemInfo reports what the program runs on: the system, the display and
// the GPU rendering. The toolkit's desktop/systeminfo package implements it.
type SystemInfo interface {
	// Read is the report as it is now; the numbers that change (frame rate,
	// memory) are read at each call.
	Read() systeminfo.Report
}

// ports is Ports as the window keeps it, with the null adapters in place.
type ports struct {
	protocols    func(request.ProtocolConfig) request.Registry
	grpcMethods  request.GRPCMethods
	grpcSchema   request.GRPCSchema
	curl         func(request.Request) string
	store        collection.Store
	states       workspace.StateStore
	clients      clientconfig.Source
	files        FileDialog
	folders      workspace.Folders
	now          func() time.Time
	histories    func(dir string) history.History
	fonts        formpage.FontStore
	system       SystemInfo
	settingFiles settings.SettingsStore
	previews     send.BodyPreview
}
