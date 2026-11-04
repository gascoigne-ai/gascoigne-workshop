package ui

import (
	"fmt"
	"mime"
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

// extensions are the file extensions of the content types people save;
// the system's MIME table is the fallback, and its first choice is not
// always the usual one (".ehtml" for HTML).
var extensions = map[string]string{
	"application/json": ".json", "text/html": ".html", "application/xml": ".xml", "text/xml": ".xml",
	"text/plain": ".txt", "text/csv": ".csv", "text/css": ".css", "text/javascript": ".js",
	"image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif", "application/pdf": ".pdf",
}

// suggestedFileName is a name for saving the body: response.json, .html...
// from the content type.
func suggestedFileName(r *model.Response) string {
	for _, h := range r.Headers {
		if !strings.EqualFold(h.Name, "Content-Type") {
			continue
		}
		mt, _, err := mime.ParseMediaType(h.Value)
		if err != nil {
			break
		}
		if ext, ok := extensions[mt]; ok {
			return "response" + ext
		}
		if strings.HasSuffix(mt, "+json") {
			return "response.json"
		}
		if exts, _ := mime.ExtensionsByType(mt); len(exts) > 0 {
			return "response" + exts[0]
		}
	}
	return "response.txt"
}

// saveBody asks for a path and writes the response body there.
func (a *App) saveBody(t *reqTab) {
	if t.resp == nil || t.resp.Err != "" {
		a.toast("There is no response to save", true)
		return
	}
	a.pickFile("Save response body", true, suggestedFileName(t.resp), func(path string) { a.writeBody(t, path) })
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
