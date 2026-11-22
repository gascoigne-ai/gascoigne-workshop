// Package sentlog is the workspace log of sent requests: every finished
// send is remembered (and written to the state folder), and the sidebar can
// list them newest first, filtered by text, to open the request again.
package sentlog

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/severity"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Host is what the log needs from the application.
type Host interface {
	// SentStore keeps the log between runs, nil when nothing is saved.
	SentStore() response.SentStore
	// Toast shows a message over the window.
	Toast(text string, level severity.Level)
	// SetStatus replaces the status line.
	SetStatus(text string)
	// OpenRequest focuses the request at path of the collection saved in
	// folder ("" for an unsaved one or a row older than the folder) and
	// reports whether there is one.
	OpenRequest(folder, path string) bool
	// Badge is the small method label shown before a request's name.
	Badge(protocol request.Protocol, method string) Node
}

// Log holds the sent requests, newest first, and the scroll state of their
// list. The zero value is an empty log.
type Log struct {
	rows   []response.Sent
	scroll ScrollState
}

// Record adds a finished send of the request at path of the collection
// saved in folder ("" when not saved), writes it to the state
// folder, and drops the oldest past response.SentLimit.
func (l *Log) Record(host Host, at time.Time, folder, path string, req *request.Request, status int) {
	sent := response.Sent{At: at, Collection: folder, Path: path, Protocol: req.Protocol, Method: req.Method, URL: req.URL, Status: status}
	l.rows = append([]response.Sent{sent}, l.rows...)
	if len(l.rows) > response.SentLimit {
		l.rows = l.rows[:response.SentLimit]
	}
	store := host.SentStore()
	if store == nil {
		return
	}
	if err := store.Append(sent); err != nil {
		host.Toast("History is not being saved: "+err.Error(), severity.Error)
	}
}

// Load reads the log saved by earlier runs.
func (l *Log) Load(host Host) {
	store := host.SentStore()
	if store == nil {
		return
	}
	rows, err := store.Load()
	if err != nil {
		host.Toast("History could not be read: "+err.Error(), severity.Error)
		return
	}
	l.rows = rows
}

// Rows are the entries whose path, method or URL contains filter, ignoring
// case, newest first; an empty filter keeps all.
func (l *Log) Rows(filter string) []response.Sent {
	needle := strings.ToLower(filter)
	if needle == "" {
		return l.rows
	}
	var kept []response.Sent
	for _, sent := range l.rows {
		if strings.Contains(strings.ToLower(sent.Path+" "+sent.Method+" "+sent.URL), needle) {
			kept = append(kept, sent)
		}
	}
	return kept
}

// Open opens the request an entry was sent from; one that has since been
// deleted or renamed is reported in the status line.
func (l *Log) Open(host Host, sent response.Sent) {
	if !host.OpenRequest(sent.Collection, sent.Path) {
		host.SetStatus(fmt.Sprintf("%s is no longer in the collection", sent.Path))
	}
}

var (
	listTemplate  = markup.Must("list.vml", listSource)
	emptyTemplate = markup.Must("empty.vml", emptySource)
	rowTemplate   = markup.Must("row.vml", rowSource)
)

// scope is the sizes of the faces the context draws with plus the view's own
// entries, so a template holds no size as a number.
func scope(sizes w.Sizes, own markup.Scope) markup.Scope {
	merged := sizes.Scope()
	for name, value := range own {
		merged[name] = value
	}
	return merged
}

// List is the sidebar body while it shows the log instead of the tree:
// the rows, or a note while none match.
func (l *Log) List(ctx *w.Ctx, host Host, filter string) Node {
	sent := l.Rows(filter)
	if len(sent) == 0 { // a template has one root, so the note is its own
		return emptyTemplate.MustBuild(scope(ctx.Sizes, nil))
	}
	rows := make([]Node, 0, len(sent))
	for _, entry := range sent {
		rows = append(rows, l.row(ctx.Sizes, host, entry))
	}
	return listTemplate.MustBuild(scope(ctx.Sizes, markup.Scope{"scroll": &l.scroll, "rows": rows}))
}

// sentRow is what the row template shows of one sent request.
type sentRow struct {
	Badge  Node
	Path   string
	Status string // the status code, or "failed"
	When   string // the URL and the time of day it was sent
	Open   func()
}

// row is one sent request: badge, path and status above, URL and time below.
func (l *Log) row(sizes w.Sizes, host Host, sent response.Sent) Node {
	return rowTemplate.MustBuild(scope(sizes, markup.Scope{
		"row": sentRow{
			Badge:  host.Badge(sent.Protocol, sent.Method),
			Path:   sent.Path,
			Status: status(sent.Status),
			When:   sent.URL + "  " + sent.At.Format("15:04:05"),
			Open:   func() { l.Open(host, sent) },
		},
	}))
}

func status(code int) string {
	if code == 0 {
		return "failed"
	}
	return fmt.Sprint(code)
}

// Templates, parsed once above.
const (
	listSource = `<scroll state="scroll" bg="sunken">
  <slot for="row in rows" name="row"/>
</scroll>
`

	emptySource = `<col bg="sunken" pad="{{gutter}}">
  <text muted>Nothing sent yet</text>
</col>
`

	rowSource = `<col pad-x="{{gutter}}" pad-y="{{nearGap}}" hover-bg="hover" control="row" on-click="row.Open">
  <row gap="{{tightGap}}" align-items="baseline">
    <slot name="row.Badge"/>
    <text ellipsis grow="1">{{row.Path}}</text>
    <text small muted>{{row.Status}}</text>
  </row>
  <row><text small muted ellipsis grow="1">{{row.When}}</text></row>
</col>
`
)
