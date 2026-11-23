package response

import (
	"fmt"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/viewer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/format"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/response/respview"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/hajimehoshi/ebiten/v2"
)

// SetMode picks the response view of tab: pretty-printed, as received, or
// compact.
func (c *Component) SetMode(tab *tabs.Tab, mode viewer.Mode) {
	state := c.state(tab)
	state.look = viewer.Update(state.look, viewer.ModeChosen{Mode: mode})
	c.Refresh(tab)
}

// Refresh rebuilds the response view of tab for the current mode and
// redoes an open search. Bodies that are not text are not loaded.
func (c *Component) Refresh(tab *tabs.Tab) {
	state := c.state(tab)
	state.picture, state.pictureInfo = nil, ""
	if tab.Resp != nil {
		if picture, ok := respview.Decode(tab.Resp.Body); ok {
			state.picture = ebiten.NewImageFromImage(picture.Image)
			state.pictureInfo = fmt.Sprintf("%s %dx%d, %s", strings.ToUpper(picture.Format), picture.Width, picture.Height, format.Size(len(tab.Resp.Body)))
			state.view.SetLines(nil)
			return
		}
	}
	switch {
	case tab.Resp == nil || respview.LooksBinary(tab.Resp.Body):
		state.view.SetLines(nil)
	case respview.TooBig(tab.Resp.Body): // formatting megabytes would freeze the window
		state.view.SetLines(respview.LimitedLines(tab.Resp.Body))
	case state.look.Raw():
		state.view.SetLines(respview.RawLines(tab.Resp.Body))
	case state.look.Compact():
		state.compactAll = respview.CompactLines(tab.Resp.Body)
		c.showCompact(tab)
		return // showCompact searches too
	default:
		state.view.SetLines(respview.CodeLines(tab.Resp.Body))
	}
	if state.look.FindOpen {
		c.runFind(tab)
	}
}

// ClearView empties the lines of tab, for a response that was replaced by
// an error and has no body to show. The find bar stays as it is: the message
// that covers the pane hides it.
func (c *Component) ClearView(tab *tabs.Tab) {
	c.state(tab).view.SetLines(nil)
}

// OpenFind shows the find bar for the response of the current tab and
// focuses it.
func (c *Component) OpenFind() {
	current := c.deps.Tabs.Current()
	if current == nil {
		return
	}
	c.deps.Window.CommitEdit()
	c.deps.Window.ShowResponse(current)
	state := c.state(current)
	state.look = viewer.Update(state.look, viewer.FindOpened{})
	state.find.Focused = true
	state.find.SelectAll()
	c.runFind(current)
}

// CloseFind hides the find bar of tab and the highlights it left.
func (c *Component) CloseFind(tab *tabs.Tab) {
	state := c.state(tab)
	state.look = viewer.Update(state.look, viewer.FindClosed{})
	state.find.Focused = false
	state.view.SetMatches(nil, -1)
	if state.look.Compact() && formattable(tab) {
		c.showCompact(tab) // leave the filtered view
	}
}

// Find replaces the query of tab with text and searches for it, as typing
// would.
func (c *Component) Find(tab *tabs.Tab, text string) {
	state := c.state(tab)
	state.find.SelectAll()
	state.find.Insert(text)
	c.runFind(tab)
}

// showCompact fills the view with the compact lines: all of them with
// the deep blocks folded, or, while a search has text, only the matching
// lines with their children and the blocks around them (respview.Filter).
func (c *Component) showCompact(tab *tabs.Tab) {
	state := c.state(tab)
	if state.look.FindOpen && state.find.Text != "" {
		state.view.SetLines(respview.Filter(state.compactAll, state.find.Text))
	} else {
		state.view.SetLines(state.compactAll)
		respview.FoldDeep(&state.view, respview.CompactFoldDepth)
	}
	if state.look.FindOpen {
		c.searchView(tab)
	}
}

// formattable is whether tab has a text body small enough to be formatted.
func formattable(tab *tabs.Tab) bool {
	return tab.Resp != nil && !respview.LooksBinary(tab.Resp.Body) && !respview.TooBig(tab.Resp.Body)
}

// runFind searches for the typed text and goes to the first match. The
// compact view is filtered to the matches first.
func (c *Component) runFind(tab *tabs.Tab) {
	if c.state(tab).look.Compact() && formattable(tab) {
		c.showCompact(tab)
		return
	}
	c.searchView(tab)
}

// searchView highlights the matches in the view and reveals the first.
func (c *Component) searchView(tab *tabs.Tab) {
	state := c.state(tab)
	matches := state.view.Find(state.find.Text)
	state.look = viewer.Update(state.look, viewer.Searched{Matches: len(matches)})
	if len(matches) > 0 {
		state.view.RevealMatch(matches[0])
	}
	state.view.SetMatches(matches, state.look.Match)
}

// findStep moves to the next (dir 1) or previous (dir -1) match, wrapping.
func (c *Component) findStep(tab *tabs.Tab, dir int) {
	state := c.state(tab)
	matches := state.view.Matches()
	if len(matches) == 0 {
		return
	}
	state.look = viewer.Update(state.look, viewer.Stepped{Dir: dir, Matches: len(matches)})
	next := state.look.Match
	state.view.SetMatches(matches, next)
	state.view.RevealMatch(matches[next])
}

// findEnter is Enter in the find field: next match, previous with Shift.
func (c *Component) findEnter(ctx *core.Ctx, tab *tabs.Tab) {
	dir := 1
	if ctx.In.Shift {
		dir = -1
	}
	c.findStep(tab, dir)
}
