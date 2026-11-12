// Package sentlog is the workspace log of sent requests: every finished
// send is remembered (and written to the state folder), and the sidebar can
// list them newest first, filtered by text, to open the request again.
package sentlog

import (
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/port"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Host is what the log needs from the application.
type Host interface {
	// SentStore keeps the log between runs, nil when nothing is saved.
	SentStore() port.SentStore
	// Toast shows a message over the window.
	Toast(text string, isErr bool)
	// SetStatus replaces the status line.
	SetStatus(text string)
	// OpenRequest focuses the request at path of the collection and reports
	// whether there is one.
	OpenRequest(path string) bool
	// Badge is the small method label shown before a request's name.
	Badge(protocol, method string) Node
}

// Log holds the sent requests, newest first, and the scroll state of their
// list. The zero value is an empty log.
type Log struct {
	rows   []port.Sent
	scroll ScrollState
}

// Record adds a finished send of the request at path, writes it to the state
// folder, and drops the oldest past port.SentLimit.
func (l *Log) Record(host Host, at time.Time, path string, req *model.Request, status int) {
	sent := port.Sent{At: at, Path: path, Protocol: req.Protocol, Method: req.Method, URL: req.URL, Status: status}
	l.rows = append([]port.Sent{sent}, l.rows...)
	if len(l.rows) > port.SentLimit {
		l.rows = l.rows[:port.SentLimit]
	}
	store := host.SentStore()
	if store == nil {
		return
	}
	if err := store.Append(sent); err != nil {
		host.Toast("History is not being saved: "+err.Error(), true)
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
		host.Toast("History could not be read: "+err.Error(), true)
		return
	}
	l.rows = rows
}

// Rows are the entries whose path, method or URL contains filter, ignoring
// case, newest first; an empty filter keeps all.
func (l *Log) Rows(filter string) []port.Sent {
	needle := strings.ToLower(filter)
	if needle == "" {
		return l.rows
	}
	var kept []port.Sent
	for _, sent := range l.rows {
		if strings.Contains(strings.ToLower(sent.Path+" "+sent.Method+" "+sent.URL), needle) {
			kept = append(kept, sent)
		}
	}
	return kept
}

// Open opens the request an entry was sent from; one that has since been
// deleted or renamed is reported in the status line.
func (l *Log) Open(host Host, sent port.Sent) {
	if !host.OpenRequest(sent.Path) {
		host.SetStatus(fmt.Sprintf("%s is no longer in the collection", sent.Path))
	}
}

//go:embed list.vml empty.vml row.vml
var templates embed.FS

var (
	listTemplate  = markup.MustLoad(templates, "list.vml")
	emptyTemplate = markup.MustLoad(templates, "empty.vml")
	rowTemplate   = markup.MustLoad(templates, "row.vml")
)

// List is the sidebar body while it shows the log instead of the tree:
// the rows, or a note while none match.
func (l *Log) List(host Host, filter string) Node {
	sent := l.Rows(filter)
	if len(sent) == 0 { // a template has one root, so the note is its own
		return emptyTemplate.MustBuild(markup.Scope{"gutter": w.Gutter})
	}
	rows := make([]Node, 0, len(sent))
	for _, entry := range sent {
		rows = append(rows, l.row(host, entry))
	}
	return listTemplate.MustBuild(markup.Scope{"scroll": &l.scroll, "rows": rows})
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
func (l *Log) row(host Host, sent port.Sent) Node {
	return rowTemplate.MustBuild(markup.Scope{
		"row": sentRow{
			Badge:  host.Badge(sent.Protocol, sent.Method),
			Path:   sent.Path,
			Status: status(sent.Status),
			When:   sent.URL + "  " + sent.At.Format("15:04:05"),
			Open:   func() { l.Open(host, sent) },
		},
		"gutter": w.Gutter,
	})
}

func status(code int) string {
	if code == 0 {
		return "failed"
	}
	return fmt.Sprint(code)
}
