// Package window is the root composite of the screen: the Component that owns
// the frame's state, builds the screen areas (sidebar, header, tab strip,
// editor, response, status bar and overlays) from the stores it is given,
// routes the keys of a frame to them in order, and lays their views into one
// tree every frame. The app wires the stores and runs the frame loop around it.
//
// Only the UI goroutine touches Component state. A goroutine reads only
// fields that never change after New (the bus, the ports) and reports back
// through the bus, which the frame drains in beginFrame.
package window

import (
	"cmp"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"path/filepath"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/sends"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/autosave"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/notice"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/serial"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/storage"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/assertions"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/auth"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/body"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/graphql"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/grpc"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/mcp"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/urlbar"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/ws"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/header"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/header/envchip"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog/formpage/fontlist"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/response"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sidebar"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/statusbar"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/tabstrip"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

const version = "v0.1.0-alpha"

// Component is the whole window: the state of a frame, the settings in
// effect, what is remembered between runs, and the screen areas it hosts.
type Component struct {
	ports

	// What one frame reads and decides.
	w, h           int
	dirty          bool         // something changed since the last draw
	drawEveryFrame bool         // every frame is built and drawn, as a capture needs (Start.DrawEveryFrame)
	drawTree       *El          // the tree the last step built and updated; Draw paints it
	treeStale      bool         // a handler or a step after the build changed what drawTree shows
	builds         int          // view trees built, so tests can count them
	lastIn         core.Input   // the previous frame's input
	pacer          engine.Pacer // update rate from idle time
	windowScale    float64      // window units per logical pixel, set by Resize
	winX, winY     int          // window position, read each frame
	tips           TooltipState
	sidebarSplit   SplitState // width of the collections sidebar
	paneSplit      SplitState // request pane share of the request/response area

	// What is remembered between runs and written in the background.
	stateFile        *autosave.StateFile   // the remembered state: window, trusted accesses, welcomed version
	saves            *autosave.Collections // when each open collection was last written, and its writing
	writes           *serial.Writer        // the background writes, one per file at a time
	keys             *autosave.Keys        // the comparable forms of the collection and the state, recomputed only on frames where something could have changed
	defaultWorkspace string                // beside it is the collections folder, where a first save goes

	settings settings.Settings // the settings in effect; the dialog that edits them (over.Settings) owns the file, its path and what is unsaved

	ctx        *core.Ctx
	editor     *editor.Component   // the request pane's components: URL bar, body, auth, tests, messages, GraphQL, gRPC, MCP and the tables
	resp       *response.Component // the response panes: the body, its find bar, the headers, the timeline and the status pill
	strip      *tabstrip.Component // the strip of open requests above the request pane
	header     *header.Component   // the banner over the main column: the title, the environment chip and the settings button
	retryDelay time.Duration       // wait before a dropped connection reopens
	reg        request.Registry
	cols       *collections.Library      // the open collections, in tree order, each with its folder and environment, and what is done to whole ones
	sample     func() request.Collection // the collection shown when nothing is saved or configured
	sends      *sends.Sends              // the requests in flight, by the request of their tab
	tabs       *tabs.Set                 // the open tabs, the current one and the drag
	widgets    map[*tabs.Tab]*reqTab     // the widget state of each open tab
	bus        *bus.Bus                  // what goroutines report back, drained once per frame
	notices    notice.Notices            // the status line under the window
	bar        *statusbar.Component
	side       *sidebar.Component
	driver     Driver              // what the end-to-end screenshots drive the window with
	err        error               // a failure that ends the run, reported by Err
	over       *overlays.Component // everything that floats over the window: the dialog, menus, toasts, file browser, call trace and settings
	varsTarget varsTarget          // what the Vars pane edits; the zero value is the active environment's or the collection's
	jar        http.CookieJar      // cookies the server set, kept until the window closes
}

// New builds the window from stores: settings, user themes, the screen areas
// and the start collection. A port the caller leaves out is replaced by a null
// one, so no code below checks it.
func New(stores Stores) (*Component, error) {
	stores = withDefaults(stores)
	start := stores.Start
	st, loadErr, themeProblems := loadSettings(stores)
	ctx, err := core.NewCtx(theme.ThemeWith(st))
	if err != nil {
		return nil, err
	}
	a := &Component{
		ctx:              ctx,
		retryDelay:       start.RetryDelay,
		sample:           start.Sample,
		widgets:          map[*tabs.Tab]*reqTab{},
		bus:              stores.Bus,
		jar:              newJar(),
		tabs:             stores.Tabs,
		sends:            stores.Sends,
		ports:            portsOf(stores.Ports),
		dirty:            true,
		drawEveryFrame:   start.DrawEveryFrame,
		defaultWorkspace: start.DefaultWorkspace,
		settings:         st,
	}
	a.writes = serial.NewWriter(a.bus, a.drain)
	a.keys = autosave.NewKeys(keysHost{a})
	a.saves = autosave.NewCollections(a.store, a.writes)
	a.stateFile = autosave.NewStateFile(a.states, start.StatePath, stateHost{a}, stateHost{a}, a.writes)
	a.cols = collections.NewLibrary(a.store, collectionsDir(start.DefaultWorkspace), collections.Deps{Settings: librarySettings{a}, Saves: librarySaves{a}})
	a.driver.app = a
	a.notices.SetStatus("Ready")
	settingsAccess := storage.AccessAfter(loadErr)
	over := overlaysHost{a}
	a.over = overlays.New(overlays.Deps{
		Ctx: ctx, Window: over, Clock: over, Desktop: over, Folders: over, Bus: over,
		Settings: settingsdialog.Deps{
			Settings: settingsHost{a}, Window: settingsHost{a}, Files: settingsHost{a}, Dialogs: settingsHost{a},
			Bus: settingsHost{a}, Looks: settingsHost{a}, Keys: settingsHost{a},
			Path: start.SettingsPath, Store: a.settingFiles, Access: settingsAccess, Fonts: a.fonts, System: a.system,
		},
	})
	a.side = sidebar.New(sidebar.Deps{
		Collections: sidebarHost{a}, Requests: sidebarHost{a}, Nodes: sidebarHost{a}, Drops: sidebarHost{a},
		Folders: sidebarHost{a}, Extras: sidebarHost{a}, Collection: collectionHost{a}, Workspace: sidebarHost{a},
		Menus: sidebarHost{a}, Sent: a,
	})
	host := editorHost{a}
	a.editor = editor.New(editor.Deps{
		Ctx: ctx, Tabs: host, Connections: host, Settings: host, VarsPane: host, Window: bodyHost{a},
		URLBar:     urlbar.Deps{Tabs: urlbarHost{a}, Sends: urlbarHost{a}, Connections: urlbarHost{a}, GRPC: urlbarHost{a}, Styles: urlbarHost{a}, Hints: urlbarHost{a}},
		Body:       body.Deps{Tabs: bodyHost{a}, Vars: bodyHost{a}, Files: bodyHost{a}, Schemas: bodyHost{a}},
		GraphQL:    graphql.Deps{Tabs: graphqlHost{a}, Sends: graphqlHost{a}, Window: graphqlHost{a}},
		Auth:       auth.Deps{Tabs: authHost{a}, Vars: authHost{a}, Trust: authHost{a}, Cookies: authHost{a}, Window: authHost{a}},
		Assertions: assertions.Deps{Tabs: assertionsHost{a}, Vars: assertionsHost{a}, Status: assertionsHost{a}, Defaults: assertionsHost{a}},
		GRPC:       grpc.Deps{Tabs: grpcHost{a}, Methods: grpcHost{a}, Vars: grpcHost{a}, Files: grpcHost{a}, Connections: grpcHost{a}, Window: grpcHost{a}},
		MCP:        mcp.Deps{Tabs: mcpHost{a}, Bus: mcpHost{a}, Composer: mcpHost{a}, Dialogs: mcpHost{a}, Looks: mcpHost{a}},
		WS:         ws.Deps{Tabs: wsHost{a}, Outbox: wsHost{a}, Tools: wsHost{a}, Window: wsHost{a}},
		Tables:     kvtable.Deps{Menus: kvtableHost{a}, Secrets: kvtableHost{a}, Styles: kvtableHost{a}},
	})
	a.resp = response.New(response.Deps{Tabs: responseHost{a}, Slots: responseHost{a}, Headers: responseHost{a}, Files: responseHost{a}, Exports: responseHost{a}, Window: responseHost{a}, Hints: responseHost{a}, Defaults: responseHost{a}})
	a.strip = tabstrip.New(tabstrip.Deps{Tabs: tabstripHost{a}, Marks: tabstripHost{a}, Menus: tabstripHost{a}, Requests: tabstripHost{a}, Hints: tabstripHost{a}})
	a.header = header.New(header.Deps{Title: headerHost{a}, Settings: headerHost{a}, Hints: headerHost{a}, Env: envchip.Deps{Envs: envchipHost{a}, Menus: envchipHost{a}, Dialogs: envchipHost{a}}})
	a.bar = statusbar.New(statusbar.Deps{Status: &a.notices, Log: a.over.Calls(), Welcome: a, Version: version})
	if loadErr != nil {
		a.notices.SetStatus(loadErr.Error())
	}
	a.reg = a.newRegistry()
	ctx.Clipboard = core.ReportClipboard(ctx.Clipboard, a.clipboardNote)
	if start.HostOS != "" {
		ctx.HostOS = start.HostOS
	}
	ctx.Now = func() time.Time { return a.now() } // the caret blinks by the window clock
	a.noteReadOnly(settingsAccess, loadErr)
	if start.ScanFonts {
		a.over.Settings.ScanFonts()
	}
	a.openWorkspace(start.WorkspaceDir)
	stateErr := a.stateFile.Load()
	if start.Welcome && (start.WelcomeEvery || a.stateFile.Welcomed() != version) {
		a.OpenWelcome()
	}
	a.side.Sent().Load(a)
	a.notices.SetStatus(stateNote(a.notices.Status(), stateErr))
	if len(themeProblems) > 0 {
		a.notices.SetStatus(stateNote(a.notices.Status(), fmt.Errorf("%s", themeProblems[0])))
	}
	a.stateFile.Begin()
	return a, nil
}

// startCollection is the collection shown before a workspace or client
// config replaces it.
func startCollection(sample func() request.Collection) request.Collection {
	if sample == nil {
		return collection.GitHubSample()
	}
	return sample()
}

// withDefaults fills what a caller may leave out: null ports for the adapters
// the window always calls, the clock and the retry delay.
func withDefaults(stores Stores) Stores {
	ports := &stores.Ports
	ports.SettingsFiles = cmp.Or[settings.SettingsStore](ports.SettingsFiles, noSettingFiles{})
	ports.States = cmp.Or[workspace.StateStore](ports.States, noStates{})
	ports.Store = cmp.Or[collection.Store](ports.Store, noStore{})
	ports.Fonts = cmp.Or[fontlist.FontStore](ports.Fonts, noFonts{})
	ports.Files = cmp.Or[FileDialog](ports.Files, noFileDialog{})
	ports.Folders = cmp.Or[workspace.Folders](ports.Folders, noFolders{})
	ports.Previews = cmp.Or[send.BodyPreview](ports.Previews, noBrowser{})
	if ports.Now == nil {
		ports.Now = time.Now
	}
	stores.Start.RetryDelay = cmp.Or(stores.Start.RetryDelay, defaultRetryDelay)
	return stores
}

// portsOf is Ports as the window keeps it.
func portsOf(given Ports) ports {
	return ports{
		protocols: given.Protocols, grpcMethods: given.GRPCMethods, grpcSchema: given.GRPCSchema, curl: given.Curl,
		store: given.Store, states: given.States, clients: given.Clients, files: given.Files, folders: given.Folders,
		now: given.Now, histories: given.History, fonts: given.Fonts, system: given.System,
		settingFiles: given.SettingsFiles, previews: given.Previews,
	}
}

// loadSettings reads the settings file named by stores, the defaults when it
// names none, and the user themes beside it.
func loadSettings(stores Stores) (settings.Settings, error, []string) {
	st, loadErr := settings.Default(), error(nil)
	if stores.Start.SettingsPath != "" {
		st, loadErr = stores.Ports.SettingsFiles.Load(stores.Start.SettingsPath)
	}
	themes, themeProblems := theme.LoadUserThemes(stores.Start.ThemesDir)
	theme.SetUserThemes(themes)
	return st, loadErr, themeProblems
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

// stateNote joins a state load problem into the status line.
func stateNote(status string, err error) string {
	if err == nil {
		return status
	}
	return strings.TrimSpace(status + " " + err.Error())
}

// noteReadOnly tells once, at the start, that a file a newer build wrote is
// left as it is for this session.
func (a *Component) noteReadOnly(access storage.Access, loadErr error) {
	if access == storage.ReadOnly {
		a.toast(loadErr.Error()+"; it is not changed this session", severity.Error)
	}
}

// newJar is the cookie jar of a session; with no options it cannot fail.
func newJar() http.CookieJar {
	jar, _ := cookiejar.New(nil)
	return jar
}

// newRegistry builds the protocols from the cookie jar, the network
// settings (proxy, certificates) and the window's sampling dialog.
func (a *Component) newRegistry() request.Registry {
	if a.protocols == nil {
		return request.Registry{}
	}
	s := a.settings
	return a.protocols(request.ProtocolConfig{Jar: a.jar, Proxy: s.Proxy, CACert: s.CACert, ClientCert: s.ClientCert, ClientKey: s.ClientKey, Sampler: a.editor.MCP.Sample})
}

// Sizes of the window chrome, in logical pixels.
const (
	sidebarW = 260           // the collections tree's default width
	headerH  = sizes.HeaderH // the banner: title, environment chip, settings
	tabsH    = sizes.TabsH   // the request tab row: a tab's height plus the room above it
	urlH     = sizes.URLBarH // the URL bar: its controls plus padding above and below

	// Smallest sizes the dividers allow.
	minSidebarW = 160
	minMainW    = 420
	minPaneSize = 220
)

// sidebarWidth is the current width of the sidebar in pixels.
func (a *Component) sidebarWidth() int {
	if a.sidebarSplit.Size > 0 {
		return int(a.sidebarSplit.Size)
	}
	return sidebarW - 1
}

var windowTemplate = markup.Must("window.vml", windowSource)

// view is the whole window as a function of the app state (window.vml):
// the collections tree beside the main column (header, request tabs, URL
// bar, request | response) above the status bar, and the overlays over
// them. The popup menu and the tooltip are layers that markup has no
// element for, so they come in as slots like the views built here.
func (a *Component) view() Node {
	t := a.cur()
	scope := markup.Scope{
		"sidebarSplit":   &a.sidebarSplit,
		"sidebarDefault": sidebarW - 1,
		"minSidebarW":    minSidebarW,
		"minMainW":       minMainW,
		"sidebar":        a.side.View(a.ctx, a.h),
		"header":         a.header.View(a.ctx),
		"requestTabs":    a.strip.View(a.ctx),
		"urlBar":         a.editor.URLBar.View(a.ctx),
		"paneSplit":      &a.paneSplit,
		"minPaneSize":    minPaneSize,
		"stacked":        a.settings.Layout == "stacked",
		"requestPane":    a.editor.RequestPane(a.ctx, t.Tab),
		"responsePane":   a.responsePane(t, a.responsePanesFor(t)),
		"statusBar":      a.statusBar(),
	}
	// The overlays are built after the panes, as before, one node per slot of
	// the template; the tooltip layer is last so it is on top.
	over := a.over.Views(a.ctx)
	scope["menu"], scope["picker"], scope["browser"], scope["popupMenu"] = over.Settings, over.Picker, over.Browser, over.Popup
	scope["dialog"], scope["callLog"], scope["toasts"] = over.Dialog, over.CallLog, over.Toasts
	scope["tooltips"] = TooltipLayer(&a.tips)
	return windowTemplate.MustBuild(scope)
}

// ── Main column ────────────────────────────────────────────────────────

var paneTemplate = markup.Must("pane.vml", paneSource)

// responsePane is the response area (pane.vml): a tab strip, with the
// status at its right end, above the body of the active tab. The request
// pane is the editor's twin of it.
func (a *Component) responsePane(tab *reqTab, panes []pane) Node {
	selectTab := func(index int) { tab.respTab, tab.respScroll.Y = index, 0 }
	tab.respTab = min(tab.respTab, len(panes)-1) // a socket has fewer response panes
	return paneTemplate.MustBuild(markup.Scope{
		"tabsX":     &tab.respTabsX,
		"tabs":      a.tabItems(panes, tab),
		"active":    tab.respTab,
		"selectTab": selectTab,
		"extra":     a.resp.StatusView(a.ctx, tab.Tab),
		"body":      a.paneBody(panes[tab.respTab], tab, &tab.respScroll),
	})
}

// Templates, parsed once above.
const (
	windowSource = `<stack>
  <!-- the frame: collections tree | main column, above the status bar -->
  <col bg="frame">
    <split state="sidebarSplit" default="{{sidebarDefault}}" grow="1" min-first="{{minSidebarW}}" min-second="{{minMainW}}">
      <slot name="sidebar"/>
      <!-- the main column: header, request tabs, URL bar, then request | response -->
      <col grow="1">
        <slot name="header"/>
        <slot name="requestTabs"/>
        <slot name="urlBar"/>
        <split state="paneSplit" default="0.5" as-ratio grow="1" min-first="{{minPaneSize}}" min-second="{{minPaneSize}}" vertical="{{stacked}}">
          <slot name="requestPane"/>
          <slot name="responsePane"/>
        </split>
      </col>
    </split>
    <slot name="statusBar"/>
  </col>
  <!-- the overlays, each only while open, in z-order: the tooltip last so it is on top -->
  <slot if="menu" name="menu"/>
  <slot if="picker" name="picker"/>
  <slot if="browser" name="browser"/>
  <slot if="popupMenu" name="popupMenu"/>
  <slot if="dialog" name="dialog"/>
  <slot if="callLog" name="callLog"/>
  <slot if="toasts" name="toasts"/>
  <slot name="tooltips"/>
</stack>
`

	paneSource = `<col grow="1" bg="bg">
  <tab-bar>
    <scroll-tabs state="tabsX" items="tabs" active="{{active}}" on-select="selectTab"/>
    <slot if="extra" name="extra"/>
  </tab-bar>
  <slot name="body"/>
</col>
`
)

// scope is the scope of a template: the sizes of the faces the context draws
// and the fixed widths, then own.
func (a *Component) scope(own markup.Scope) markup.Scope { return sizes.Scope(a.ctx, own) }

// statusBar is the line under the window.
func (a *Component) statusBar() Node { return a.bar.View(a.ctx) }
