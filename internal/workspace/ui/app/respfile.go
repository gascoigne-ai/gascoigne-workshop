package app

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/ui/respview"
	"os"
	"path/filepath"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/response/model/textfmt"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// historyMenu lists the earlier responses; picking one shows it again.
func (a *App) historyMenu(tab *reqTab) []MenuItem {
	items := make([]MenuItem, 0, len(tab.Earlier))
	for i, h := range tab.Earlier {
		label := fmt.Sprintf("%d %s  %s  %s", h.Resp.Status, h.Resp.StatusText, textfmt.Duration(h.Resp.Duration), textfmt.Size(h.Resp.Size))
		items = append(items, MenuItem{Label: label, Run: func() { a.showHistory(tab, i) }})
	}
	return items
}

// showHistory makes the response of history entry i the shown one; the
// shown one goes into the history in its place, so nothing is lost.
func (a *App) showHistory(tab *reqTab, index int) {
	if tab.Recall(index, a.now()) {
		a.refreshView(tab)
	}
}

// saveBody asks for a path and writes the response body there.
func (a *App) saveBody(tab *reqTab) {
	if tab.Resp == nil || tab.Resp.Err != "" {
		a.toast("There is no response to save", true)
		return
	}
	a.pickFile("Save response body", true, respview.SuggestedFileName(tab.Resp), func(path string) { a.writeBody(tab, path) })
}

// writeBody saves the body to path (a leading ~ is the home folder); an
// existing file is not overwritten without the user saying so by typing
// its name again, which is what the prompt already did, so it is replaced.
func (a *App) writeBody(tab *reqTab, path string) {
	path = strings.TrimSpace(path)
	if path == "" || tab.Resp == nil {
		return
	}
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, rest)
		}
	}
	if err := os.WriteFile(path, tab.Resp.Body, 0o644); err != nil {
		a.toast("Saving failed: "+err.Error(), true)
		return
	}
	a.toast(fmt.Sprintf("Saved %s to %s", textfmt.Size(len(tab.Resp.Body)), filepath.Base(path)), false)
}

// openInBrowser serves the response body on a temporary loopback port
// and opens it in the user's browser, which renders what this app cannot.
func (a *App) openInBrowser(tab *reqTab) {
	if tab.Resp == nil || tab.Resp.Err != "" {
		a.toast("There is no response to open", true)
		return
	}
	contentType := respview.ContentType(tab.Resp)
	if contentType == "" {
		contentType = "text/plain; charset=utf-8"
	}
	if err := a.previews.Show(tab.Resp.Body, contentType); err != nil {
		a.toast("Preview failed: "+err.Error(), true)
		return
	}
	a.toast("Opened in the browser for 5 minutes", false)
}

// toggleWrap switches the response body between scrolling sideways and
// breaking long lines at the width of the pane.
func (a *App) toggleWrap(tab *reqTab) { tab.view.Wrap = !tab.view.Wrap }
