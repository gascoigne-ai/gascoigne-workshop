// Package calltrace is the log of calls the Log dialog shows: requests
// sent, how each ended, and connections opened, lost or refused. It keeps
// the last entries in memory, filters them for display and draws the
// dialog; the application adds entries and opens the dialog.
package calltrace

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/format"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Limit is how many entries the log keeps.
const Limit = 500

// The size of the dialog, like the other dialogs.
const (
	dialogW    = 820
	dialogMaxH = w.DialogMaxH
	margin     = w.DialogMargin
)

// clearBtnW is the width of the Clear button.
const clearBtnW = 64

// Host is what the dialog needs from the application that shows it.
type Host interface {
	// Size is the window's size in logical pixels.
	Size() (width, height int)
}

// Entry is one line of the log: a request sent, its answer, a connection
// opened or lost.
type Entry struct {
	At    time.Time
	Text  string
	Level severity.Level
}

// Log is the call trace and the state of its dialog. The zero value is an
// empty, closed log.
type Log struct {
	entries []Entry
	filter  TextField
	code    CodeView
	key     string // what code shows now
	shown   bool
	move    MoveState // where the user dragged the dialog
}

// Info adds an entry, dropping the oldest past Limit.
func (l *Log) Info(at time.Time, text string) { l.add(Entry{At: at, Text: text}) }

// Error adds an entry that is shown as a failure.
func (l *Log) Error(at time.Time, text string) {
	l.add(Entry{At: at, Text: text, Level: severity.Error})
}

func (l *Log) add(entry Entry) {
	l.entries = append(l.entries, entry)
	if len(l.entries) > Limit {
		l.entries = l.entries[len(l.entries)-Limit:]
	}
}

// Sent notes that req is about to be sent.
func (l *Log) Sent(at time.Time, req request.Request) {
	l.Info(at, fmt.Sprintf("→ %s %s", req.Method, req.URL))
}

// Answer notes how a send ended.
func (l *Log) Answer(at time.Time, req *request.Request, resp *request.Response) {
	if resp.Err != "" {
		l.Error(at, fmt.Sprintf("✗ %s %s failed: %s", req.Method, req.URL, resp.Err))
		return
	}
	l.Info(at, fmt.Sprintf("← %d %s in %s, %s", resp.Status, resp.StatusText, format.Duration(resp.Duration), format.Size(resp.Size)))
}

// Entries are all the entries, oldest first.
func (l *Log) Entries() []Entry { return l.entries }

// SetFilter sets the text the shown entries must contain.
func (l *Log) SetFilter(text string) { l.filter.Text = text }

// matches are the entries the filter keeps.
func (l *Log) matches() []Entry {
	needle := strings.ToLower(l.filter.Text)
	if needle == "" {
		return l.entries
	}
	var kept []Entry
	for _, entry := range l.entries {
		if strings.Contains(strings.ToLower(entry.Text), needle) {
			kept = append(kept, entry)
		}
	}
	return kept
}

func stamp(at time.Time) string { return at.Format("15:04:05.000") }

// Lines are the shown entries as text, oldest first.
func (l *Log) Lines() []string {
	entries := l.matches()
	lines := make([]string, len(entries))
	for i, entry := range entries {
		lines[i] = stamp(entry.At) + "  " + entry.Text
	}
	return lines
}

// Clear forgets every entry.
func (l *Log) Clear() {
	l.entries = nil
	l.key = ""
}

// Open shows the dialog; Close hides it; Shown says whether it is open.
func (l *Log) Open()       { l.shown = true }
func (l *Log) Close()      { l.shown = false }
func (l *Log) Shown() bool { return l.shown }

// refresh rebuilds the code view when the entries or the filter changed,
// so a selection in it survives the frames in between.
func (l *Log) refresh() {
	key := fmt.Sprintf("%d|%d|%s", len(l.entries), l.newest(), l.filter.Text)
	if key == l.key {
		return
	}
	l.key = key
	entries := l.matches()
	lines := make([]Line, len(entries))
	for i, entry := range entries {
		style := w.Text
		if entry.Level == severity.Error {
			style = w.Error
		}
		lines[i] = Line{Spans: []Span{{Text: stamp(entry.At) + "  ", Style: w.Muted}, {Text: entry.Text, Style: style}}}
	}
	l.code.SetLines(lines)
	l.code.Follow = true
}

// newest is the time of the newest entry, which tells a trimmed full list
// from an unchanged one of the same length.
func (l *Log) newest() int64 {
	if len(l.entries) == 0 {
		return 0
	}
	return l.entries[len(l.entries)-1].At.UnixNano()
}

var (
	dialogTemplate  = markup.Must("dialog.vml", dialogSource)
	titleTemplate   = markup.Must("title.vml", titleSource)
	toolbarTemplate = markup.Must("toolbar.vml", toolbarSource)
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

// View is the dialog while it is open, nothing otherwise.
func (l *Log) View(ctx *w.Ctx, host Host) Node {
	if !l.shown {
		return nil
	}
	l.refresh()
	_, height := host.Size()
	return dialogTemplate.MustBuild(markup.Scope{
		"width":   dialogW,
		"height":  min(dialogMaxH, height-2*margin),
		"close":   l.Close,
		"move":    &l.move,
		"title":   l.titleBar(),
		"toolbar": l.toolbar(ctx.Sizes),
		"code":    &l.code,
	})
}

// titleBar is the dialog's title bar: the name and the close button. It is
// its own template so its tests can lay it out alone.
func (l *Log) titleBar() Node {
	count := fmt.Sprintf("%d entries", len(l.entries))
	if len(l.entries) == 1 {
		count = "1 entry"
	}
	return titleTemplate.MustBuild(markup.Scope{"close": l.Close, "subtitle": DialogSubtitle(count)})
}

// toolbar is the row under the title bar: the filter and Clear.
func (l *Log) toolbar(sizes w.Sizes) Node {
	return toolbarTemplate.MustBuild(scope(sizes, markup.Scope{
		"filter": &l.filter,
		"clear":  l.Clear,
		"clearW": clearBtnW,
	}))
}

// Templates, parsed once above.
const (
	dialogSource = `<dialog w="{{width}}" h="{{height}}" moved="move" on-dismiss="close">
  <slot name="title"/>
  <slot name="toolbar"/>
  <code state="code" grow="1" bg="sunken" round-b="menu"/>
</dialog>
`

	titleSource = `<!-- The dialog's title bar: the name, how many entries there are and the close
     button. The filter and Clear are in toolbar.vml. -->
<dialog-title title="Log" on-dismiss="close">
  <slot name="subtitle"/>
</dialog-title>
`

	toolbarSource = `<!-- The row under the title bar: the filter and Clear. -->
<dialog-toolbar>
  <field state="filter" h="{{rowH}}" grow="1" bg="sunken" placeholder="Filter" text-menu="filter"/>
  <button w="{{clearW}}" h="{{compactH}}" on-click="clear" tooltip="Forget the entries">Clear</button>
</dialog-toolbar>
`
)
