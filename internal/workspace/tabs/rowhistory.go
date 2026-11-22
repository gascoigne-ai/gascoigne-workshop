package tabs

import "github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"

// HistoryLimit is how many table changes can be undone per table.
const HistoryLimit = 100

// RowHistory is the undo and redo stack of one table of a tab.
type RowHistory struct{ undo, redo [][]model.KV }

// History is the undo history of the table called label, made on first use.
func (tab *Tab) History(label string) *RowHistory {
	if tab.histories == nil {
		tab.histories = map[string]*RowHistory{}
	}
	if history, ok := tab.histories[label]; ok {
		return history
	}
	history := &RowHistory{}
	tab.histories[label] = history
	return history
}

func cloneRows(rows []model.KV) []model.KV { return append([]model.KV(nil), rows...) }

func sameRows(first, second []model.KV) bool {
	if len(first) != len(second) {
		return false
	}
	for i := range first {
		if first[i] != second[i] {
			return false
		}
	}
	return true
}

// Apply runs change on a copy of the rows and, when the result differs,
// stores it and records the old rows for undo. It reports whether the rows
// changed.
func (history *RowHistory) Apply(rows *[]model.KV, change func([]model.KV) []model.KV) bool {
	before := cloneRows(*rows)
	after := change(cloneRows(*rows))
	if sameRows(before, after) {
		return false
	}
	history.undo = append(history.undo, before)
	if len(history.undo) > HistoryLimit {
		history.undo = history.undo[len(history.undo)-HistoryLimit:]
	}
	history.redo = nil
	*rows = after
	return true
}

// Undo takes back the last change; Redo reapplies it. Each reports whether
// there was one.
func (history *RowHistory) Undo(rows *[]model.KV) bool {
	return step(rows, &history.undo, &history.redo)
}

// Redo reapplies the change Undo took back.
func (history *RowHistory) Redo(rows *[]model.KV) bool {
	return step(rows, &history.redo, &history.undo)
}

func step(rows *[]model.KV, from, to *[][]model.KV) bool {
	if len(*from) == 0 {
		return false
	}
	*to = append(*to, cloneRows(*rows))
	*rows = (*from)[len(*from)-1]
	*from = (*from)[:len(*from)-1]
	return true
}
