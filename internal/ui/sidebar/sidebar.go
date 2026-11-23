// Package sidebar is the left pane: the header with the mode chip and the
// filter, and below it the collections tree or the log of sent requests.
// It owns the tree state (selection, open folders, drag, scroll, filter);
// what changes a collection is asked of the application through Deps.
package sidebar

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sidebar/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sidebar/sentlog"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Component draws the sidebar and keeps its state.
type Component struct {
	deps     Deps
	tree     *logic.Tree
	filter   widget.TextField // the filter over the tree or the sent log
	sent     sentlog.Log      // requests sent in this workspace, newest first
	showSent bool             // the sent log is listed instead of the tree
	scroll   widget.ScrollState

	// shown is what the last View built of the tree: the geometry it cut
	// the rows by and the range it built. Tick compares the scroll position
	// with it, since a wheel step or a thumb drag moves the scroll without
	// a rebuild and the rows in view then are not the rows built. The zero
	// value, kept in sent mode and with no collection, never moves.
	//
	// It is written by View, not derived in Tick: Tick runs every frame and
	// must not walk the rows again. Rejected: a ViewH or H field on the
	// toolkit's ScrollState (a toolkit change for one list), a SetViewH from
	// setSize (misses SetSize and the first frame), and a map from node to
	// built Node (invalidation follows select, drag-over, filter and Alt).
	// Revisit when VisibleRows or FilteredRows shows in a profile at 100x
	// the rows (then cache the row list), or when rows stop having one
	// height (then a prefix sum of heights).
	shown struct {
		geometry logic.RowGeometry
		built    logic.RowRange
	}
}

// moved reports whether the rows in view at scrollY are not the rows built.
func (c *Component) moved(scrollY int) bool {
	return c.shown.geometry.Range(scrollY) != c.shown.built
}

// New returns a sidebar asking deps.
func New(deps Deps) *Component { return &Component{deps: deps, tree: logic.New()} }

// Tree is the selection, open folders and drag of the tree.
func (c *Component) Tree() *logic.Tree { return c.tree }

// Filter is the sidebar's filter field.
func (c *Component) Filter() *widget.TextField { return &c.filter }

// Typing reports whether the filter has the keyboard.
func (c *Component) Typing() bool { return c.filter.Focused }

// Sent is the log of sent requests.
func (c *Component) Sent() *sentlog.Log { return &c.sent }

// ShowSent reports whether the sent log is listed instead of the tree.
func (c *Component) ShowSent() bool { return c.showSent }

// SetShowSent chooses the sent log or the tree.
func (c *Component) SetShowSent(on bool) { c.showSent = on }

// ScrollY is how far the tree is scrolled down.
func (c *Component) ScrollY() int { return c.scroll.Y }

// VisibleRows are the tree rows shown, after the filter, of every open
// collection in order.
func (c *Component) VisibleRows() []collection.Row {
	var rows []collection.Row
	for _, entry := range c.deps.Collections.Set().Entries {
		rows = append(rows, collection.FilteredRows(&entry.Collection.Folder, c.tree.Expanded(), c.filter.Text)...)
	}
	return rows
}

// ownerOf is the open collection that holds node, nil for none.
func (c *Component) ownerOf(node request.Node) *request.Collection {
	if node.None() {
		return nil
	}
	entry, _ := c.deps.Collections.Set().Owner(node)
	return entry.Collection
}
