// Package calltrace is the log of calls the Log dialog shows: requests
// sent, how each ended, and connections opened, lost or refused. It keeps
// the last entries in memory, filters them for display and draws the
// dialog; the application adds entries and opens the dialog.
package calltrace

import (
	"fmt"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/textfmt"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Limit is how many entries the log keeps.
const Limit = 500

// The size of the dialog, like the other dialogs.
const (
	dialogW    = 820
	dialogMaxH = 560
	margin     = 40
	titleH     = 44
)

// Host is what the dialog needs from the application that shows it.
type Host interface {
	// Size is the window's size in logical pixels.
	Size() (width, height int)
	// TextMenu is the option that gives a field its right-click menu.
	TextMenu(field *TextField) Option
}

// Entry is one line of the log: a request sent, its answer, a connection
// opened or lost.
type Entry struct {
	At    time.Time
	Text  string
	IsErr bool
}

// Log is the call trace and the state of its dialog. The zero value is an
// empty, closed log.
type Log struct {
	entries []Entry
	filter  TextField
	code    CodeView
	key     string // what code shows now
	shown   bool
}

// Info adds an entry, dropping the oldest past Limit.
func (l *Log) Info(at time.Time, text string) { l.add(Entry{At: at, Text: text}) }

// Error adds an entry that is shown as a failure.
func (l *Log) Error(at time.Time, text string) { l.add(Entry{At: at, Text: text, IsErr: true}) }

func (l *Log) add(entry Entry) {
	l.entries = append(l.entries, entry)
	if len(l.entries) > Limit {
		l.entries = l.entries[len(l.entries)-Limit:]
	}
}

// Sent notes that req is about to be sent.
func (l *Log) Sent(at time.Time, req model.Request) {
	l.Info(at, fmt.Sprintf("→ %s %s", req.Method, req.URL))
}

// Answer notes how a send ended.
func (l *Log) Answer(at time.Time, req *model.Request, resp *model.Response) {
	if resp.Err != "" {
		l.Error(at, fmt.Sprintf("✗ %s %s failed: %s", req.Method, req.URL, resp.Err))
		return
	}
	l.Info(at, fmt.Sprintf("← %d %s in %s, %s", resp.Status, resp.StatusText, textfmt.Duration(resp.Duration), textfmt.Size(resp.Size)))
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
		if entry.IsErr {
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

// View is the dialog while it is open, nothing otherwise.
func (l *Log) View(host Host) Node {
	if !l.shown {
		return nil
	}
	l.refresh()
	_, height := host.Size()
	h := min(dialogMaxH, height-2*margin)
	return Frag(
		Row(Bg(w.Shadow)), // dims the window
		Col(W(dialogW), H(h), Bg(w.Bg), Control("menu"), Shadow(), FloatCenter(), FocusScope(), Modal(), OnDismiss(l.Close),
			Row(H(titleH), PadX(16), Gap(8), AlignItems(AlignCenter), BorderB(w.BorderSoft),
				Text("Log", Bold(), W(40)),
				Field(&l.filter, H(28), Grow(1), Bg(w.Sunken), Placeholder("Filter"), host.TextMenu(&l.filter)),
				Button("Clear", W(64), H(w.CompactH), OnClick(l.Clear), Tooltip("Forget the entries")),
				IconButton(CrossIcon(Grow(1)), W(w.IconBtnW), H(w.CompactH), OnClick(l.Close), Tooltip("Close (Esc)")),
			),
			Code(&l.code, Grow(1), Bg(w.Sunken)),
		),
	)
}
