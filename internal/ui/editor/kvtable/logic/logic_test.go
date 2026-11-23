package logic

import (
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
)

func TestKindLabels(t *testing.T) {
	cases := []struct {
		kind       Kind
		label, key string
	}{
		{Params, "Params", "Params"},
		{Headers, "Headers", "Headers"},
		{Vars, "Vars", "Vars"},
		{Form, "Body", "Body"},
		{ResponseHeaders, "Headers", "response Headers"},
	}
	for _, tc := range cases {
		if tc.kind.Label() != tc.label || tc.kind.Key() != tc.key {
			t.Errorf("kind %d: label %q key %q, want %q and %q", tc.kind, tc.kind.Label(), tc.kind.Key(), tc.label, tc.key)
		}
		if back, ok := KindOfKey(tc.key); !ok || back != tc.kind {
			t.Errorf("KindOfKey(%q) = %d, %v", tc.key, back, ok)
		}
	}
	if _, ok := KindOfKey("nonsense"); ok {
		t.Error("an unknown key is no table")
	}
	if got := ResponseHeaders.Key(); got == Headers.Key() {
		t.Error("the response Headers table has its own widths slot")
	}
}

var (
	tabA = &tabs.Tab{}
	tabB = &tabs.Tab{}
)

func cellAt(tab *tabs.Tab, kind Kind, row, col int) Cell {
	return Cell{Table: Table{Tab: tab, Kind: kind}, Row: row, Col: col}
}

func TestNextCellWrapsAcrossRowsAndStopsAtTheEnds(t *testing.T) {
	cases := []struct {
		name     string
		from     Cell
		back     bool
		rows     int
		row, col int
	}{
		{"forward in a row", cellAt(tabA, Params, 0, 0), false, 3, 0, 1},
		{"past the last column to the next row", cellAt(tabA, Params, 0, 2), false, 3, 1, 0},
		{"back from the first column to the previous row", cellAt(tabA, Params, 1, 0), true, 3, 0, 2},
		{"back at the very start stays", cellAt(tabA, Params, 0, 0), true, 3, 0, 0},
		{"past the placeholder stops in it", cellAt(tabA, Params, 3, 2), false, 3, 3, 2},
		{"into the placeholder row", cellAt(tabA, Params, 2, 2), false, 3, 3, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NextCell(tc.from, tc.back, tc.rows)
			if got.Row != tc.row || got.Col != tc.col || got.Table != tc.from.Table {
				t.Fatalf("got row %d col %d (%v), want %d, %d", got.Row, got.Col, got.Table == tc.from.Table, tc.row, tc.col)
			}
		})
	}
}

