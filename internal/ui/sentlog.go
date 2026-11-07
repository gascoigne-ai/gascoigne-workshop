package ui

import (
	"fmt"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/appearance"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/history"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// recordSent adds a finished send of tab to the workspace log, newest
// first, and writes it to the state folder. A failed write is shown in the
// status line.
func (a *App) recordSent(t *reqTab, resp *model.Response) {
	key := a.requestPath(t.req)
	if key == "" {
		return
	}
	sent := history.Sent{At: a.now(), Path: key, Protocol: t.req.Protocol, Method: t.req.Method, URL: t.req.URL, Status: resp.Status}
	a.sent = append([]history.Sent{sent}, a.sent...)
	if len(a.sent) > history.SentLimit {
		a.sent = a.sent[:history.SentLimit]
	}
	dir := a.historyDir()
	if dir == "" {
		return
	}
	if err := history.AppendSent(dir, sent); err != nil {
		a.toast("History is not being saved: "+err.Error(), true)
	}
}

// loadSent reads the workspace log saved by earlier runs.
func (a *App) loadSent() {
	dir := a.historyDir()
	if dir == "" {
		return
	}
	sent, err := history.LoadSent(dir)
	if err != nil {
		a.toast("History could not be read: "+err.Error(), true)
		return
	}
	a.sent = sent
}

// sentRows is the log rows the sidebar filter keeps, newest first.
func (a *App) sentRows() []history.Sent {
	needle := strings.ToLower(a.filter.Text)
	if needle == "" {
		return a.sent
	}
	var rows []history.Sent
	for _, sent := range a.sent {
		if strings.Contains(strings.ToLower(sent.Path+" "+sent.Method+" "+sent.URL), needle) {
			rows = append(rows, sent)
		}
	}
	return rows
}

// openSent opens the request a log row was sent from; one that has since
// been deleted or renamed is reported.
func (a *App) openSent(sent history.Sent) {
	var found *model.Request
	walkRequests(&a.col.Folder, "", func(path string, req *model.Request) {
		if path == sent.Path {
			found = req
		}
	})
	if found == nil {
		a.status = fmt.Sprintf("%s is no longer in the collection", sent.Path)
		return
	}
	a.openRequest(found)
}

// sentList is the sidebar body while it shows the log instead of the tree.
func (a *App) sentList() Node {
	rows := a.sentRows()
	if len(rows) == 0 {
		return Col(Bg(w.Sunken), Pad(w.Gutter), Text("Nothing sent yet", Muted()))
	}
	return Scroll(&a.sentScroll, Bg(w.Sunken),
		For(rows, func(_ int, sent history.Sent) Node {
			return Col(PadX(w.Gutter), PadY(4), HoverBg(w.Hover), Control("row"), OnClick(func() { a.openSent(sent) }),
				Row(Gap(6), AlignItems(AlignBaseline),
					Text(requestBadge(sent.Protocol, sent.Method), Small(), Color(appearance.MethodToken(sent.Method)), W(badgeW)),
					Text(sent.Path, Ellipsis(), Grow(1)),
					Text(sentStatus(sent.Status), Small(), Muted()),
				),
				Row(Text(sent.URL+"  "+sent.At.Format("15:04:05"), Small(), Muted(), Ellipsis(), Grow(1))),
			)
		}),
	)
}

func sentStatus(status int) string {
	if status == 0 {
		return "failed"
	}
	return fmt.Sprint(status)
}
