package app

import (
	"cmp"
	"context"
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/collection"
	cport "github.com/gascoigne-ai/gascoigne-workshop/internal/collection/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/ui/destpicker"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model/assert"
	reqport "github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/ui/calltrace"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/controller/viewer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/model/textfmt"
	rport "github.com/gascoigne-ai/gascoigne-workshop/internal/response/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/ui/sentlog"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/appearance"
	sport "github.com/gascoigne-ai/gascoigne-workshop/internal/settings/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/ui/colorpage"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/ui/settingsui"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/ui/shortcutpage"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/autosave"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/serial"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/port"
	"net/http"
	"net/http/cookiejar"
	"slices"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/controller/editor"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/controller/mcpexplorer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/ui/mcpform"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/fileversion"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/model/state"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

const (
	version = "v0.1.0-alpha"

	// defaultTimeoutSec is used when neither the request nor the settings name one.
	defaultTimeoutSec = 60
	maxTimeout        = 24 * time.Hour // longer would overflow time.Duration for a huge setting
)

// methodOrder is cycled by clicking the method chip of the URL bar; the
// protocol chip cycles protocolViews.
var methodOrder = []string{"GET", "POST", "PUT", "PATCH", "DELETE"}

// next returns the entry after cur in order, wrapping; the first entry
// when cur is not listed.
func next[T comparable](order []T, cur T) T {
	for i, s := range order {
		if s == cur {
			return order[(i+1)%len(order)]
		}
	}
	return order[0]
}

// reqTab is one open request with its editor and response state.
type reqTab struct {
	*tabs.Tab
	url                   widget.TextField
	paneTab               int
	respTab               int
	view                  widget.CodeView              // response body
	bodyView              widget.CodeView              // request body, read-only
	picture               *ebiten.Image                // the response body when it is an image
	pictureInfo           string                       // "PNG 640x480, 12.5KB"
	tests                 widget.TextArea              // the assertions editor
	testResults           []assert.Result              // what the assertions found in the shown response
	gqlVars               widget.TextArea              // the variables of a GraphQL body
	file                  widget.TextField             // the path of a file body
	fileCheck             bodyFileCheck                // what the file system said about file's path
	body                  widget.TextArea              // request body editor
	reqScroll, respScroll widget.ScrollState           // scroll of the table and note panes
	bodyTypesX            widget.ScrollState           // sideways scroll of the body type chips
	reqTabsX, respTabsX   widget.ScrollState           // sideways scroll of the two pane tab strips
	timeout               widget.TextField             // the request's timeout seconds, as typed
	cancel                context.CancelFunc           // stops the running request
	auth                  map[string]*widget.TextField // Auth pane inputs by field id, created on first use
	showSecrets           bool                         // show masked credential fields
	look                  viewer.Model                 // the response mode, the find bar and its active match
	compactAll            []widget.Line                // the unfiltered compact lines, kept while a search filters the view
	find                  widget.TextField             // the search text
	wsCancel              context.CancelFunc           // ends the timeout of the connect in progress
	outbox                []queuedSend                 // sends and pings waiting for the one in flight, in order
	outboxBusy            bool                         // a send is in flight
	wsView                widget.CodeView              // the message log
	msg                   widget.TextField             // the message being composed
	grpcStream            bool                         // a gRPC tab streams instead of making single calls
	proto                 protoState                   // the Proto pane of a gRPC tab
	messageSchema         messageSchemaState           // the Schema pane of a gRPC tab
	schema                *model.Schema                // what a GraphQL introspection answered
	explorer              mcpexplorer.State            // the MCP explorer: the list shown, the open entry, its answer
	form                  *mcpform.Form                // the argument form of the open entry, nil for none
	resultView            widget.CodeView              // the last answer of the explorer
	listScroll            widget.ScrollState           // scroll of the explorer's list
	detailScroll          widget.ScrollState           // scroll of the open entry
}

// makeTab is newTab with the response view the settings ask for.
func (a *App) makeTab(tab *tabs.Tab) *reqTab {
	t := newReqTab(tab)
	t.look = viewer.New(a.settings.ResponseView)
	t.Earlier = a.loadHistory(tab.Req)
	return t
}

// updateTabs applies msg to the tab list, then does what the change means
// for the window: an inline edit in a closed tab ends, closed tabs lose
// their widgets and new ones get theirs.
func (a *App) updateTabs(msg tabs.Msg) {
	next := tabs.Update(a.list, msg)
	for tab, widgets := range a.widgets {
		if slices.Contains(next.Tabs, tab) {
			continue
		}
		if a.edit != nil && a.edit.tab == widgets {
			a.cancelEdit()
		}
		a.cancelSend(widgets) // a closed tab keeps neither a request nor a socket
		a.disconnect(widgets)
		delete(a.widgets, tab)
	}
	for _, tab := range next.Tabs {
		if a.widgets[tab] == nil {
			a.widgets[tab] = a.makeTab(tab)
		}
	}
	was := a.activeRequest()
	a.list = next
	if now := a.activeRequest(); now != nil && now != was {
		a.revealRequest(now)
	}
}

// activeRequest is the request of the active tab, nil when none is open.
func (a *App) activeRequest() *model.Request {
	if a.list.Active < 0 || a.list.Active >= len(a.list.Tabs) {
		return nil
	}
	return a.list.Tabs[a.list.Active].Req
}

// openTabs are the widget states of the open tabs, in the order of the strip.
func (a *App) openTabs() []*reqTab {
	open := make([]*reqTab, len(a.list.Tabs))
	for i, tab := range a.list.Tabs {
		open[i] = a.widgets[tab]
	}
	return open
}

func newTab(req *model.Request) *reqTab { return newReqTab(tabs.New(req)) }

func newReqTab(tab *tabs.Tab) *reqTab {
	req := tab.Req
	t := &reqTab{Tab: tab, url: widget.NewTextField(req.URL)}
	t.look.Match = -1
	t.wsView.Follow = true
	t.wsCancel = func() {}
	t.explorer = mcpexplorer.New()
	t.timeout = widget.NewTextField(timeoutText(req.TimeoutSec))
	t.body = widget.NewTextArea(req.Body)
	t.tests = widget.NewTextArea(req.Tests)
	t.file = widget.NewTextField(req.Body)
	t.gqlVars = widget.NewTextArea(req.GraphQLVars)
	return t
}

type sendResult struct {
	tab  *reqTab
	seq  int // which send of the tab this answers
	resp model.Response
}

// App is the whole window. It implements ebiten.Game.
type App struct {
	ctx         *core.Ctx
	retryDelay  time.Duration // wait before a dropped connection reopens
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

// NewApp builds the window state around the sample collection.
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

func (a *App) cur() *reqTab { return a.widgets[a.list.Tabs[a.list.Active]] }

// openRequest focuses the tab of req, opening one if needed.
func (a *App) openRequest(req *model.Request) {
	a.selected = model.Of(req)
	a.updateTabs(tabs.Open{Req: req})
}

// send runs t's request off the UI goroutine; the result comes back
// through the bus and is applied in drain.
func (a *App) send(tab *reqTab) {
	if isSocket(tab) {
		a.toggleConnection(tab)
		return
	}
	a.dispatch(tab, tab.Req)
}

// dispatch sends req on behalf of tab t: the request of the tab, or a
// helper one such as a GraphQL schema query that uses the tab's URL and
// credentials.
func (a *App) dispatch(tab *reqTab, source *model.Request) {
	if !a.allowed(tab, source, func() { a.dispatch(tab, source) }) {
		return
	}
	seq, started := tab.BeginSend()
	if !started {
		return
	}
	a.status = "Sending " + source.URL
	req := source.Expanded(a.varsFor(tab.Req))
	a.calls.Sent(a.now(), req)
	ctx, cancel := context.WithTimeout(context.Background(), a.timeoutFor(req))
	tab.cancel = cancel
	reg := a.reg // the command must not read App fields the UI goroutine replaces
	a.bus.Run(ctx, func(ctx context.Context) bus.Msg {
		defer cancel()
		resp, err := reg.Send(ctx, req)
		if err != nil {
			resp = model.Response{Err: err.Error()}
		}
		return sendResult{tab: tab, seq: seq, resp: resp}
	})
}

// cancelSend stops the running request of t. Its late result, which
// arrives after this, belongs to an older send and is dropped.
func (a *App) cancelSend(tab *reqTab) {
	if !tab.Cancel() {
		return
	}
	tab.cancel()
	tab.view.SetLines(nil)
	a.status = model.ErrCancelled
}

// timeoutFor is how long a request may run: its own setting, else the
// default from the settings, else a minute.
func (a *App) timeoutFor(req model.Request) time.Duration {
	sec := req.TimeoutSec
	if sec <= 0 {
		sec = a.settings.TimeoutSec
	}
	if sec <= 0 {
		sec = defaultTimeoutSec
	}
	return time.Duration(min(sec, int(maxTimeout/time.Second))) * time.Second // before the multiply, which overflows
}

// timeoutText is a timeout as shown in its field; 0 (the default) is blank.
func timeoutText(sec int) string {
	if sec <= 0 {
		return ""
	}
	return strconv.Itoa(sec)
}

// drain applies what other goroutines reported and reports whether there was
// anything.
func (a *App) drain() bool { return a.bus.Drain(a.handle) }

// handle applies one message from the bus.
func (a *App) handle(msg bus.Msg) {
	switch msg := msg.(type) {
	case sendResult:
		a.apply(msg)
	case wsEvent:
		a.applyWS(msg)
	case func(): // something a goroutine wants done on the UI goroutine, such as a chosen file
		msg()
	case bus.Failed: // a background task panicked; the app lives on
		a.status = msg.Err.Error()
	}
}

func (a *App) apply(sendResult sendResult) {
	t := sendResult.tab
	if !t.Accept(sendResult.seq) {
		return // the answer to a send that was cancelled or replaced
	}
	t.Remember(t.Resp, a.now())
	t.Resp = &sendResult.resp
	a.saveHistory(t)
	if path := a.requestPath(t.Req); path != "" {
		owner, _ := a.open.Owner(model.Of(t.Req))
		a.sent.Record(a, a.now(), owner.Dir, path, t.Req, sendResult.resp.Status)
	}
	a.calls.Answer(a.now(), t.Req, &sendResult.resp)
	a.rememberSchema(t, &sendResult.resp)
	a.refreshView(t)
	if sendResult.resp.Err != "" {
		a.status = "Request failed"
		a.runTests(t)
		return
	}
	a.status = fmt.Sprintf("%s %s: %d %s in %s", t.Req.Method, t.Req.URL, sendResult.resp.Status, sendResult.resp.StatusText, textfmt.Duration(sendResult.resp.Duration))
	a.runTests(t)
}

// cursorShapes maps the cursor widgets ask for to the window system's.
var cursorShapes = map[core.Cursor]ebiten.CursorShapeType{
	core.CursorDefault:    ebiten.CursorShapeDefault,
	core.CursorResizeH:    ebiten.CursorShapeEWResize,
	core.CursorResizeV:    ebiten.CursorShapeNSResize,
	core.CursorMove:       ebiten.CursorShapeMove,
	core.CursorResizeNWSE: ebiten.CursorShapeNWSEResize,
	core.CursorResizeNESW: ebiten.CursorShapeNESWResize,
}

// Layout implements ebiten.Game. The UI is laid out in logical pixels
// (w by h); the screen is that size times the monitor's scale factor so
// text and lines stay sharp on HiDPI displays.
func (a *App) Layout(width, height int) (int, int) {
	reported := ebiten.Monitor().DeviceScaleFactor()
	f := frameSize(width, height, reported, a.displayScale(reported), a.settings.UIScale())
	a.windowScale = f.WindowScale
	a.setSize(f.LW, f.LH, f.Scale)
	return f.SW, f.SH
}

// Update implements ebiten.Game: it samples input and runs one frame.
func (a *App) Update() error {
	a.winX, a.winY = ebiten.WindowPosition()
	a.step(a.sampler.Read().Scaled(a.ctx.Scale))
	if title := a.Title(); title != a.title {
		a.title = title
		ebiten.SetWindowTitle(title)
	}
	ebiten.SetCursorShape(cursorShapes[a.ctx.Cursor])
	if tps := a.pacer.TPS(); tps != a.tps {
		ebiten.SetTPS(tps)
		a.tps = tps
	}
	return a.shotUpdate()
}

// step runs one frame with the given input: it applies finished sends,
// builds the view, lays it out and lets it react. Handlers in the view
// change App state. Tests drive the app through step without a window.
func (a *App) step(in core.Input) {
	a.ctx.Tick++
	a.ctx.In = in
	a.ctx.Cursor = core.CursorDefault
	if !in.Held && !in.MiddleHeld {
		a.updateTabs(tabs.DragEnd{})
		a.dragRow = nil
	}
	drained := a.drain()
	pruned := a.pruneToasts()
	active := engine.InputActive(a.lastIn, in)
	a.noteGeometry()
	if active || drained || pruned {
		a.markStale()
	}
	if in.RestartsCaret() {
		a.ctx.RestartCaret()
	}
	a.lastIn = in
	if in.Held {
		a.dropTarget = model.Node{} // set again by the row under the pointer
	} else {
		a.finishDrag()
	}
	hadOverlay := a.overlayOpen()
	wasRecording := a.shortcutKeys.Recording() != ""
	menuWasOpen := widget.MenuOpen(a.ctx)
	a.overlayKeys(in)
	if menuWasOpen {
		a.swallowMenuKeys()
	}
	if !hadOverlay && !a.overlayOpen() && !wasRecording {
		a.treeKeys(in)
	}
	if wasRecording {
		a.shortcutKeys.RecordKey(a, in)
	} else if !a.overlayOpen() {
		a.runShortcuts(in)
	}
	if in.Escape && !wasRecording { // one step per press: overlays, the settings menu, a cell edit, find, a running request
		a.escape()
	}
	if (in.Undo || in.Redo) && !a.overlayOpen() && !a.textFocused() {
		if p, t, ok := a.visibleTable(); ok {
			if in.Undo {
				a.undoRows(p, t)
			} else {
				a.redoRows(p, t)
			}
		}
	}
	root := a.view()
	view.Layout(a.ctx, root, a.w, a.h)
	view.Update(a.ctx, root)
	if a.edit != nil && !a.edit.field.Focused { // a click elsewhere ends the edit
		a.commitEdit()
	}
	if a.ctx.Tick%settingsui.PollTicks == 0 {
		a.prefs.Poll(a)
	}
	a.dragHover()
	a.showGrab()
	a.settleBulk()
	a.stateTick()
	a.workspaceTick()
	a.markFrame(active || drained || pruned || a.tips.Waiting())
	a.pacer.NoteIdle(active || drained || pruned || a.tips.Waiting())
}

// Draw implements ebiten.Game. The view is rebuilt so it shows the
// state the handlers just changed.
func (a *App) Draw(screen *ebiten.Image) {
	if !a.dirty && !a.alwaysDraw() {
		return // nothing changed: the screen keeps what it has and the GPU stays idle
	}
	root := a.view()
	view.Layout(a.ctx, root, a.w, a.h)
	view.Draw(a.ctx, root, screen)
	a.dirty = false
	a.shotDraw(screen)
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

// protocolsFor are the protocols the chip cycles through for a request on
// current: the experimental ones only when settings switch them on, except
// the one the request already uses, so it never loses its protocol.
func (a *App) protocolsFor(current model.Protocol) []model.Protocol {
	var offered []model.Protocol
	for _, view := range protocolViews {
		if view.offered == nil || view.offered(a.settings) || view.protocol == current {
			offered = append(offered, view.protocol)
		}
	}
	return offered
}

func (a *App) cycleProtocol(tab *reqTab) {
	a.disconnect(tab) // a connection belongs to the protocol that opened it
	editRequest(tab, editor.ProtocolChosen{Protocol: next(a.protocolsFor(tab.Req.Protocol), tab.Req.Protocol)})
	tab.paneTab, tab.respTab = 0, 0 // the indexes point into the panes of the protocol
}

func (a *App) cycleMethod(tab *reqTab) {
	editRequest(tab, editor.MethodChosen{Method: next(methodOrder, tab.Req.Method)})
}

// urlEdited keeps the request and its Params table in step with the URL field.
func (a *App) urlEdited(tab *reqTab) { editRequest(tab, editor.URLTyped{Text: tab.url.Text}) }

// editRequest applies an edit to the request of t, which is the collection's
// own, so the sidebar and the autosave see it at once.
func editRequest(tab *reqTab, msg editor.Msg) { *tab.Req = editor.Update(*tab.Req, msg) }

// dirtyMark is shown on tabs whose request changed since it was opened.
const dirtyMark = "\u2022"

// dirty reports whether the request differs from its state when opened.
func dirty(tab *reqTab) bool { return tab.Dirty() }