func TestSetCellWritesTheRowOrAddsOneFromThePlaceholder(t *testing.T) {
	rows := func() []request.KV { return []request.KV{{Name: "a", Value: "1", Enabled: true}} }
	cases := []struct {
		name     string
		row, col int
		text     string
		want     []request.KV
	}{
		{"an existing row", 0, 1, "2", []request.KV{{Name: "a", Value: "2", Enabled: true}}},
		{"the description", 0, 2, "why", []request.KV{{Name: "a", Value: "1", Description: "why", Enabled: true}}},
		{"the placeholder with text adds an enabled row", 1, 0, "b", []request.KV{{Name: "a", Value: "1", Enabled: true}, {Name: "b", Enabled: true}}},
		{"an empty placeholder adds nothing", 1, 0, "", rows()},
		{"a row past the end after deletions adds one row", 5, 1, "late", []request.KV{{Name: "a", Value: "1", Enabled: true}, {Value: "late", Enabled: true}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SetCell(rows(), tc.row, tc.col, tc.text); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestCellTextOfThePlaceholderRowIsEmpty(t *testing.T) {
	rows := []request.KV{{Name: "a", Value: "1", Description: "d"}}
	for col, want := range []string{"a", "1", "d"} {
		if got := CellText(rows, 0, col); got != want {
			t.Errorf("col %d = %q, want %q", col, got, want)
		}
	}
	if got := CellText(rows, 1, 0); got != "" {
		t.Errorf("the placeholder row = %q", got)
	}
}

func TestSettleDropsAnEditOrSelectionPastTheRows(t *testing.T) {
	params := Table{Tab: tabA, Kind: Params}
	at := func(row int) *Cell { c := cellAt(tabA, Params, row, 0); return &c }
	other := func(row int) *Cell { c := cellAt(tabA, Headers, row, 0); return &c }
	cases := []struct {
		name              string
		edit, selected    *Cell
		keepEdit, keepSel bool
	}{
		{"edit in the placeholder row stays", at(3), nil, true, false},
		{"edit past the placeholder goes", at(4), nil, false, false},
		{"selection at the row count goes", nil, at(3), false, false},
		{"selection inside stays", nil, at(2), false, true},
		{"another table is untouched", other(9), other(9), true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			edit, selected := Settle(tc.edit, tc.selected, params, 3)
			if (edit != nil) != tc.keepEdit || (selected != nil) != tc.keepSel {
				t.Fatalf("edit kept %v, selection kept %v", edit != nil, selected != nil)
			}
		})
	}
}

func TestTargetIsTheEditedRowElseTheVisibleSelection(t *testing.T) {
	visible := Table{Tab: tabA, Kind: Params}
	rows := func(Table) int { return 3 }
	at := func(tab *tabs.Tab, kind Kind, row, col int) *Cell { c := cellAt(tab, kind, row, col); return &c }
	cases := []struct {
		name           string
		edit, selected *Cell
		shown          bool
		want           Cell
		editing, ok    bool
	}{
		{"a valid edit", at(tabA, Params, 1, 2), nil, true, cellAt(tabA, Params, 1, 2), true, true},
		{"an edit past the rows gives none, not the selection", at(tabA, Params, 3, 0), at(tabA, Params, 0, 0), true, Cell{}, false, false},
		{"the selection of the visible table", nil, at(tabA, Params, 2, 0), true, cellAt(tabA, Params, 2, 0), false, true},
		{"a selection of another table", nil, at(tabA, Headers, 0, 0), true, Cell{}, false, false},
		{"a selection of another tab", nil, at(tabB, Params, 0, 0), true, Cell{}, false, false},
		{"a selection past the rows", nil, at(tabA, Params, 3, 0), true, Cell{}, false, false},
		{"no table showing", nil, at(tabA, Params, 0, 0), false, Cell{}, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, editing, ok := Target(tc.edit, tc.selected, visible, tc.shown, rows)
			if got != tc.want || editing != tc.editing || ok != tc.ok {
				t.Fatalf("got %+v editing %v ok %v, want %+v %v %v", got, editing, ok, tc.want, tc.editing, tc.ok)
			}
		})
	}
}

func TestSelectionAfterDeleteStaysInRange(t *testing.T) {
	cases := []struct {
		name      string
		row, rows int
		want      int
		ok        bool
	}{
		{"a middle row keeps its place", 1, 3, 1, true},
		{"the last row moves up", 3, 3, 2, true},
		{"the only row leaves nothing", 0, 0, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got, ok := AfterDelete(tc.row, tc.rows); got != tc.want || ok != tc.ok {
				t.Fatalf("got %d, %v", got, ok)
			}
		})
	}
}

func TestSecretLabel(t *testing.T) {
	if SecretLabel(false) != "Mark as secret" || SecretLabel(true) != "Not a secret" {
		t.Fatalf("labels %q %q", SecretLabel(false), SecretLabel(true))
	}
}

func TestBulkLabel(t *testing.T) {
	if BulkLabel(false) != "Bulk edit" || BulkLabel(true) != "Table" {
		t.Fatalf("labels %q %q", BulkLabel(false), BulkLabel(true))
	}
}
