// Package kvtable is the key-value table of the request pane: the Params,
// Headers, Vars and form-field tables, and the read-only Headers table of
// the response. It draws the table (or its rows as text, "bulk edit"), edits
// a cell, moves and drags rows, and claims Ctrl+Z and Ctrl+Y for the table
// that is showing. Where the rows live, what a change to them means (the URL
// follows the Params) and the menus are asked of the application through
// Deps.
//
// State is one slot each, not one per tab: one cell edit, one bulk edit, one
// selected row and one dragged row, each remembering its table (and so its
// tab), and one column-width record per Kind shared by every tab. That is
// what the window did before the table became a component and the move
// keeps it, so the user sees no change. A map per tab was rejected because
// it would allow an edit open in two tabs at once; revisit it when a request
// pane can show two tables side by side. A record of a closed tab is dropped
// by Tick, in the order Tick documents, and a bulk text of a closed tab is
// applied first, so nothing typed is lost.
//
// Undo history is not here: it stays on tabs.Tab (History, HistoryLimit),
// keyed by the table's label. Because the Vars rows are shared by every tab
// while their history is per tab, undoing in one tab can overwrite a change
// made through another; that is known and left alone (a task of its own).
//
// The response Headers table (Kind ResponseHeaders) is the same component in
// its read-only mode: no checkboxes, no edit, only the copy menu. It stays
// here until the response pane takes it over (task 037); it is about twenty
// lines of View.
//
// The form table is drawn inside the Body pane: it reaches the pane through
// body.Deps.Slots.FormTable, which the parent answers, since siblings never
// import each other.
//
// At most one activity runs at a time. Editing starts with BeginEdit, which
// commits any edit first, and ends by Commit, CancelEdit, the field losing
// focus (Tick commits), the rows shrinking past it, or its tab closing
// (dropped, the text is lost). Dragging starts with GrabRow, which commits
// an edit first, and ends on the button release (BeginFrame). Bulk starts
// with ToggleBulk, which commits an edit first, and ends by toggling again
// or when its table is no longer visible (Tick applies it, even for a closed
// tab). The selection is independent: it is dropped when its row is gone or
// its tab closes. A sum type over these activities was rejected because it
// changes the model; revisit it when a fourth activity appears or a bug
// shows two at once.
package kvtable

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// cellEdit is the cell being edited and the field holding the text.
type cellEdit struct {
	logic.Cell
	field widget.TextField
}

// bulkEdit is a table shown as text, one "name: value" line per row.
type bulkEdit struct {
	logic.Table
	area widget.TextArea
}

// Component draws the tables and keeps their editing state.
type Component struct {
	deps     Deps
	edit     *cellEdit                          // the cell being edited, nil when none
	bulk     *bulkEdit                          // the table in bulk-edit mode, nil when none
	selected *logic.Cell                        // the row the keyboard acts on
	dragged  *logic.Cell                        // the row being dragged by its grip
	widths   [logic.KindCount]widget.TableState // column sizes, one per table, weights nil for the default
}

// New returns a component asking deps.
func New(deps Deps) *Component { return &Component{deps: deps} }

// Widths is the column sizes of the table of kind. The request's and the
// response's Headers tables have each their own.
func (c *Component) Widths(kind logic.Kind) *widget.TableState { return &c.widths[kind] }

// rows are the rows of table.
func (c *Component) rows(table logic.Table) *[]request.KV { return c.deps.Tables.Rows(table) }

// Select makes row of table the one the keyboard acts on.
func (c *Component) Select(table logic.Table, row int) {
	c.selected = &logic.Cell{Table: table, Row: row}
}

// EditCell is the cell being edited, if any.
func (c *Component) EditCell() (logic.Cell, bool) {
	edit := c.liveEdit()
	if edit == nil {
		return logic.Cell{}, false
	}
	return edit.Cell, true
}

// BulkArea is the text of the table in bulk-edit mode, nil when none.
func (c *Component) BulkArea() *widget.TextArea {
	if c.bulk == nil {
		return nil
	}
	return &c.bulk.area
}
