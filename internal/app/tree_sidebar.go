package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/collection"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// sidebarWidth is the current width of the sidebar in pixels.
func (a *App) sidebarWidth() int {
	if a.sidebarSplit.Size > 0 {
		return int(a.sidebarSplit.Size)
	}
	return sidebarW - 1
}

// ── Sidebar ────────────────────────────────────────────────────────────

var (
	sidebarTemplate     = markup.Must("sidebar.vml", sidebarSource)
	treeFolderTemplate  = markup.Must("treefolder.vml", treefolderSource)
	treeRequestTemplate = markup.Must("treerequest.vml", treerequestSource)
)

// sidebar is the header (the chip that switches between the collection
// tree and the log of sent requests, and the filter) above the one shown.
func (a *App) sidebar() Node {
	label, switchTo := "Collections", func() { a.showSent = true }
	if a.showSent {
		label, switchTo = "Sent", func() { a.showSent = false }
	}
	var sentList Node
	var rows []Node
	if a.showSent {
		sentList = a.sent.List(a.ctx, a, a.filter.Text)
	} else {
		for _, row := range a.visibleRows() {
			rows = append(rows, a.treeRow(row, row.Node() == a.selected))
		}
	}
	return sidebarTemplate.MustBuild(a.scope(markup.Scope{
		"mode":           label,
		"switchMode":     switchTo,
		"collectionMenu": func(posX, posY int) { a.openMenu(posX, posY, a.collectionMenu()) },
		"filter":         &a.filter,
		"showSent":       a.showSent,
		"showTree":       !a.showSent && len(a.open.Entries) > 0,
		"noCollection":   !a.showSent && len(a.open.Entries) == 0,
		"openFolder":     a.promptWorkspace,
		"newCollection":  a.promptNewCollection,
		"sentList":       sentList,
		"treeScroll":     &a.treeScroll,
		"rows":           rows,
		"headerH":        headerH,
	}))
}

// treeItem is what a tree row's template shows of one row.
type treeItem struct {
	Name     string // the whole name, shown when the filter matched none of it
	Before   string // the name cut around the part the filter matched
	Match    string
	After    string
	Indent   int
	Selected bool
	// Root marks a collection's own row: its name is bold, so where one
	// collection ends and the next begins reads at a glance.
	Root     bool
	Triangle string  // a folder's disclosure icon
	Badge    string  // a request's method or protocol
	Color    w.Token // the badge's color
	// DropBorder outlines the row a drag would drop on; empty, no outline.
	DropBorder w.Token
	Stripe     w.Token // a collection root's color
	Press      func()
	Menu       func(pointerX, pointerY int)
	DragOver   func()
}

// treeRow is one row of the tree, a folder or a request: one template
// each, since their rows differ in padding and alignment.
func (a *App) treeRow(row collection.Row, selected bool) Node {
	item := a.treeItemOf(row)
	item.Selected = selected
	template := treeFolderTemplate
	if !row.Folder {
		template = treeRequestTemplate
	}
	return template.MustBuild(a.scope(markup.Scope{"row": item, "badgeW": badgeSlot(a.ctx, badgeW)}))
}

// treeItemOf is the view model of one tree row, except whether it is selected.
func (a *App) treeItemOf(row collection.Row) treeItem {
	node := row.Node()
	item := treeItem{
		Name:     row.Name,
		Indent:   w.Gutter + row.Depth*treeIndent,
		Press:    func() { a.pressTreeRow(row) },
		Menu:     func(pointerX, pointerY int) { a.openMenu(pointerX, pointerY, a.treeMenuItems(row)) },
		DragOver: func() { a.dropTarget = node },
	}
	item.Before, item.Match, item.After = collection.SplitMatch(row.Name, a.filter.Text)
	if row.Folder {
		item.Root = a.isRoot(row.Dir)
		if item.Root {
			item.Stripe = a.collectionToken(node)
		}
		item.Triangle = "triangle-right"
		if row.Open {
			item.Triangle = "triangle-down"
		}
	} else {
		item.Indent += 6
		item.Badge = requestBadge(row.Req.Protocol, row.Req.Method)
		item.Color = theme.MethodToken(row.Req.Method)
	}
	if !a.dragNode.None() && a.dropTarget == node && node != a.dragNode {
		item.DropBorder = w.AccentLine
		if a.ctx.In.Alt { // released now, it copies: the outline turns green
			item.DropBorder = w.OK
		}
	}
	return item
}

