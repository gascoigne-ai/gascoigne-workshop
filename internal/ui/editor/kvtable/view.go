package kvtable

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// Look is what the pane around a table decides about it.
type Look struct {
	Caption        string // the title above the rows, "" leaves the row out
	Editable       bool   // rows can be toggled, deleted, moved and edited; otherwise the table is read-only
	FlagDuplicates bool   // mark rows that repeat a name
}

// View is the table with the given padding around it, or its rows as text
// while it is in bulk-edit mode.
func (c *Component) View(ctx *w.Ctx, table logic.Table, look Look, scroll *widget.ScrollState, pad int) Node {
	if c.bulk != nil && c.bulk.Table == table {
		return c.bulkView(ctx, table, look, pad)
	}
	rows := c.rows(table)
	spec := widget.TableSpec{
		Styler:  c.deps.Styles.CellStyler(table.Tab.Req),
		State:   c.Widths(table.Kind),
		Columns: []string{"Name", "Value"},
		Weights: []float64{35, 65},
		Rows:    kvRows(*rows, false, false),
	}
	if !look.Editable {
		spec.OnRowMenu = func(row, posX, posY int) { c.deps.Menus.Open(posX, posY, copyRowItems(ctx, *rows, row, true)) }
	}
	if look.Editable {
		c.editable(ctx, &spec, table, look)
	}
	scope := c.captionScope(ctx, table, look, pad)
	scope["scroll"] = scroll
	scope["spec"] = spec
	return tableViewTemplate.MustBuild(scope)
}

// editable fills in what only an editable table has: checkboxes, the cell
// edit, the drag handles and the row menu.
func (c *Component) editable(ctx *w.Ctx, spec *widget.TableSpec, table logic.Table, look Look) {
	rows := c.rows(table)
	spec.Editable = true
	spec.Columns = []string{"Name", "Value", "Description"}
	spec.Weights = []float64{28, 40, 32}
	spec.Rows = kvRows(*rows, true, look.FlagDuplicates)
	spec.Placeholder = spec.Columns
	spec.AllChecked = request.AllEnabled(*rows)
	spec.OnToggleAll = func() {
		c.Commit()
		c.applyRows(table, func(rows []request.KV) []request.KV { return request.SetAll(rows, !request.AllEnabled(rows)) })
	}
	spec.OnToggle = func(index int) {
		c.Commit()
		c.applyRows(table, func(rows []request.KV) []request.KV {
			rows[index].Enabled = !rows[index].Enabled
			return rows
		})
	}
	spec.OnDelete = func(index int) {
		c.Commit()
		c.applyRows(table, func(rows []request.KV) []request.KV { return request.DeleteRow(rows, index) })
	}
	spec.OnGrabRow = func(row int) { c.GrabRow(table, row) }
	spec.OnDragOverRow = c.dragRowOver
	spec.OnEditCell = func(row, col int) { c.BeginEdit(table, row, col) }
	spec.OnRowMenu = func(row, posX, posY int) { c.deps.Menus.Open(posX, posY, c.rowMenuItems(ctx, table, row)) }
	if edit := c.edit; edit != nil && edit.Table == table {
		spec.Editing = &widget.TableEdit{Row: edit.Row, Col: edit.Col, Field: &edit.field, OnEnter: func() { c.Commit() }, OnTab: c.tabEdit, Options: widget.TextMenu(&edit.field)}
	}
	if sel := c.selected; sel != nil && sel.Table == table && sel.Row < len(*rows) {
		spec.Selected = &sel.Row
	}
}

// The table and its rows as text are two roots, so two templates; Go
// picks one in View.
var (
	tableViewTemplate = markup.Must("tableview.vml", tableviewSource)
	bulkViewTemplate  = markup.Must("bulkview.vml", bulkviewSource)
)

// captionScope is what both templates show in the row above the rows: the
// pane's title, and for editable tables the switch to bulk editing. An empty
// title leaves the row out.
func (c *Component) captionScope(ctx *w.Ctx, table logic.Table, look Look, pad int) markup.Scope {
	return sizes.Scope(ctx, markup.Scope{
		"pad":        pad,
		"caption":    look.Caption,
		"editable":   look.Editable,
		"bulkLabel":  logic.BulkLabel(c.bulk != nil),
		"toggleBulk": func() { c.ToggleBulk(table) },
	})
}

// bulkView shows the rows as text under the same caption.
func (c *Component) bulkView(ctx *w.Ctx, table logic.Table, look Look, pad int) Node {
	scope := c.captionScope(ctx, table, look, pad)
	scope["area"] = &c.bulk.area
	scope["areaText"] = &c.bulk.area.TextField
	return bulkViewTemplate.MustBuild(scope)
}

// Templates, parsed once above.
const (
	tableviewSource = `<scroll state="scroll" pad="{{pad}}" gap="{{tightGap}}">
  <!-- the caption row is copied in bulkview.vml, which keeps it above the rows as text -->
  <row if="caption" h="{{rowH}}">
    <caption>{{caption}}</caption>
    <spacer/>
    <link-button if="editable" tooltip="Edit all rows as text, one per line" on-click="toggleBulk">{{bulkLabel}}</link-button>
  </row>
  <table spec="spec"/>
</scroll>
`

	bulkviewSource = `<col grow="1" pad="{{pad}}" gap="{{tightGap}}">
  <!-- the caption row is copied from tableview.vml, so switching to text keeps the caption in place -->
  <row if="caption" h="{{rowH}}">
    <caption>{{caption}}</caption>
    <spacer/>
    <link-button if="editable" tooltip="Edit all rows as text, one per line" on-click="toggleBulk">{{bulkLabel}}</link-button>
  </row>
  <area state="area" grow="1" bg="sunken" text-menu="areaText"/>
</col>
`
)
