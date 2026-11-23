// Package response draws the four panes of the response area (the body with
// its toolbar and find bar or the picture of an image, the headers table, the
// timeline and the tests) and the status pill, and keeps per open tab the
// state they need: the code view of the body, the picture, the view mode and
// find bar (viewer.Model), the unfiltered compact lines and the find field.
// The strip above the panes, its badges, the message that replaces a pane
// ("Sending...", an error) and which pane shows stay in the application: the
// panes of a socket tab share them.
//
// Panes are a table, paneViews, indexed by Pane; the application's pane rows
// name theirs, so a new pane is a row in both. The Tests pane is the
// assertions component's, reached through Deps.Slots since siblings never
// import each other. The sent log is not here: it belongs to the sidebar, and
// the call trace belongs to the overlays (`ui/overlays/calltrace`).
//
// The state of the tabs is a map of pointers made on first use, with the view
// mode the settings name, and dropped by Tick when its tab closes. Lookup
// happens for the current tab every frame and for the tab a response arrives
// for (which may be hidden), so a map beats a slice in tab order and a single
// slot, which would lose the find text, the wrap choice, the compact cache and
// the picture of the other tabs. Revisit when tabs get stable ids or the find
// text must outlive a tab. compactAll is a derived value kept stored: the
// source is tabs.Tab.Resp, but each keystroke of a search refilters it and
// rebuilding it reformats a body of up to 2 MiB; Refresh rebuilds it, and a
// second writer of Resp that bypasses Refresh would make it stale (the cancel
// of a send replaces Resp and only clears the lines, so a search that runs
// afterwards fills the hidden view with the old lines; the error message
// covers the pane, and the gap is left alone).
//
// The column widths of the headers table are not here: they are the record
// the key-value tables keep under "response Headers" in the state file, and
// reach the pane through Deps.Headers. A record of this component was rejected
// because it would move the owner of persisted data in a refactor; revisit
// with the state of the window (task 044).
//
// typesOnly in keys.go is the third copy of the one in the URL bar and the
// body; moving it to the toolkit waits for the user's approval.
package response

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/viewer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// Pane is one of the four tabs of the response area.
type Pane int

// The panes, in the order of the strip.
const (
	Body Pane = iota
	Headers
	Timeline
	Tests
	PaneCount
)

// Component draws the response panes and keeps the state of the open tabs.
type Component struct {
	deps   Deps
	states map[*tabs.Tab]*state
}

// state is what one tab keeps of its response between frames.
type state struct {
	view        widget.CodeView  // the body as lines
	picture     *ebiten.Image    // the body when it is an image
	pictureInfo string           // "PNG 640x480, 12.5KB"
	look        viewer.Model     // the view mode, the find bar and its active match
	compactAll  []widget.Line    // the unfiltered compact lines, kept while a search filters the view
	find        widget.TextField // the search text
}

// New returns a component asking deps.
func New(deps Deps) *Component {
	return &Component{deps: deps, states: map[*tabs.Tab]*state{}}
}

// state is the state of tab, made on first use with the view mode the
// settings ask for.
func (c *Component) state(tab *tabs.Tab) *state {
	made, ok := c.states[tab]
	if !ok {
		made = &state{look: viewer.New(c.deps.Defaults.ResponseView())}
		c.states[tab] = made
	}
	return made
}

// Look is the view mode, find bar and active match of tab.
func (c *Component) Look(tab *tabs.Tab) viewer.Model { return c.state(tab).look }

// Code is the code view that shows the body of tab.
func (c *Component) Code(tab *tabs.Tab) *widget.CodeView { return &c.state(tab).view }

// Field is the find field of tab.
func (c *Component) Field(tab *tabs.Tab) *widget.TextField { return &c.state(tab).find }
