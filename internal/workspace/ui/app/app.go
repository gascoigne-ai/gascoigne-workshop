package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/collection"
	cport "github.com/gascoigne-ai/gascoigne-workshop/internal/collection/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/ui/destpicker"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	reqport "github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/ui/calltrace"
	rport "github.com/gascoigne-ai/gascoigne-workshop/internal/response/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/ui/sentlog"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/fileversion"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	sport "github.com/gascoigne-ai/gascoigne-workshop/internal/settings/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/ui/colorpage"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/ui/settingsui"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/ui/shortcutpage"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/autosave"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/serial"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/model/state"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/port"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"net/http"
	"time"
)

const (
	version = "v0.1.0-alpha"

	// defaultTimeoutSec is used when neither the request nor the settings name one.
	defaultTimeoutSec = 60
	maxTimeout        = 24 * time.Hour // longer would overflow time.Duration for a huge setting
)

// App is the whole window. It implements ebiten.Game.
type App struct {
	ctx         *core.Ctx
	retryDelay  time.Duration // wait before a dropped connection reopens
	baseScale   float64       // the platform's interface factor under the user's scale
	reg         reqport.Registry
	protocols   func(reqport.ProtocolConfig) reqport.Registry // builds the registry; nil has no protocols
	grpcMethods reqport.GRPCMethods
	grpcSchema  reqport.GRPCSchema
	curl        func(model.Request) string
	open        collections.Set         // the open collections, in tree order, each with its folder and environment
	sample      func() model.Collection // the collection shown when nothing is saved or configured
	expanded    map[*model.Folder]bool
	list        tabs.List             // the open tabs, the active one and the one being dragged
	widgets     map[*tabs.Tab]*reqTab // the widget state of each open tab
	bus         *bus.Bus              // what goroutines report back, drained once per frame
	status      string
	w, h        int
	shot        screenshot

	settings           settings.Settings
	settingsPath       string             // "" keeps settings in memory only
	settingsAccess     fileversion.Access // read-only when a newer build wrote the file (decision 22)
	menu               bool               // settings menu is open
	edit               *cellEdit          // the table cell being edited, if any
	dragRow            *rowSelection      // the table row being dragged by its grip, nil when none
	bulk               *bulkEdit          // the table in bulk-edit mode, if any
	actionIDs          []string           // the ids of the shortcut actions, for completing settings.json
	store              cport.Store
	clients            cport.ClientConfigs // the build's client config; nil has none
	states             port.StateStore
	files              port.FileDialog                 // the desktop file chooser, replaced in tests
	folders            port.Folders                    // what the in-app file browser lists
	browser            *fileBrowser                    // the in-app file browser while it is open
	pickerHost         destpicker.Host                 // what the open picker asks of the App
	picker             *destpicker.Picker              // the move or copy destination picker while it is open
	cancelSettingsPick func()                          // takes back a file question the settings dialog asked
	settingsUnsaved    bool                            // settings were changed but not written (auto-save is off)
	prefs              settingsui.SettingsDialog       // the settings dialog
	shortcutKeys       shortcutpage.Page               // the Shortcuts page of the settings dialog
	tabsScroll         widget.ScrollState              // sideways scroll of the request tab strip
	dirty              bool                            // something changed since the last draw
	tps                int                             // the update rate last set
	lastIn             core.Input                      // the previous frame's input
	pacer              engine.Pacer                    // update rate from idle time
	colors             colorpage.Page                  // the Colors page of the settings dialog
	colWidths          map[string]*widget.ColumnWidths // the column sizes of each table, by name
	rowSel             *rowSelection                   // the table row the keyboard acts on
	selected           model.Node                      // the tree node last clicked or moved to
	treeFocus          bool                            // the tree has the keyboard
	filter             widget.TextField                // the sidebar filter
	sent               sentlog.Log                     // requests sent in this workspace, newest first
	calls              calltrace.Log                   // the call trace behind the Log button
	showSent           bool                            // the sidebar lists sent requests instead of the tree
	dragNode           model.Node                      // the tree node being dragged, none when not dragging
	pressedFolder      *model.Folder                   // the folder row the button went down on, toggled on release there
	hoverFolder        *model.Folder                   // the closed folder a drag has been over
	hoverTicks         int                             // for how many frames
	dropTarget         model.Node                      // the node the pointer is over while dragging
	dialog             *dialog                         // the open dialog, if any
	toasts             widget.ToastStack
	now                func() time.Time               // the clock, replaced in tests
	histories          func(dir string) rport.History // nil keeps no history
	fonts              sport.FontStore
	display            sport.DisplayScale
	system             sport.SystemInfo // what the System page of the settings shows; nil has no report
	windowScale        float64          // window units per logical pixel, set by Layout
	settingFiles       sport.SettingsStore
	previews           rport.BodyPreview // shows a response body in the browser, replaced in tests
	sampler            core.Sampler
	treeScroll         widget.ScrollState
	tips               widget.TooltipState
	sidebarSplit       widget.SplitState                        // width of the collections sidebar
	paneSplit          widget.SplitState                        // request pane share of the request/response area
	statePath          string                                   // where the UI state is remembered
	stateAccess        fileversion.Access                       // read-only when a newer build wrote the file (decision 22)
	varsTarget         varsTarget                               // what the Vars pane edits; the zero value is the active environment's or the collection's
	title              string                                   // the window title last set
	jar                http.CookieJar                           // cookies the server set, kept until the app closes
	defaultWorkspace   string                                   // beside it is the collections folder, where a first save goes
	trusted            model.AccessSet                          // what the user allowed; remembered in the state file
	setIsRemembered    bool                                     // the open set is the remembered list (opened from it, or changed by the user since), so it is written whole
	unreadable         []settings.OpenCollection                // remembered folders that failed to load; kept in the list so a mount or a fix brings them back
	saves              map[*model.Collection]*autosave.Debounce // when each open collection was last written
	writes             serial.Queue                             // the background writes in flight and waiting, by file
	keys               changeKeys                               // the comparable forms of the collection and the state, recomputed only on frames where something could have changed
	stateSave          autosave.Debounce                        // when the remembered state was last written
	welcomed           string                                   // the version whose welcome dialog was shown
	initial            state.Window                             // window geometry of the last run
	winX, winY         int                                      // window position, read each frame
}

func (a *App) clickTree(row collection.Row) {
	a.selectNode(row.Node())
	if row.Folder {
		a.expanded[row.Dir] = !a.expanded[row.Dir]
		return
	}
	a.openRequest(row.Req)
}

func (a *App) newRequest() { a.newRequestIn(a.selectedFolder()) }
