package app

import (
	"fmt"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/viewer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/format"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/response/respview"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"strings"
	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// responseEmpty is noResponse plus the case of a body that is not text.
func responseEmpty(app *App, tab *reqTab) (string, bool) {
	if msg, isErr := noResponse(app, tab); msg != "" {
		return msg, isErr
	}
	if tab.picture != nil {
		return "", false
	}
	if respview.LooksBinary(tab.Resp.Body) {
		return fmt.Sprintf("Binary body, %s (not shown)", format.Size(len(tab.Resp.Body))), false
	}
	return "", false
}

// The text view and the picture are two roots, so two templates; Go
// picks one in responsePane.
var (
	responsePaneTemplate    = markup.Must("responsepane.vml", responsepaneSource)
	responsePictureTemplate = markup.Must("responsepicture.vml", responsepictureSource)
)

// responsePane is the response body with its toolbar and find bar, or
// the picture of an image body with its size and a Save button.
func (a *App) responsePane(_ pane, tab *reqTab, _ *ScrollState) Node {
	if tab.picture != nil {
		return responsePictureTemplate.MustBuild(a.scope(markup.Scope{
			"pictureInfo": tab.pictureInfo,
			"picture":     tab.picture,
			"save":        func() { a.saveBody(tab) },
		}))
	}
	var findBar Node // stays nil, so the slot is left out, while the bar is closed
	if tab.look.FindOpen {
		findBar = a.findBar(tab)
	}
	return responsePaneTemplate.MustBuild(a.scope(markup.Scope{
		"compact":     tab.look.Compact(),
		"pretty":      tab.look.Mode == viewer.Pretty,
		"raw":         tab.look.Raw(),
		"wrap":        tab.view.Wrap,
		"history":     len(tab.Earlier) > 0,
		"findTip":     a.hint("Find in response", "find"),
		"findBar":     findBar,
		"view":        &tab.view,
		"showCompact": func() { a.setView(tab, viewer.Compact) },
		"showPretty":  func() { a.setView(tab, viewer.Pretty) },
		"showRaw":     func() { a.setView(tab, viewer.Raw) },
		"toggleWrap":  func() { a.toggleWrap(tab) },
		"openHistory": func(posX, posY int) { a.openMenu(posX, posY, a.historyMenu(tab)) },
		"openFind":    a.openFind,
		"copyBody":    func() { a.copyBody(tab) },
		"copyCurl":    func() { a.copyCurl(tab) },
		"openMenu":    func(posX, posY int) { a.openMenu(posX, posY, a.responseMenuItems(tab)) },
	}))
}

func (a *App) viewChip(label string, selected bool, width int, onClick func()) Node {
	return SegmentChip(label, selected, W(width), HOf(w.CompactOf), OnClick(onClick))
}

var findBarTemplate = markup.Must("findbar.vml", findbarSource)

// findBar is the search row above the response: the query, the hit
// counter and the previous, next and close buttons.
func (a *App) findBar(tab *reqTab) Node {
	return findBarTemplate.MustBuild(a.scope(markup.Scope{
		"find":     &tab.find,
		"count":    viewer.Count(tab.find.Text, len(tab.view.Matches()), tab.look.Match),
		"search":   func() { a.runFind(tab) },
		"enter":    func() { a.findEnter(tab) },
		"previous": func() { a.findStep(tab, -1) },
		"next":     func() { a.findStep(tab, 1) },
		"close":    func() { a.closeFind(tab) },
	}))
}

// setView picks the response view: pretty-printed, as received, or compact.
func (a *App) setView(tab *reqTab, mode viewer.Mode) {
	tab.look = viewer.Update(tab.look, viewer.ModeChosen{Mode: mode})
	a.refreshView(tab)
}

// refreshView rebuilds the response view for the current mode and
// redoes an open search. Bodies that are not text are not loaded.
func (a *App) refreshView(tab *reqTab) {
	tab.picture, tab.pictureInfo = nil, ""
	if tab.Resp != nil {
		if p, ok := respview.Decode(tab.Resp.Body); ok {
			tab.picture = ebiten.NewImageFromImage(p.Image)
			tab.pictureInfo = fmt.Sprintf("%s %dx%d, %s", strings.ToUpper(p.Format), p.Width, p.Height, format.Size(len(tab.Resp.Body)))
			tab.view.SetLines(nil)
			return
		}
	}
	switch {
	case tab.Resp == nil || respview.LooksBinary(tab.Resp.Body):
		tab.view.SetLines(nil)
	case respview.TooBig(tab.Resp.Body): // formatting megabytes would freeze the window
		tab.view.SetLines(respview.LimitedLines(tab.Resp.Body))
	case tab.look.Raw():
		tab.view.SetLines(respview.RawLines(tab.Resp.Body))
	case tab.look.Compact():
		tab.compactAll = respview.CompactLines(tab.Resp.Body)
		a.showCompact(tab)
		return // showCompact searches too
	default:
		tab.view.SetLines(respview.CodeLines(tab.Resp.Body))
	}
	if tab.look.FindOpen {
		a.runFind(tab)
	}
}

func (a *App) copyBody(tab *reqTab) {
	if tab.Resp != nil {
		_ = a.ctx.Clipboard.Write(string(tab.Resp.Body)) // the clipboard reports its own failures
	}
}

func (a *App) copyCurl(tab *reqTab) {
	_ = a.ctx.Clipboard.Write(a.curlOf(tab.Req.Expanded(a.varsFor(tab.Req))))
}

// openFind shows the find bar for the current response and focuses it.
func (a *App) openFind() {
	a.commitEdit()
	t := a.cur()
	t.respTab = 0
	t.look = viewer.Update(t.look, viewer.FindOpened{})
	t.find.Focused = true
	t.find.SelectAll()
	a.runFind(t)
}

// showCompact fills the view with the compact lines: all of them with
// the deep blocks folded, or, while a search has text, only the matching
// lines with their children and the blocks around them (filterCompact).
func (a *App) showCompact(tab *reqTab) {
	if tab.look.FindOpen && tab.find.Text != "" {
		tab.view.SetLines(respview.Filter(tab.compactAll, tab.find.Text))
	} else {
		tab.view.SetLines(tab.compactAll)
		respview.FoldDeep(&tab.view, respview.CompactFoldDepth)
	}
	if tab.look.FindOpen {
		a.searchView(tab)
	}
}

// formattable is whether tab has a text body small enough to be formatted.
func formattable(tab *reqTab) bool {
	return tab.Resp != nil && !respview.LooksBinary(tab.Resp.Body) && !respview.TooBig(tab.Resp.Body)
}

// runFind searches for the typed text and goes to the first match. The
// compact view is filtered to the matches first.
func (a *App) runFind(tab *reqTab) {
	if tab.look.Compact() && formattable(tab) {
		a.showCompact(tab)
		return
	}
	a.searchView(tab)
}

// searchView highlights the matches in the view and reveals the first.
func (a *App) searchView(tab *reqTab) {
	matches := tab.view.Find(tab.find.Text)
	tab.look = viewer.Update(tab.look, viewer.Searched{Matches: len(matches)})
	if len(matches) > 0 {
		tab.view.RevealMatch(matches[0])
	}
	tab.view.SetMatches(matches, tab.look.Match)
}

// findStep moves to the next (dir 1) or previous (dir -1) match, wrapping.
func (a *App) findStep(tab *reqTab, dir int) {
	matches := tab.view.Matches()
	if len(matches) == 0 {
		return
	}
	tab.look = viewer.Update(tab.look, viewer.Stepped{Dir: dir, Matches: len(matches)})
	next := tab.look.Match
	tab.view.SetMatches(matches, next)
	tab.view.RevealMatch(matches[next])
}

// findEnter is Enter in the find field: next match, previous with Shift.
func (a *App) findEnter(tab *reqTab) {
	dir := 1
	if a.ctx.In.Shift {
		dir = -1
	}
	a.findStep(tab, dir)
}

func (a *App) closeFind(tab *reqTab) {
	tab.look = viewer.Update(tab.look, viewer.FindClosed{})
	tab.find.Focused = false
	tab.view.SetMatches(nil, -1)
	if tab.look.Compact() && formattable(tab) {
		a.showCompact(tab) // leave the filtered view
	}
}

// curlOf is the cURL command of req, "" when the app has no formatter.
func (a *App) curlOf(req request.Request) string {
	if a.curl == nil {
		return ""
	}
	return a.curl(req)
}

// Templates, parsed once above.
const (
	responsepaneSource = `<col grow="1">
  <!-- same top inset as the request pane, so first rows line up -->
  <spacer h="{{gutter}}"/>
  <!-- written out: markup has no toolbar element; this is widget.Toolbar, copied in responsepicture.vml -->
  <row h="{{toolbarH}}" pad-x="{{gutter}}" gap="{{tightGap}}" align-items="center" border-b="border">
    <segment-chip selected="{{compact}}" w="{{compactChipW}}" h="{{compactH}}" on-click="showCompact">Compact</segment-chip>
    <segment-chip selected="{{pretty}}" w="{{prettyChipW}}" h="{{compactH}}" on-click="showPretty">Pretty</segment-chip>
    <segment-chip selected="{{raw}}" w="{{rawChipW}}" h="{{compactH}}" on-click="showRaw">Raw</segment-chip>
    <row w="{{gutter}}"/>
    <segment-chip selected="{{wrap}}" w="{{wrapChipW}}" h="{{compactH}}" on-click="toggleWrap">Wrap</segment-chip>
    <spacer/>
    <button if="history" w="{{historyBtnW}}" h="{{compactH}}" on-click-at="openHistory" tooltip="Earlier responses of this request">History</button>
    <icon-button icon="search" w="{{iconBtnW}}" h="{{compactH}}" on-click="openFind" tooltip="{{findTip}}"/>
    <icon-button icon="copy" w="{{iconBtnW}}" h="{{compactH}}" on-click="copyBody" tooltip="Copy the response body"/>
    <icon-button icon="prompt" w="{{iconBtnW}}" h="{{compactH}}" on-click="copyCurl" tooltip="Copy the request as a cURL command"/>
  </row>
  <slot if="findBar" name="findBar"/>
  <code state="view" grow="1" bg="sunken" wrap="{{wrap}}" on-right-click="openMenu"/>
</col>
`

	responsepictureSource = `<col grow="1">
  <spacer h="{{gutter}}"/>
  <!-- written out: markup has no toolbar element; this is widget.Toolbar, copied from responsepane.vml -->
  <row h="{{toolbarH}}" pad-x="{{gutter}}" gap="{{tightGap}}" align-items="center" border-b="border">
    <text small muted>{{pictureInfo}}</text>
    <spacer/>
    <button w="{{saveBtnW}}" h="{{compactH}}" on-click="save" tooltip="Save the image to a file">Save</button>
  </row>
  <image img="picture" grow="1" bg="sunken" tooltip="The image, scaled down to fit"/>
</col>
`

	findbarSource = `<row h="{{rowH}}" pad-x="{{gutter}}" gap="{{tightGap}}" border-b="border">
  <field state="find" grow="1" placeholder="Find" on-change="search" on-enter="enter" text-menu="find"/>
  <text small muted w="{{countW}}">{{count}}</text>
  <icon-button icon="chevron-up" w="{{iconBtnW}}" h="{{compactH}}" on-click="previous" tooltip="Previous match"/>
  <icon-button icon="chevron-down" w="{{iconBtnW}}" h="{{compactH}}" on-click="next" tooltip="Next match"/>
  <icon-button icon="cross" w="{{iconBtnW}}" h="{{compactH}}" on-click="close" tooltip="Close search (Esc)"/>
</row>
`
)
