package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	// The view DSL is dot-imported so views read like markup: Row(...), Text(...).
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func (a *App) tablePane(panel pane, tab *reqTab, scroll *ScrollState) Node {
	return a.tableView(panel, tab, scroll, w.Gutter)
}

// tableView is a pane's table with the given padding around it.
func (a *App) tableView(panel pane, tab *reqTab, scroll *ScrollState, pad int) Node {
	if b := a.bulk; b != nil && b.tab == tab && b.pane.label == panel.label {
		return a.bulkView(panel, tab, pad)
	}
	rows := panel.rows(a, tab)
	spec := TableSpec{
		Styler:  a.cellStyler(tab.Req),
		State:   a.widthsFor(panel),
		Columns: []string{"Name", "Value"},
		Weights: []float64{35, 65},
		Rows:    kvRows(*rows, false, false),
	}
	if !panel.editable {
		spec.OnRowMenu = func(row, posX, posY int) { a.openMenu(posX, posY, a.copyRowItems(*rows, row, true)) }
	}
	if panel.editable {
		spec.Editable = true
		spec.Columns = []string{"Name", "Value", "Description"}
		spec.Weights = []float64{28, 40, 32}
		spec.Rows = kvRows(*rows, true, panel.flagDuplicates)
		spec.Placeholder = spec.Columns
		spec.AllChecked = model.AllEnabled(*rows)
		spec.OnToggleAll = func() {
			a.commitEdit()
			a.applyRows(panel, tab, func(rows []model.KV) []model.KV { return model.SetAll(rows, !model.AllEnabled(rows)) })
		}
		spec.OnToggle = func(index int) {
			a.commitEdit()
			a.applyRows(panel, tab, func(rows []model.KV) []model.KV {
				rows[index].Enabled = !rows[index].Enabled
				return rows
			})
		}
		spec.OnDelete = func(index int) {
			a.commitEdit()
			a.applyRows(panel, tab, func(rows []model.KV) []model.KV { return model.DeleteRow(rows, index) })
		}
		spec.OnGrabRow = func(row int) { a.grabRow(panel, tab, row) }
		spec.OnDragOverRow = func(row int) { a.dragRowOver(row) }
		spec.OnEditCell = func(row, col int) { a.beginEdit(panel, tab, row, col) }
		spec.OnRowMenu = func(row, posX, posY int) { a.openMenu(posX, posY, a.rowMenuItems(panel, tab, row)) }
		if e := a.edit; e != nil && e.pane.label == panel.label && e.tab == tab {
			spec.Editing = &TableEdit{Row: e.row, Col: e.col, Field: &e.field, OnEnter: a.commitEdit, OnTab: a.tabEdit, Options: TextMenu(&e.field)}
		}
		if sel := a.rowSel; sel != nil && sel.tab == tab && sel.pane.label == panel.label && sel.row < len(*rows) {
			spec.Selected = &sel.row
		}
	}
	scope := a.tableCaptionScope(panel, tab, pad)
	scope["scroll"] = scroll
	scope["spec"] = spec
	return tableViewTemplate.MustBuild(scope)
}

// The table and its rows as text are two roots, so two templates; Go
// picks one in tableView.
var (
	tableViewTemplate = markup.Must("tableview.vml", tableviewSource)
	bulkViewTemplate  = markup.Must("bulkview.vml", bulkviewSource)
)

// tableCaptionScope is what both templates show in the row above the
// rows: the pane's title, and for editable tables the switch to bulk
// editing. An empty title leaves the row out.
func (a *App) tableCaptionScope(panel pane, tab *reqTab, pad int) markup.Scope {
	return markup.Scope{
		"pad":        pad,
		"rowH":       w.RowH,
		"tightGap":   w.TightGap,
		"caption":    panel.captionOf(a),
		"editable":   panel.editable,
		"bulkLabel":  bulkLabel(a.bulk != nil),
		"toggleBulk": func() { a.toggleBulk(panel, tab) },
	}
}

func bulkLabel(inBulk bool) string {
	if inBulk {
		return "Table"
	}
	return "Bulk edit"
}

// bulkView shows the rows as text under the same caption.
func (a *App) bulkView(panel pane, tab *reqTab, pad int) Node {
	scope := a.tableCaptionScope(panel, tab, pad)
	scope["area"] = &a.bulk.area
	scope["areaText"] = &a.bulk.area.TextField
	return bulkViewTemplate.MustBuild(scope)
}

func (a *App) rowsChanged(panel pane, tab *reqTab) {
	if panel.changed != nil {
		panel.changed(tab)
	}
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
