package app

import (
	cport "github.com/gascoigne-ai/gascoigne-workshop/internal/collection/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	reqport "github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
	rport "github.com/gascoigne-ai/gascoigne-workshop/internal/response/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/ui/sentlog"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/fileversion"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	sport "github.com/gascoigne-ai/gascoigne-workshop/internal/settings/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/ui/colorpage"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/ui/settingsui"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/ui/shortcutpage"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/autosave"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/serial"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/model/state"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/port"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"time"
)

// ports holds the adapters the app talks to, each replaced by a null one or a fake when the build gives none.
type ports struct {
	protocols    func(reqport.ProtocolConfig) reqport.Registry // builds the registry; nil has no protocols
	grpcMethods  reqport.GRPCMethods
	grpcSchema   reqport.GRPCSchema
	curl         func(model.Request) string
	store        cport.Store
	clients      cport.ClientConfigs // the build's client config; nil has none
	states       port.StateStore
	files        port.FileDialog                // the desktop file chooser, replaced in tests
	folders      port.Folders                   // what the in-app file browser lists
	now          func() time.Time               // the clock, replaced in tests
	histories    func(dir string) rport.History // nil keeps no history
	fonts        sport.FontStore
	display      sport.DisplayScale
	system       sport.SystemInfo // what the System page of the settings shows; nil has no report
	settingFiles sport.SettingsStore
	previews     rport.BodyPreview // shows a response body in the browser, replaced in tests
}

// treeState holds the sidebar: selection, filter, drag, expanded folders and scroll.
type treeState struct {
	expanded      map[*model.Folder]bool
	selected      model.Node       // the tree node last clicked or moved to
	treeFocus     bool             // the tree has the keyboard
	filter        widget.TextField // the sidebar filter
	sent          sentlog.Log      // requests sent in this workspace, newest first
	showSent      bool             // the sidebar lists sent requests instead of the tree
	dragNode      model.Node       // the tree node being dragged, none when not dragging
	pressedFolder *model.Folder    // the folder row the button went down on, toggled on release there
	hoverFolder   *model.Folder    // the closed folder a drag has been over
	hoverTicks    int              // for how many frames
	dropTarget    model.Node       // the node the pointer is over while dragging
	treeScroll    widget.ScrollState
}

// tableState holds the key-value tables: the cell edit, the row drag, bulk edit, the selected row and the column widths.
type tableState struct {
	edit      *cellEdit                     // the table cell being edited, if any
	dragRow   *rowSelection                 // the table row being dragged by its grip, nil when none
	bulk      *bulkEdit                     // the table in bulk-edit mode, if any
	rowSel    *rowSelection                 // the table row the keyboard acts on
	colWidths map[string]*widget.TableState // the column sizes of each table, by name
}

// frameState holds what one frame reads and decides: size, redraw flag, update rate, input, pacing, window geometry and the widgets that outlive a frame.
type frameState struct {
	w            int
	h            int
	dirty        bool         // something changed since the last draw
	tps          int          // the update rate last set
	lastIn       core.Input   // the previous frame's input
	pacer        engine.Pacer // update rate from idle time
	windowScale  float64      // window units per logical pixel, set by Layout
	winX         int
	winY         int    // window position, read each frame
	title        string // the window title last set
	tips         widget.TooltipState
	sampler      core.Sampler
	sidebarSplit widget.SplitState  // width of the collections sidebar
	paneSplit    widget.SplitState  // request pane share of the request/response area
	tabsScroll   widget.ScrollState // sideways scroll of the request tab strip
	toasts       widget.ToastStack
}

// persistState holds what is remembered between runs and written in the background: the state file, the collections' autosave and the trust list.
type persistState struct {
	statePath        string                                   // where the UI state is remembered
	stateAccess      fileversion.Access                       // read-only when a newer build wrote the file (decision 22)
	saves            map[*model.Collection]*autosave.Debounce // when each open collection was last written
	writes           serial.Queue                             // the background writes in flight and waiting, by file
	keys             changeKeys                               // the comparable forms of the collection and the state, recomputed only on frames where something could have changed
	stateSave        autosave.Debounce                        // when the remembered state was last written
	welcomed         string                                   // the version whose welcome dialog was shown
	initial          state.Window                             // window geometry of the last run
	setIsRemembered  bool                                     // the open set is the remembered list (opened from it, or changed by the user since), so it is written whole
	unreadable       []settings.OpenCollection                // remembered folders that failed to load; kept in the list so a mount or a fix brings them back
	defaultWorkspace string                                   // beside it is the collections folder, where a first save goes
	trusted          model.AccessSet                          // what the user allowed; remembered in the state file
}

// settingsState holds the settings, their file and the settings dialog with its pages.
type settingsState struct {
	settings           settings.Settings
	settingsPath       string                    // "" keeps settings in memory only
	settingsAccess     fileversion.Access        // read-only when a newer build wrote the file (decision 22)
	menu               bool                      // settings menu is open
	actionIDs          []string                  // the ids of the shortcut actions, for completing settings.json
	cancelSettingsPick func()                    // takes back a file question the settings dialog asked
	settingsUnsaved    bool                      // settings were changed but not written (auto-save is off)
	prefs              settingsui.SettingsDialog // the settings dialog
	shortcutKeys       shortcutpage.Page         // the Shortcuts page of the settings dialog
	colors             colorpage.Page            // the Colors page of the settings dialog
}
