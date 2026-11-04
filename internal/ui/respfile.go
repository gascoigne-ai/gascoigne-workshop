package ui

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/respview"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/textfmt"
)

// respHistoryLimit is how many earlier responses a tab keeps for its History menu.
const respHistoryLimit = 10

// remember puts the response a new one replaces into the tab's history,
// newest first. Failed requests are not kept: there is nothing to look at.
func remember(t *reqTab, old *model.Response) {
	if old == nil || old.Err != "" {
		return
	}
	t.respHist = append([]respEntry{{resp: old, at: time.Now()}}, t.respHist...)
	if len(t.respHist) > respHistoryLimit {
		t.respHist = t.respHist[:respHistoryLimit]
	}
}

// respEntry is an earlier response of a tab and when it was replaced.
type respEntry struct {
	resp *model.Response
	at   time.Time
}

// historyMenu lists the earlier responses; picking one shows it again.
func (a *App) historyMenu(t *reqTab) []menuItem {
	items := make([]menuItem, 0, len(t.respHist))
	for i, h := range t.respHist {
		label := fmt.Sprintf("%d %s  %s  %s", h.resp.Status, h.resp.StatusText, textfmt.Duration(h.resp.Duration), textfmt.Size(h.resp.Size))
		items = append(items, menuItem{label: label, run: func() { a.showHistory(t, i) }})
	}
	return items
}

// showHistory makes the response of history entry i the shown one; the
// shown one goes into the history in its place, so nothing is lost.
func (a *App) showHistory(t *reqTab, i int) {
	if i < 0 || i >= len(t.respHist) {
		return
	}
	picked := t.respHist[i].resp
	if t.resp != nil && t.resp.Err == "" {
		t.respHist[i] = respEntry{resp: t.resp, at: time.Now()}
	} else {
		t.respHist = append(t.respHist[:i], t.respHist[i+1:]...)
	}
	t.resp = picked
	a.refreshView(t)
}

// saveBody asks for a path and writes the response body there.
func (a *App) saveBody(t *reqTab) {
	if t.resp == nil || t.resp.Err != "" {
		a.toast("There is no response to save", true)
		return
	}
	a.pickFile("Save response body", true, respview.SuggestedFileName(t.resp), func(path string) { a.writeBody(t, path) })
}

// writeBody saves the body to path (a leading ~ is the home folder); an
// existing file is not overwritten without the user saying so by typing
// its name again, which is what the prompt already did, so it is replaced.
func (a *App) writeBody(t *reqTab, path string) {
	path = strings.TrimSpace(path)
	if path == "" || t.resp == nil {
		return
	}
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, rest)
		}
	}
	if err := os.WriteFile(path, t.resp.Body, 0o644); err != nil {
		a.toast("Saving failed: "+err.Error(), true)
		return
	}
	a.toast(fmt.Sprintf("Saved %s to %s", textfmt.Size(len(t.resp.Body)), filepath.Base(path)), false)
}
