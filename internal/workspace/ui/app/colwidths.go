package ui

import "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"

// widthsFor is the column sizes of a pane's table, created on first use.
// Each table keeps its own; request tables are named by their pane,
// response tables get a prefix because the request and response panes
// both have a Headers table.
func (a *App) widthsFor(p pane) *view.ColumnWidths {
	key := p.label
	if !p.editable {
		key = "response " + p.label
	}
	if w, ok := a.colWidths[key]; ok {
		return w
	}
	if a.colWidths == nil {
		a.colWidths = map[string]*view.ColumnWidths{}
	}
	w := &view.ColumnWidths{}
	a.colWidths[key] = w
	return w
}