// pressTreeRow is a press on a row: it selects it, gives the tree the
// keyboard and starts a drag. A request opens at once; a folder toggles
// when the button is released on the same row (see finishDrag), so an
// open folder can still be dragged.
func (a *App) pressTreeRow(row collection.Row) {
	a.treeFocus = true
	a.dragNode = row.Node()
	if a.isRoot(row.Dir) {
		a.dragNode = request.Node{} // the collection itself cannot be moved
	}
	if row.Folder {
		a.selectNode(row.Node())
		a.pressedFolder = row.Dir
		return
	}
	a.clickTree(row)
}

// requestBadge is the short label shown before a request name.
func requestBadge(protocol request.Protocol, method string) string {
	if info := protocol.Info(); info.Label != "" && info.Method != request.MethodHTTP {
		return info.Label
	}
	if method == "DELETE" {
		return "DEL"
	}
	return method
}

// Templates, parsed once above.
const (
	sidebarSource = `<col>
  <row h="{{headerH}}" pad-x="{{gutter}}" gap="{{tightGap}}" bg="bg" border-b="border">
    <button w="{{modeBtnW}}" h="{{rowH}}" on-click="switchMode" on-right-click="collectionMenu" tooltip="Collections and sent requests. Right-click: new, open, rename, save, close, delete a collection">{{mode}}</button>
    <field state="filter" h="{{rowH}}" grow="1" bg="sunken" placeholder="Filter" text-menu="filter"/>
  </row>
  <slot if="sentList" name="sentList"/>
  <scroll if="showTree" state="treeScroll" bg="sunken">
    <slot for="row in rows" name="row"/>
  </scroll>
  <!-- closing the last collection leaves this; nothing is made on disk until asked -->
  <col if="noCollection" grow="1" bg="sunken" pad="{{gutter}}" gap="{{looseGap}}">
    <text muted>No collection is open</text>
    <button h="{{rowH}}" on-click="openFolder">Open collection folder...</button>
    <button h="{{rowH}}" on-click="newCollection">New collection...</button>
  </col>
</col>
`

	treefolderSource = `<list-row selected="{{row.Selected}}" border="{{row.DropBorder}}" h="{{rowH}}" pad-l="{{row.Indent}}" gap="{{tightGap}}" on-click="row.Press" on-right-click="row.Menu" on-drag-over="row.DragOver">
  <row if="row.Stripe" w="{{treeStripeW}}" h="{{treeStripeH}}" bg="{{row.Stripe}}"/>
  <icon name="{{row.Triangle}}" w="{{treeTriangleW}}"/>
  <!-- the name, with the part the sidebar filter matched in the accent color; treerequest.vml has the same -->
  <text if="!row.Match" bold="{{row.Root}}" ellipsis grow="1">{{row.Name}}</text>
  <row if="row.Match" grow="1">
    <text bold="{{row.Root}}">{{row.Before}}</text>
    <text bold="{{row.Root}}" color="accent-line">{{row.Match}}</text>
    <text bold="{{row.Root}}" ellipsis grow="1">{{row.After}}</text>
  </row>
</list-row>
`

	treerequestSource = `<list-row selected="{{row.Selected}}" border="{{row.DropBorder}}" h="{{rowH}}" pad-l="{{row.Indent}}" pad-r="{{gutter}}" align-items="baseline" on-click="row.Press" on-right-click="row.Menu" on-drag-over="row.DragOver">
  <text small color="{{row.Color}}" w="{{badgeW}}">{{row.Badge}}</text>
  <!-- the name, with the part the sidebar filter matched in the accent color; treefolder.vml has the same -->
  <text if="!row.Match" ellipsis grow="1">{{row.Name}}</text>
  <row if="row.Match" grow="1">
    <text>{{row.Before}}</text>
    <text color="accent-line">{{row.Match}}</text>
    <text ellipsis grow="1">{{row.After}}</text>
  </row>
</list-row>
`
)
