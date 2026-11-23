// Package app is the window: the App that owns the state, wires the
// adapters it is given to the controllers and builds the view from VML
// templates every frame.
//
// Only the UI goroutine touches App state. A goroutine reads only fields
// that never change after NewApp (the bus, the ports) and reports back
// through the bus, which the frame drains in beginFrame.
package app

import (
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/sends"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings/keymap"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/notice"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/assertions"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/auth"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/body"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/graphql"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/grpc"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/mcp"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/urlbar"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/ws"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/destpicker"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/overlays/settingsdialog/shortcutpage"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/response"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/response/calltrace"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sidebar"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/statusbar"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/tabstrip"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/engine"
	"net/http"
	"time"
)

const (
	version = "v0.1.0-alpha"
)

// App is the whole window. It implements ebiten.Game.
type App struct {
	ports
	frameState
	persistState
	settingsState

	ctx        *core.Ctx
	kv         *kvtable.Component    // the key-value tables of the request pane
	resp       *response.Component   // the response panes: the body, its find bar, the headers, the timeline and the status pill
	assertions *assertions.Component // the Tests and Settings panes, the response Tests pane and the assertion results of each tab
	ws         *ws.Component         // the Messages pane: the log, its toolbar and the composer of each socket tab
	graphql    *graphql.Component    // the variables and schema buttons of a GraphQL body, per tab
	grpc       *grpc.Component       // the gRPC method picker, the Stream switch and the Proto pane of each tab
	mcp        *mcp.Component        // the MCP explorer, the tool call form and the sampling dialog of each tab
	auth       *auth.Component       // the Auth pane, the trust prompt and the cookie jar view
	body       *body.Component       // the Body and Schema panes: text areas, file paths and the gRPC message schema of each tab
	urlbar     *urlbar.Component     // the bar above the request pane: URL fields, chips and the Send button
	strip      *tabstrip.Component   // the strip of open requests above the request pane
	retryDelay time.Duration         // wait before a dropped connection reopens
	baseScale  float64               // the platform's interface factor under the user's scale
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
	driver     Driver             // what the end-to-end screenshots drive the window with
	err        error              // a failure that ends the run, reported by Err
	browser    *fileBrowser       // the in-app file browser while it is open
	pickerHost destpicker.Host    // what the open picker asks of the App
	picker     *destpicker.Picker // the move or copy destination picker while it is open
	calls      calltrace.Log      // the call trace behind the Log button
	dialog     *dialog            // the open dialog, if any
	varsTarget varsTarget         // what the Vars pane edits; the zero value is the active environment's or the collection's
	jar        http.CookieJar     // cookies the server set, kept until the app closes
}

func (a *App) newRequest() { a.newRequestIn(a.selectedFolder()) }

// commands are the shortcut actions in the order the Shortcuts page lists
// them: each component offers its own, and the App offers the ones that
// reach across components (they read a table cell edit, the text focus or the
// focused collection). A command that moves to a component keeps its place.
func (a *App) commands() []keymap.Command {
	ctrl := func(key ebiten.Key) keymap.Chord { return keymap.Chord{Key: int(key), Ctrl: true} }
	return slices.Concat(
		[]keymap.Command{
			{ID: "send", Name: "Send request", Default: ctrl(ebiten.KeyEnter), Run: a.sendCurrent},
			{ID: "focus-url", Name: "Focus URL", Default: ctrl(ebiten.KeyL), Run: a.focusURL},
		},
		a.strip.Commands(),
		a.kv.Commands(),
		[]keymap.Command{
			{ID: "delete-row", Name: "Delete row", Default: keymap.Chord{Key: int(ebiten.KeyDelete)}, Run: a.deleteRow},
			{ID: "complete-variable", Name: "Complete variable", Default: ctrl(ebiten.KeySpace), Run: a.completeVariable},
			{ID: "save", Name: "Save collection", Default: ctrl(ebiten.KeyS), Run: a.saveWorkspace},
		},
		a.side.Commands(),
		a.resp.Commands(),
		a.prefs.Commands(),
	)
}

// keymap is the binding of every command under the settings in effect. It is
// derived where it is read, never kept, so a changed setting needs no
// invalidation.
func (a *App) keymap() keymap.Keymap {
	return keymap.Resolve(a.settings.Shortcuts, a.commands(), shortcutpage.ParseChord)
}

// runKeymap runs the commands whose chord went down this frame.
func (a *App) runKeymap(in core.Input) {
	keys := a.keymap()
	for _, key := range in.Keys {
		binding := engine.Binding{Key: key, Ctrl: in.Ctrl, Shift: in.Shift, Alt: in.Alt, Bound: true}
		chord, _ := shortcutpage.ChordOf(binding)
		keys.Run(keymap.Press{Chord: chord, TypesText: binding.TypesText()}, a.textFocused())
	}
}

// sendCurrent sends the request; while a table cell is edited, Ctrl+Enter
// adds a row below instead.
func (a *App) sendCurrent() {
	if a.kv.Editing() {
		a.kv.InsertRowBelow()
		return
	}
	a.urlbar.Send()
}

// focusURL puts the keyboard in the URL field with its text selected.
func (a *App) focusURL() {
	a.commitEdit()
	a.urlbar.FocusURL()
}

// Err is the error that ended the run, if any.
func (a *App) Err() error { return a.err }
