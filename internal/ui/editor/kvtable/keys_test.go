package kvtable

import (
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestKeysClaimUndoAndRedoOfTheVisibleTable(t *testing.T) {
	cases := []struct {
		name        string
		in          core.Input
		setup       func(w *world)
		textFocused bool
		claimed     bool
		names       []string // the Params after the key
	}{
		{"undo", core.Input{Undo: true}, nil, false, true, []string{"q", "order", "sort"}},
		{"redo", core.Input{Redo: true}, func(w *world) { w.kv.undo(w.params) }, false, true, []string{"order", "sort"}},
		{"while a cell is edited", core.Input{Undo: true}, func(w *world) { w.kv.BeginEdit(w.params, 0, 0) }, false, false, []string{"order", "sort"}},
		{"while typing in the bulk text", core.Input{Undo: true}, func(w *world) {
			w.kv.ToggleBulk(w.params)
			w.kv.BulkArea().Focused = true
		}, false, false, []string{"order", "sort"}},
		{"while another text field has the keyboard", core.Input{Undo: true}, nil, true, false, []string{"order", "sort"}},
		{"with no table showing", core.Input{Undo: true}, func(w *world) { w.fake.shown = false }, false, false, []string{"order", "sort"}},
		{"another key", core.Input{}, nil, false, false, []string{"order", "sort"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := newWorld(t)
			w.kv.applyRows(w.params, func(rows []request.KV) []request.KV { return request.DeleteRow(rows, 0) })
			if tc.setup != nil {
				tc.setup(w)
			}
			if got := w.kv.Keys(tc.in, tc.textFocused); got != tc.claimed {
				t.Fatalf("claimed %v, want %v", got, tc.claimed)
			}
			if got := w.names(); !reflect.DeepEqual(got, tc.names) {
				t.Fatalf("params %v, want %v", got, tc.names)
			}
		})
	}
}

func TestTickDropsAClosedTabsEditBeforeCommittingOnBlur(t *testing.T) {
	w := newWorld(t)
	w.kv.BeginEdit(w.params, 0, 1)
	w.typeInto("zzz")
	w.kv.edit.field.Focused = false // would be committed if the tab were open
	w.fake.open = nil
	if w.kv.Tick() {
		t.Error("a dropped edit changes nothing on screen")
	}
	if w.kv.Editing() || w.tab.Req.Params[0].Value != "gioui" || len(w.fake.changed) != 0 {
		t.Fatalf("editing %v value %q changes %v: the text of a closed tab is dropped, not written", w.kv.Editing(), w.tab.Req.Params[0].Value, w.fake.changed)
	}
}

func TestTickCommitsAnEditThatLostFocus(t *testing.T) {
	w := newWorld(t)
	w.kv.BeginEdit(w.params, 0, 1)
	w.typeInto("go")
	w.kv.edit.field.Focused = false
	if !w.kv.Tick() {
		t.Error("a commit is a change")
	}
	if w.kv.Editing() || w.tab.Req.Params[0].Value != "go" {
		t.Fatalf("editing %v value %q", w.kv.Editing(), w.tab.Req.Params[0].Value)
	}
}

func TestTickLeavesAFocusedEditAlone(t *testing.T) {
	w := newWorld(t)
	w.kv.BeginEdit(w.params, 0, 1)
	if w.kv.Tick() || !w.kv.Editing() {
		t.Fatal("a focused edit goes on")
	}
}

func TestTickAppliesBulkWhenItsTableIsHidden(t *testing.T) {
	cases := []struct {
		name  string
		setup func(w *world)
		want  bool // the text is applied
	}{
		{"the table is still visible", func(w *world) {}, false},
		{"another pane shows", func(w *world) { w.fake.shown = false }, true},
		{"another table shows", func(w *world) { w.fake.visible = logic.Table{Tab: w.tab, Kind: logic.Headers} }, true},
		{"the tab closed", func(w *world) { w.fake.open, w.fake.shown = nil, false }, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := newWorld(t)
			w.kv.ToggleBulk(w.params)
			w.kv.BulkArea().SelectAll()
			w.kv.BulkArea().Insert("only: one")
			tc.setup(w)
			if got := w.kv.Tick(); got != tc.want {
				t.Errorf("Tick = %v, want %v", got, tc.want)
			}
			if applied := reflect.DeepEqual(w.names(), []string{"only"}); applied != tc.want || (w.kv.BulkArea() == nil) != tc.want {
				t.Fatalf("params %v, bulk open %v", w.names(), w.kv.BulkArea() != nil)
			}
		})
	}
}

