package app

import (
	"context"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/history"
	request "github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/outbox"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/serial"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/tabstrip"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"slices"
	"time"
)

// reqTab is one open request with its editor and response state.
type reqTab struct {
	*tabs.Tab
	socketBlock
	responseBlock
}

// socketBlock holds the state of a socket tab the connection lifecycle owns: the connect in progress and the sends waiting their turn. The message log and the composer are the ws component's.
type socketBlock struct {
	wsCancel context.CancelFunc // ends the timeout of the connect in progress
	outbox   outbox.Outbox      // the sends and pings in order, and whether one is in flight
}

// responseBlock holds what the response area keeps per tab outside the
// response component: which pane is shown and how it scrolls. The panes of
// a socket tab share them, which is why they stay here.
type responseBlock struct {
	respTab    int
	respScroll widget.ScrollState // scroll of the table and note panes
	respTabsX  widget.ScrollState // sideways scroll of the two pane tab strips
}

// makeTab is newTab with the history of its request.
func (a *App) makeTab(tab *tabs.Tab) *reqTab {
	t := newReqTab(tab)
	t.Earlier = a.loadHistory(tab.Req)
	return t
}

// closeTab closes tab index; closing the last one leaves a blank request.
func (a *App) closeTab(index int) { a.updateTabs(tabs.Close{Index: index}) }

// updateTabs applies msg to the tab list, then does what the change means
// for the window: closed tabs lose their widgets and new ones get theirs.
func (a *App) updateTabs(msg tabs.Msg) {
	was := a.activeRequest()
	a.tabs.Apply(msg)
	// Widgets are made first: a restore reads cur() right after an Open.
	for _, tab := range a.tabs.Tabs() {
		if a.widgets[tab] == nil {
			a.widgets[tab] = a.makeTab(tab)
		}
	}
	for tab, widgets := range a.widgets {
		if slices.Contains(a.tabs.Tabs(), tab) {
			continue
		}
		a.cancelSend(widgets) // a closed tab keeps neither a request nor a socket
		a.disconnect(widgets)
		delete(a.widgets, tab)
	}
	if now := a.activeRequest(); now != nil && now != was {
		a.revealRequest(now)
	}
}

// activeRequest is the request of the current tab, nil when none is open.
func (a *App) activeRequest() *request.Request {
	current := a.tabs.Current()
	if current == nil {
		return nil
	}
	return current.Req
}

// openTabs are the widget states of the open tabs, in the order of the strip.
func (a *App) openTabs() []*reqTab {
	open := make([]*reqTab, len(a.tabs.Tabs()))
	for i, tab := range a.tabs.Tabs() {
		open[i] = a.widgets[tab]
	}
	return open
}

func newTab(req *request.Request) *reqTab { return newReqTab(tabs.New(req)) }

func newReqTab(tab *tabs.Tab) *reqTab {
	t := &reqTab{Tab: tab}
	t.wsCancel = func() {}
	return t
}

// tabOfRequest is the widget state of the tab showing req, nil when no open
// tab shows it.
func (a *App) tabOfRequest(req *request.Request) *reqTab {
	for tab, widgets := range a.widgets {
		if tab.Req == req {
			return widgets
		}
	}
	return nil
}

func (a *App) cur() *reqTab { return a.widgets[a.tabs.Current()] }

// openRequest focuses the tab of req, opening one if needed.
func (a *App) openRequest(req *request.Request) {
	a.side.Tree().Select(request.Node{Req: req})
	a.updateTabs(tabs.Open{Req: req})
}

// editRequest applies an edit to the request of tab, which is the collection's
// own, so the sidebar and the autosave see it at once.
func editRequest(tab *reqTab, msg requestedit.Msg) { *tab.Req = requestedit.Update(*tab.Req, msg) }

// dirtyMark is shown on tabs and in the window title when a request changed since it was opened.
const dirtyMark = tabstrip.DirtyMark

// dirty reports whether the request differs from its state when opened.
func dirty(tab *reqTab) bool { return tab.Dirty() }

// requestPath is the remembered path of req in its collection, "" when it
// is in no open collection.
func (a *App) requestPath(req *request.Request) string {
	_, path, _ := a.cols.Set().PathOf(req)
	return path
}

// historyStore is the history folder next to the state file, nil when
// nothing is saved or no history adapter was given.
func (a *App) historyStore() history.History {
	dir := a.stateFile.HistoryDir()
	if dir == "" || a.histories == nil {
		return nil
	}
	return a.histories(dir)
}

// saveHistory writes the successful responses of the tab's request, the
// shown one first, so they are there after a restart. A failed write is
// shown in the status line and nothing else changes.
func (a *App) saveHistory(tab *reqTab) {
	store, key := a.historyStore(), a.cols.Set().HistoryKey(tab.Req)
	if store == nil || key == "" {
		return
	}
	var entries []history.Entry
	if tab.Resp != nil && tab.Resp.Err == "" {
		entries = append(entries, historyEntry(tab.Resp, a.now()))
	}
	for _, earlier := range tab.Earlier {
		entries = append(entries, historyEntry(earlier.Resp, earlier.At))
	}
	a.writes.Write("history:"+key, serial.Job{ // the entries are values; the bodies are never changed
		Run: func() error { return store.Save(key, entries) },
		Done: func(err error) {
			if err != nil {
				a.toast("History is not being saved: "+err.Error(), severity.Error)
			}
		},
	})
}

// loadHistory is the saved responses of req, newest first.
func (a *App) loadHistory(req *request.Request) []tabs.PastResponse {
	store, key := a.historyStore(), a.cols.Set().HistoryKey(req)
	if store == nil || key == "" {
		return nil
	}
	saved, err := store.Load(key)
	if legacy := a.cols.Set().LegacyHistoryKey(req); err == nil && len(saved) == 0 && legacy != "" {
		saved, err = store.Load(legacy)
	}
	if err != nil {
		a.toast("History could not be read: "+err.Error(), severity.Error)
		return nil
	}
	entries := make([]tabs.PastResponse, 0, len(saved))
	for _, entry := range saved {
		entries = append(entries, tabs.PastResponse{Resp: &request.Response{
			Status: entry.Status, StatusText: entry.StatusText, Duration: entry.Duration,
			Size: len(entry.Body), Headers: entry.Headers, Body: entry.Body,
		}, At: entry.At})
	}
	return entries
}

func historyEntry(resp *request.Response, at time.Time) history.Entry {
	return history.Entry{At: at, Status: resp.Status, StatusText: resp.StatusText, Duration: resp.Duration, Headers: resp.Headers, Body: resp.Body}
}
