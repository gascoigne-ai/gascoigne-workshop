package tabs

import (
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
)

func rows(names ...string) []model.KV {
	var out []model.KV
	for _, name := range names {
		out = append(out, model.KV{Name: name, Enabled: true})
	}
	return out
}

func add(name string) func([]model.KV) []model.KV {
	return func(current []model.KV) []model.KV { return append(current, rows(name)...) }
}

func TestApplyChangesRowsAndUndoRedoWalkBack(t *testing.T) {
	var history RowHistory
	table := rows("a")
	if !history.Apply(&table, add("b")) || !reflect.DeepEqual(table, rows("a", "b")) {
		t.Fatalf("after apply: %v", table)
	}
	if !history.Undo(&table) || !reflect.DeepEqual(table, rows("a")) {
		t.Fatalf("after undo: %v", table)
	}
	if !history.Redo(&table) || !reflect.DeepEqual(table, rows("a", "b")) {
		t.Fatalf("after redo: %v", table)
	}
	if history.Redo(&table) {
		t.Fatal("nothing left to redo")
	}
}

func TestAChangeThatChangesNothingIsNotRecorded(t *testing.T) {
	var history RowHistory
	table := rows("a")
	if history.Apply(&table, func(current []model.KV) []model.KV { return current }) {
		t.Fatal("an identical result is not a change")
	}
	if history.Undo(&table) {
		t.Fatal("nothing to undo")
	}
}

func TestANewChangeDropsTheRedoStack(t *testing.T) {
	var history RowHistory
	table := rows("a")
	history.Apply(&table, add("b"))
	history.Undo(&table)
	history.Apply(&table, add("c"))
	if history.Redo(&table) {
		t.Fatal("redo must be gone after a new change")
	}
}

func TestHistoryKeepsOnlyTheLastHundredChanges(t *testing.T) {
	var history RowHistory
	var table []model.KV
	for range HistoryLimit + 20 {
		history.Apply(&table, add("x"))
	}
	undone := 0
	for history.Undo(&table) {
		undone++
	}
	if undone != HistoryLimit {
		t.Fatalf("undone %d, want %d", undone, HistoryLimit)
	}
}

func TestTabKeepsOneHistoryPerTable(t *testing.T) {
	tab := New(newRequest())
	if tab.History("Headers") != tab.History("Headers") || tab.History("Headers") == tab.History("Params") {
		t.Fatal("one history per table label")
	}
}