func TestTickDropsTheSelectionAndDragOfAClosedTab(t *testing.T) {
	w := newWorld(t)
	w.kv.GrabRow(w.params, 1)
	w.fake.open = nil
	w.kv.Tick()
	if w.kv.Dragging() || w.kv.selected != nil {
		t.Fatalf("dragging %v selected %v", w.kv.Dragging(), w.kv.selected)
	}
}

// Each row of the activity table in the package doc: an entry point commits
// the open edit first.
func TestEveryEntryPointCommitsAnOpenEditFirst(t *testing.T) {
	entries := map[string]func(w *world){
		"BeginEdit":  func(w *world) { w.kv.BeginEdit(w.params, 1, 0) },
		"GrabRow":    func(w *world) { w.kv.GrabRow(w.params, 1) },
		"ToggleBulk": func(w *world) { w.kv.ToggleBulk(w.params) },
	}
	for name, enter := range entries {
		t.Run(name, func(t *testing.T) {
			w := newWorld(t)
			w.kv.BeginEdit(w.params, 0, 1)
			w.typeInto("first")
			enter(w)
			if w.tab.Req.Params[0].Value != "first" {
				t.Fatalf("the open edit was not committed: %+v", w.tab.Req.Params[0])
			}
		})
	}
}

func TestRowOperationsActOnTheEditedRowElseTheSelectedOne(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(w *world)
		do      func(w *world)
		names   []string
		editRow int // -1: no edit
		selRow  int // -1: no selection
	}{
		{"insert below an edit", func(w *world) { w.kv.BeginEdit(w.params, 1, 1) }, func(w *world) { w.kv.InsertRowBelow() },
			[]string{"q", "order", "", "sort"}, 2, 2},
		{"duplicate the edited row", func(w *world) { w.kv.BeginEdit(w.params, 0, 0) }, func(w *world) { w.kv.DuplicateRow() },
			[]string{"q", "q", "order", "sort"}, 1, 1},
		{"duplicate the selected row", func(w *world) { w.kv.Select(w.params, 2) }, func(w *world) { w.kv.DuplicateRow() },
			[]string{"q", "order", "sort", "sort"}, -1, 3},
		{"move the edited row down", func(w *world) { w.kv.BeginEdit(w.params, 0, 0) }, func(w *world) { w.kv.MoveRow(1) },
			[]string{"order", "q", "sort"}, 1, 1},
		{"move the selected row up", func(w *world) { w.kv.Select(w.params, 2) }, func(w *world) { w.kv.MoveRow(-1) },
			[]string{"q", "sort", "order"}, -1, 1},
		{"move the first row up stays", func(w *world) { w.kv.Select(w.params, 0) }, func(w *world) { w.kv.MoveRow(-1) },
			[]string{"q", "order", "sort"}, -1, 0},
		{"delete the selected row selects the next", func(w *world) { w.kv.Select(w.params, 1) }, func(w *world) { w.kv.DeleteRow() },
			[]string{"q", "sort"}, -1, 1},
		{"delete the last row selects the new last", func(w *world) { w.kv.Select(w.params, 2) }, func(w *world) { w.kv.DeleteRow() },
			[]string{"q", "order"}, -1, 1},
		{"nothing selected does nothing", func(w *world) {}, func(w *world) { w.kv.DeleteRow() },
			[]string{"q", "order", "sort"}, -1, -1},
		{"a selection of a hidden table does nothing", func(w *world) {
			w.kv.Select(w.params, 1)
			w.fake.shown = false
		}, func(w *world) { w.kv.DeleteRow() }, []string{"q", "order", "sort"}, -1, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := newWorld(t)
			tc.setup(w)
			tc.do(w)
			if got := w.names(); !reflect.DeepEqual(got, tc.names) {
				t.Fatalf("params %v, want %v", got, tc.names)
			}
			edit, editing := w.kv.EditCell()
			if tc.editRow < 0 && editing || tc.editRow >= 0 && (!editing || edit.Row != tc.editRow) {
				t.Errorf("edit %+v %v, want row %d", edit, editing, tc.editRow)
			}
			if tc.selRow < 0 && w.kv.selected != nil || tc.selRow >= 0 && (w.kv.selected == nil || w.kv.selected.Row != tc.selRow) {
				t.Errorf("selected %+v, want row %d", w.kv.selected, tc.selRow)
			}
		})
	}
}

func TestAnEditedRowOfAnotherTableIsTheTargetEvenWhenNothingShows(t *testing.T) {
	w := newWorld(t)
	w.kv.BeginEdit(w.params, 1, 0)
	w.fake.shown = false
	w.kv.DuplicateRow()
	if got := w.names(); !reflect.DeepEqual(got, []string{"q", "order", "order", "sort"}) {
		t.Fatalf("params %v", got)
	}
}
