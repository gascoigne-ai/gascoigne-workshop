package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// widthsFor is the column sizes of a pane's table, created on first use.
// Each table keeps its own; request tables are named by their pane,
// response tables get a prefix because the request and response panes
// both have a Headers table.
func (a *App) widthsFor(pane pane) *widget.ColumnWidths {
	key := pane.label
	if !pane.editable {
		key = "response " + pane.label
	}
	if w, ok := a.colWidths[key]; ok {
		return w
	}
	if a.colWidths == nil {
		a.colWidths = map[string]*widget.ColumnWidths{}
	}
	w := &widget.ColumnWidths{}
	a.colWidths[key] = w
	return w
}
