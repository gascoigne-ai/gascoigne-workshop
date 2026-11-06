package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/textfmt"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
)

// traceLimit is how many entries the call trace keeps.
const traceLimit = 500

// traceW and traceMargin size the trace view like the other dialogs.
const (
	traceW      = 820
	traceMaxH   = 560
	traceMargin = 40
)

// traceEntry is one line of the call trace: a request sent, its answer, a
// connection opened or lost.
type traceEntry struct {
	at    time.Time
	text  string
	isErr bool
}

// traceInfo adds an entry to the call trace, dropping the oldest past
// traceLimit.
func (a *App) traceInfo(text string) { a.addTrace(traceEntry{at: a.now(), text: text}) }

// traceError adds an entry that is shown as a failure.
func (a *App) traceError(text string) { a.addTrace(traceEntry{at: a.now(), text: text, isErr: true}) }

func (a *App) addTrace(entry traceEntry) {
	a.trace = append(a.trace, entry)
	if len(a.trace) > traceLimit {
		a.trace = a.trace[len(a.trace)-traceLimit:]
	}
}

// traceSent notes that req is about to be sent.
func (a *App) traceSent(req model.Request) {
	a.traceInfo(fmt.Sprintf("→ %s %s", req.Method, req.URL))
}

// traceAnswer notes how a send ended.
func (a *App) traceAnswer(req *model.Request, resp *model.Response) {
	if resp.Err != "" {
		a.traceError(fmt.Sprintf("✗ %s %s failed: %s", req.Method, req.URL, resp.Err))
		return
	}
	a.traceInfo(fmt.Sprintf("← %d %s in %s, %s", resp.Status, resp.StatusText, textfmt.Duration(resp.Duration), textfmt.Size(resp.Size)))
}

// traceMatches are the entries the filter keeps.
func (a *App) traceMatches() []traceEntry {
	needle := strings.ToLower(a.traceFilter.Text)
	if needle == "" {
		return a.trace
	}
	var kept []traceEntry
	for _, entry := range a.trace {
		if strings.Contains(strings.ToLower(entry.text), needle) {
			kept = append(kept, entry)
		}
	}
	return kept
}

// traceStamp is the time column of an entry.
func traceStamp(at time.Time) string { return at.Format("15:04:05.000") }

// traceLines are the shown entries as text, oldest first.
func (a *App) traceLines() []string {
	entries := a.traceMatches()
	lines := make([]string, len(entries))
	for i, entry := range entries {
		lines[i] = traceStamp(entry.at) + "  " + entry.text
	}
	return lines
}

func (a *App) clearTrace() {
	a.trace = nil
	a.traceKey = ""
}

func (a *App) openTrace()  { a.traceShown = true }
func (a *App) closeTrace() { a.traceShown = false }

// refreshTrace rebuilds the code view when the entries or the filter changed,
// so a selection in it survives the frames in between.
func (a *App) refreshTrace() {
	key := fmt.Sprintf("%d|%d|%s", len(a.trace), a.traceEnd(), a.traceFilter.Text)
	if key == a.traceKey {
		return
	}
	a.traceKey = key
	entries := a.traceMatches()
	lines := make([]Line, len(entries))
	for i, entry := range entries {
		style := w.Text
		if entry.isErr {
			style = w.Error
		}
		lines[i] = Line{Spans: []Span{{Text: traceStamp(entry.at) + "  ", Style: w.Muted}, {Text: entry.text, Style: style}}}
	}
	a.traceCode.SetLines(lines)
	a.traceCode.Follow = true
}

// traceEnd is the time of the newest entry, which tells a trimmed full
// list from an unchanged one of the same length.
func (a *App) traceEnd() int64 {
	if len(a.trace) == 0 {
		return 0
	}
	return a.trace[len(a.trace)-1].at.UnixNano()
}

// traceView is the call trace dialog while it is open.
func (a *App) traceView() Node {
	if !a.traceShown {
		return nil
	}
	a.refreshTrace()
	h := min(traceMaxH, a.h-2*traceMargin)
	return Frag(
		Row(Bg(w.Shadow)), // dims the window
		Col(W(traceW), H(h), Bg(w.Bg), Control("menu"), Shadow(), FloatCenter(), FocusScope(), Modal(), OnDismiss(a.closeTrace),
			Row(H(settingsTitleH), PadX(16), Gap(8), AlignItems(AlignCenter), BorderB(w.BorderSoft),
				Text("Log", Bold(), W(40)),
				Field(&a.traceFilter, H(28), Grow(1), Bg(w.Sunken), Placeholder("Filter"), a.textMenuOpt(&a.traceFilter, nil)),
				Button("Clear", W(64), H(w.CompactH), OnClick(a.clearTrace), Tooltip("Forget the entries")),
				IconButton(CrossIcon(Grow(1)), W(w.IconBtnW), H(w.CompactH), OnClick(a.closeTrace), Tooltip("Close (Esc)")),
			),
			Code(&a.traceCode, Grow(1), Bg(w.Sunken)),
		),
	)
}
