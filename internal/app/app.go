// Package app is the window: the App that owns the state, wires the
// adapters it is given to the controllers and builds the view from VML
// templates every frame.
//
// Only the UI goroutine touches App state. A goroutine reads only fields
// that never change after NewApp (the bus, the ports) and reports back
// through the bus, which the frame drains in beginFrame.
package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/ui/destpicker"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/ui/calltrace"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"net/http"
	"time"
)

const (
	version = "v0.1.0-alpha"
)

// App is the whole window. It implements ebiten.Game.
type App struct {
	ports
	treeState
	tableState
	frameState
	persistState
	settingsState

	ctx        *core.Ctx
	retryDelay time.Duration // wait before a dropped connection reopens
	baseScale  float64       // the platform's interface factor under the user's scale
	reg        request.Registry
	open       collections.Set           // the open collections, in tree order, each with its folder and environment
	sample     func() request.Collection // the collection shown when nothing is saved or configured
	list       tabs.List                 // the open tabs, the active one and the one being dragged
	widgets    map[*tabs.Tab]*reqTab     // the widget state of each open tab
	bus        *bus.Bus                  // what goroutines report back, drained once per frame
	status     string
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

func (a *App) clickTree(row collection.Row) {
	a.selectNode(row.Node())
	if row.Folder {
		a.expanded[row.Dir] = !a.expanded[row.Dir]
		return
	}
	a.openRequest(row.Req)
}

func (a *App) newRequest() { a.newRequestIn(a.selectedFolder()) }

// Err is the error that ended the run, if any.
func (a *App) Err() error { return a.err }
