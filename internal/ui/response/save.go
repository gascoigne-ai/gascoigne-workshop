package response

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/format"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/response/respview"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// historyMenu lists the earlier responses; picking one shows it again.
func (c *Component) historyMenu(tab *tabs.Tab) []widget.MenuItem {
	items := make([]widget.MenuItem, 0, len(tab.Earlier))
	for i, earlier := range tab.Earlier {
		label := fmt.Sprintf("%d %s  %s  %s", earlier.Resp.Status, earlier.Resp.StatusText, format.Duration(earlier.Resp.Duration), format.Size(earlier.Resp.Size))
		items = append(items, widget.MenuItem{Label: label, Run: func() { c.showHistory(tab, i) }})
	}
	return items
}

// showHistory makes the response of history entry i the shown one; the
// shown one goes into the history in its place, so nothing is lost.
func (c *Component) showHistory(tab *tabs.Tab, index int) {
	if tab.Recall(index, c.deps.Defaults.Now()) {
		c.Refresh(tab)
	}
}

// menuItems is the menu of the response body: copy, save, open, find.
func (c *Component) menuItems(ctx *core.Ctx, tab *tabs.Tab) []widget.MenuItem {
	state := c.state(tab)
	noBody := tab.Resp == nil || tab.Resp.Err != ""
	return []widget.MenuItem{
		{Label: "Copy", Shortcut: "Ctrl+C", Disabled: !state.view.Selected(), Run: func() {
			_ = ctx.Clipboard.Write(state.view.SelectedText()) // the clipboard reports its own failures
		}},
		{Label: "Copy body", Run: func() { c.copyBody(ctx, tab) }},
		{Label: "Save body...", Disabled: noBody, Run: func() { c.saveBody(tab) }},
		{Label: "Open in browser", Disabled: noBody, Run: func() { c.openInBrowser(tab) }},
		{Label: "Copy as cURL", Run: func() { c.copyCurl(ctx, tab) }},
		{Separator: true},
		{Label: "Select all", Shortcut: "Ctrl+A", Run: func() { state.view.SelectLines(0, len(state.view.Lines)-1) }},
		{Label: "Find", Shortcut: c.deps.Hints.Shortcut("find"), Run: c.OpenFind},
	}
}

// copyRowItems copies parts of the header row index to the clipboard, or
// all the headers as text.
func copyRowItems(ctx *core.Ctx, rows []request.KV, index int) []widget.MenuItem {
	if index < 0 || index >= len(rows) {
		return nil
	}
	row := rows[index]
	copyText := func(text string) func() {
		return func() { _ = ctx.Clipboard.Write(text) } // the clipboard reports its own failures
	}
	var all strings.Builder
	for _, each := range rows {
		all.WriteString(each.Name + ": " + each.Value + "\n")
	}
	return []widget.MenuItem{
		{Label: "Copy value", Run: copyText(row.Value)},
		{Label: "Copy name", Run: copyText(row.Name)},
		{Label: "Copy as Name: Value", Run: copyText(row.Name + ": " + row.Value)},
		{Label: "Copy all headers", Run: copyText(all.String())},
	}
}

func (c *Component) copyBody(ctx *core.Ctx, tab *tabs.Tab) {
	if tab.Resp != nil {
		_ = ctx.Clipboard.Write(string(tab.Resp.Body)) // the clipboard reports its own failures
	}
}

func (c *Component) copyCurl(ctx *core.Ctx, tab *tabs.Tab) {
	_ = ctx.Clipboard.Write(c.deps.Exports.Curl(tab)) // the clipboard reports its own failures
}

// saveBody asks for a path and writes the response body there.
func (c *Component) saveBody(tab *tabs.Tab) {
	if tab.Resp == nil || tab.Resp.Err != "" {
		c.deps.Window.Toast("There is no response to save", severity.Error)
		return
	}
	c.deps.Files.PickSave("Save response body", respview.SuggestedFileName(tab.Resp), func(path string) { c.writeBody(tab, path) })
}

// writeBody saves the body to path (a leading ~ is the home folder); an
// existing file is not overwritten without the user saying so by typing
// its name again, which is what the prompt already did, so it is replaced.
func (c *Component) writeBody(tab *tabs.Tab, path string) {
	path = strings.TrimSpace(path)
	if path == "" || tab.Resp == nil {
		return
	}
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		if home, err := c.deps.Files.Home(); err == nil {
			path = filepath.Join(home, rest)
		}
	}
	body := tab.Resp.Body
	c.deps.Files.Write(path, body, func(err error) {
		if err != nil {
			c.deps.Window.Toast("Saving failed: "+err.Error(), severity.Error)
			return
		}
		c.deps.Window.Toast(fmt.Sprintf("Saved %s to %s", format.Size(len(body)), filepath.Base(path)), severity.Info)
	})
}

// openInBrowser serves the response body on a temporary loopback port
// and opens it in the user's browser, which renders what this app cannot.
func (c *Component) openInBrowser(tab *tabs.Tab) {
	if tab.Resp == nil || tab.Resp.Err != "" {
		c.deps.Window.Toast("There is no response to open", severity.Error)
		return
	}
	contentType := respview.ContentType(tab.Resp)
	if contentType == "" {
		contentType = "text/plain; charset=utf-8"
	}
	if err := c.deps.Exports.Preview(context.Background(), tab.Resp.Body, contentType); err != nil {
		c.deps.Window.Toast("Preview failed: "+err.Error(), severity.Error)
		return
	}
	c.deps.Window.Toast("Opened in the browser for 5 minutes", severity.Info)
}

// toggleWrap switches the response body between scrolling sideways and
// breaking long lines at the width of the pane.
func (c *Component) toggleWrap(tab *tabs.Tab) {
	state := c.state(tab)
	state.view.Wrap = !state.view.Wrap
}
