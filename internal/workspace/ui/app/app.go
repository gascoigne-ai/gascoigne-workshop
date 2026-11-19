package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/controller/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/model/collection"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection/ui/destpicker"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	reqport "github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/ui/calltrace"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
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
	ports
	treeState
	tableState
	frameState
	persistState
	settingsState

	ctx        *core.Ctx
	retryDelay time.Duration // wait before a dropped connection reopens
	baseScale  float64       // the platform's interface factor under the user's scale
	reg        reqport.Registry
	open       collections.Set         // the open collections, in tree order, each with its folder and environment
	sample     func() model.Collection // the collection shown when nothing is saved or configured
	list       tabs.List               // the open tabs, the active one and the one being dragged
	widgets    map[*tabs.Tab]*reqTab   // the widget state of each open tab
	bus        *bus.Bus                // what goroutines report back, drained once per frame
	status     string
	shot       screenshot
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
