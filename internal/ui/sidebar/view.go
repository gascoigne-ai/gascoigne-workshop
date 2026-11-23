package sidebar

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sidebar/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

var (
	sidebarTemplate     = markup.Must("sidebar.vml", sidebarSource)
	treeFolderTemplate  = markup.Must("treefolder.vml", treefolderSource)
	treeRequestTemplate = markup.Must("treerequest.vml", treerequestSource)
)

// View is the header (the chip that switches between the collection
// tree and the log of sent requests, and the filter) above the one shown.
//
// The tree builds only the rows in view plus one, so a drawn frame costs
// what a screen of rows costs, not the whole tree. viewH is the height of
// the box that shows them, an upper bound is enough (the window's height:
// built rows are then never too few), and 0 builds every row. Tick tells
// the window when a scroll moved the range, so it builds again before it
// paints. Rejected: reading the box height from the toolkit's scroll state,
// which would change the toolkit for one list.
func (c *Component) View(ctx *w.Ctx, viewH int) Node {
	label, switchTo := "Collections", func() { c.showSent = true }
	if c.showSent {
		label, switchTo = "Sent", func() { c.showSent = false }
	}
	var sentList Node
	var rows []Node
	var pad treePad
	c.shown.geometry, c.shown.built = logic.RowGeometry{}, logic.RowRange{}
	if c.showSent {
		sentList = c.sent.List(ctx, c.deps.Sent, c.filter.Text)
	} else {
		rows, pad = c.treeRows(ctx, viewH)
	}
	entries := len(c.deps.Collections.Set().Entries)
	return sizes.Build(ctx, sidebarTemplate, markup.Scope{
		"mode":           label,
		"switchMode":     switchTo,
		"collectionMenu": func(posX, posY int) { c.deps.Menus.Open(posX, posY, c.CollectionMenu()) },
		"filter":         &c.filter,
		"showSent":       c.showSent,
		"showTree":       !c.showSent && entries > 0,
		"noCollection":   !c.showSent && entries == 0,
		"openFolder":     c.deps.Workspace.OpenFolder,
		"newCollection":  c.deps.Workspace.NewCollection,
		"sentList":       sentList,
		"treeScroll":     &c.scroll,
		"rows":           rows,
		"spaceAbove":     pad.top > 0,
		"padTop":         pad.top,
		"spaceBelow":     pad.bottom > 0,
		"padBottom":      pad.bottom,
		"headerH":        sizes.HeaderH,
	})
}

// treePad is the height of the rows not built above and below the ones built.
type treePad struct{ top, bottom int }

// treeRows builds the rows of the tree that are in view in a box viewH high,
// and the padding that stands for the others, so the scroll box keeps the
// height of the whole tree. It records what it built in c.shown.
func (c *Component) treeRows(ctx *w.Ctx, viewH int) ([]Node, treePad) {
	visible := c.VisibleRows()
	geometry := logic.RowGeometry{Count: len(visible), RowH: ctx.Sizes.Row, ViewH: viewH}
	built := geometry.Range(c.scroll.Y)
	c.shown.geometry, c.shown.built = geometry, built
	rows := make([]Node, 0, built.Last-built.First)
	for _, row := range visible[built.First:built.Last] {
		rows = append(rows, c.treeRow(ctx, row))
	}
	return rows, treePad{top: built.First * geometry.RowH, bottom: (geometry.Count - built.Last) * geometry.RowH}
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
func (c *Component) treeRow(ctx *w.Ctx, row collection.Row) Node {
	item := c.treeItemOf(ctx, row)
	item.Selected = row.Node() == c.tree.Selected()
	template := treeFolderTemplate
	if !row.Folder {
		template = treeRequestTemplate
	}
	return sizes.Build(ctx, template, markup.Scope{"row": item, "badgeW": sizes.BadgeSlot(ctx, sizes.BadgeW)})
}

// treeItemOf is the view model of one tree row, except whether it is selected.
func (c *Component) treeItemOf(ctx *w.Ctx, row collection.Row) treeItem {
	node := row.Node()
	item := treeItem{
		Name:     row.Name,
		Indent:   w.Gutter + row.Depth*sizes.TreeIndent,
		Press:    func() { c.press(row) },
		Menu:     func(pointerX, pointerY int) { c.deps.Menus.Open(pointerX, pointerY, c.RowMenu(row)) },
		DragOver: func() { c.tree.Over(node) },
	}
	item.Before, item.Match, item.After = collection.SplitMatch(row.Name, c.filter.Text)
	if row.Folder {
		item.Root = c.deps.Collections.Set().IsRoot(row.Dir)
		if item.Root {
			item.Stripe = c.stripe(node)
		}
		item.Triangle = "triangle-right"
		if row.Open {
			item.Triangle = "triangle-down"
		}
	} else {
		item.Indent += 6
		item.Badge = theme.RequestBadge(row.Req.Protocol, row.Req.Method)
		item.Color = theme.MethodToken(row.Req.Method)
	}
	if drag := c.tree.Drag(); drag.Phase() == logic.Dragging && drag.Target == node {
		item.DropBorder = w.AccentLine
		if ctx.In.Alt { // released now, it copies: the outline turns green
			item.DropBorder = w.OK
		}
	}
	return item
}

// press is a press on a row: the tree selects it, takes the keyboard and
// starts a drag (logic.Tree.Press); a request opens at once, a folder
// toggles when the button is released on the same row, so an open folder
// can still be dragged.
func (c *Component) press(row collection.Row) {
	c.tree.Press(row, c.deps.Collections.Set().IsRoot(row.Dir))
	if !row.Folder {
		c.deps.Requests.Show(row.Req)
	}
}

// stripe is the color of the open collection that holds node, "" for a
// node of none.
func (c *Component) stripe(node request.Node) w.Token {
	for index, entry := range c.deps.Collections.Set().Entries {
		if entry.Collection.Folder.Contains(node) {
			return theme.CollectionToken(index)
		}
	}
	return ""
}

// activateSelected is Enter on the selected row of rows: a folder toggles,
// a request is shown.
func (c *Component) activateSelected(rows []collection.Row) {
	for _, row := range rows {
		if row.Node() != c.tree.Selected() {
			continue
		}
		if row.Folder {
			c.tree.Toggle(row.Dir)
			return
		}
		c.deps.Requests.Show(row.Req)
		return
	}
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
    <row if="spaceAbove" h="{{padTop}}"/>
    <slot for="row in rows" name="row"/>
    <row if="spaceBelow" h="{{padBottom}}"/>
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
