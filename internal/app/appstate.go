package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/clientconfig"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/autosave"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/serial"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/state"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/storage"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog/formpage"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"time"
)

// ports holds the adapters the app talks to, each replaced by a null one or a fake when the build gives none.
type ports struct {
	protocols    func(request.ProtocolConfig) request.Registry // builds the registry; nil has no protocols
	grpcMethods  request.GRPCMethods
	grpcSchema   request.GRPCSchema
	curl         func(request.Request) string
	store        collection.Store
	clients      clientconfig.Source // the build's client config; nil has none
	states       workspace.StateStore
	files        FileDialog                       // the desktop file chooser, replaced in tests
	folders      workspace.Folders                // what the in-app file browser lists
	now          func() time.Time                 // the clock, replaced in tests
	histories    func(dir string) history.History // nil keeps no history
	fonts        formpage.FontStore
	display      DisplayScale
	system       SystemInfo // what the System page of the settings shows; nil has no report
	settingFiles settings.SettingsStore
	previews     send.BodyPreview // shows a response body in the browser, replaced in tests
}

// frameState holds what one frame reads and decides: size, redraw flag, update rate, input, pacing, window geometry and the widgets that outlive a frame.
type frameState struct {
	w              int
	h              int
	dirty          bool         // something changed since the last draw
	drawEveryFrame bool         // every frame is built and drawn, as a capture needs (Options.DrawEveryFrame)
	drawTree       *view.El     // the tree the last step built and updated; Draw paints it
	treeStale      bool         // a handler or a step after the build changed what drawTree shows
	builds         int          // view trees built, so tests can count them
	tps            int          // the update rate last set
	lastIn         core.Input   // the previous frame's input
	pacer          engine.Pacer // update rate from idle time
	windowScale    float64      // window units per logical pixel, set by Layout
	winX           int
	winY           int    // window position, read each frame
	title          string // the window title last set
	tips           widget.TooltipState
	sampler        core.Sampler
	sidebarSplit   widget.SplitState // width of the collections sidebar
	paneSplit      widget.SplitState // request pane share of the request/response area
}

// persistState holds what is remembered between runs and written in the background: the state file, the collections' autosave and the trust list.
type persistState struct {
	statePath        string                                     // where the UI state is remembered
	stateAccess      storage.Access                             // read-only when a newer build wrote the file
	saves            map[*request.Collection]*autosave.Debounce // when each open collection was last written
	writes           *serial.Writer                             // the background writes, one per file at a time
	keys             changeKeys                                 // the comparable forms of the collection and the state, recomputed only on frames where something could have changed
	stateSave        autosave.Debounce                          // when the remembered state was last written
	welcomed         string                                     // the version whose welcome dialog was shown
	initial          state.Window                               // window geometry of the last run
	defaultWorkspace string                                     // beside it is the collections folder, where a first save goes
	trusted          request.AccessSet                          // what the user allowed; remembered in the state file
}

// settingsState holds the settings in effect; the dialog that edits them (over.Settings) owns the file, its path and what is unsaved.
type settingsState struct {
	settings settings.Settings
}
