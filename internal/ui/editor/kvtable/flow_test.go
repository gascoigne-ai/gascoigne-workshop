package kvtable

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

func TestClickingACellEditsItAndEnterWritesItBack(t *testing.T) {
	w := newWorld(t)
	w.click(`text="gioui"`)
	cell, ok := w.kv.EditCell()
	if !ok || cell.Row != 0 || cell.Col != 1 {
		t.Fatalf("edit %+v %v", cell, ok)
	}
	w.typeInto("rust")
	w.kv.Commit()
	if w.kv.Editing() || w.tab.Req.Params[0].Value != "rust" {
		t.Fatalf("editing %v value %q", w.kv.Editing(), w.tab.Req.Params[0].Value)
	}
	if !reflect.DeepEqual(w.fake.changed, []logic.Table{w.params}) {
		t.Fatalf("the host was told %v, want one change of the Params", w.fake.changed)
	}
}

func TestClickingAnotherCellCommitsAndMovesTheEdit(t *testing.T) {
	w := newWorld(t)
	w.click(`text="gioui"`)
	w.typeInto("go")
	w.click(`text="desc"`)
	cell, ok := w.kv.EditCell()
	if w.tab.Req.Params[0].Value != "go" || !ok || cell.Row != 1 {
		t.Fatalf("value %q edit %+v %v", w.tab.Req.Params[0].Value, cell, ok)
	}
}

func TestAClickElsewhereCommitsTheEditInTick(t *testing.T) {
	w := newWorld(t)
	w.click(`text="gioui"`)
	w.typeInto("go")
	changed := w.frame(core.Input{X: 630, Y: 390, Click: true, Held: true})
	w.frame(core.Input{X: 630, Y: 390})
	if w.kv.Editing() || w.tab.Req.Params[0].Value != "go" {
		t.Fatalf("editing %v value %q", w.kv.Editing(), w.tab.Req.Params[0].Value)
	}
	_ = changed
}

func TestTabAndShiftTabMoveBetweenCells(t *testing.T) {
	w := newWorld(t)
	w.click(`text="q"`)
	w.typeInto("k")
	w.frame(core.Input{Tab: true})
	cell, ok := w.kv.EditCell()
	if !ok || cell.Row != 0 || cell.Col != 1 || w.tab.Req.Params[0].Name != "k" {
		t.Fatalf("after Tab: %+v %v name %q", cell, ok, w.tab.Req.Params[0].Name)
	}
	w.frame(core.Input{Tab: true})
	w.frame(core.Input{Tab: true})
	if cell, _ := w.kv.EditCell(); cell.Row != 1 || cell.Col != 0 {
		t.Fatalf("Tab past the last column wraps to the next row, edit %+v", cell)
	}
	w.frame(core.Input{Tab: true, Shift: true})
	if cell, _ := w.kv.EditCell(); cell.Row != 0 || cell.Col != 2 {
		t.Fatalf("Shift+Tab goes back, edit %+v", cell)
	}
}

func TestTypingInThePlaceholderRowAddsARow(t *testing.T) {
	w := newWorld(t)
	w.click(`text="Name"`)
	w.typeInto("page")
	w.kv.Commit()
	if len(w.tab.Req.Params) != 4 || w.tab.Req.Params[3].Name != "page" || !w.tab.Req.Params[3].Enabled {
		t.Fatalf("params %+v", w.tab.Req.Params)
	}
}

func TestLeavingThePlaceholderRowEmptyAddsNothing(t *testing.T) {
	w := newWorld(t)
	w.click(`text="Name"`)
	w.kv.Commit()
	if len(w.tab.Req.Params) != 3 || w.kv.Editing() || len(w.fake.changed) != 0 {
		t.Fatalf("params %+v changes %v", w.tab.Req.Params, w.fake.changed)
	}
}

// A cell edit pointing at a row past the end (the table shrank while the
// cell was open) appended one row and then indexed the row it had meant,
// which was out of range.
func TestCommittingACellEditOfAVanishedRowAppendsOneRow(t *testing.T) {
	w := newWorld(t)
	before := len(w.tab.Req.Params)
	w.kv.BeginEdit(w.params, before, 0)
	w.kv.edit.Row = before + 5 // the rows above were deleted meanwhile
	w.kv.edit.field = widget.NewTextField("late")
	w.kv.Commit()
	if len(w.tab.Req.Params) != before+1 || w.tab.Req.Params[before].Name != "late" {
		t.Fatalf("params %+v, want one new row named late", w.tab.Req.Params)
	}
}

func TestDeletingRowsUnderAnOpenCellEditDropsTheEdit(t *testing.T) {
	w := newWorld(t)
	w.kv.BeginEdit(w.params, 2, 1)
	w.kv.applyRows(w.params, func(rows []request.KV) []request.KV { return request.DeleteRow(request.DeleteRow(rows, 2), 1) })
	if w.kv.Editing() {
		t.Fatal("an edit pointing at a row that is gone must be dropped")
	}
	if !reflect.DeepEqual(w.names(), []string{"q"}) || len(w.fake.changed) != 1 {
		t.Fatalf("params %v, changes %v", w.names(), w.fake.changed)
	}
}

func TestBulkEditShowsLinesAndAppliesThemBackAsOneUndoStep(t *testing.T) {
	w := newWorld(t)
	w.tab.Req.Params[1].Enabled = false
	w.click(`text="Bulk edit"`)
	area := w.kv.BulkArea()
	if area == nil || area.Text != "q: gioui\n# order: desc\nsort: stars" {
		t.Fatalf("bulk %+v", area)
	}
	area.SelectAll()
	area.Insert("q: rust\npage: 2\n# off: 1")
	w.click(`text="Table"`)
	if w.kv.BulkArea() != nil {
		t.Fatal("the link leaves bulk mode")
	}
	want := []request.KV{
		{Name: "q", Value: "rust", Description: "repo name", Enabled: true},
		{Name: "page", Value: "2", Enabled: true},
		{Name: "off", Value: "1", Enabled: false},
	}
	if !reflect.DeepEqual(w.tab.Req.Params, want) {
		t.Fatalf("params %+v", w.tab.Req.Params)
	}
	w.kv.Keys(core.Input{Undo: true}, false)
	if len(w.tab.Req.Params) != 3 || w.tab.Req.Params[0].Value != "gioui" {
		t.Fatalf("a bulk edit is one undo step: %+v", w.tab.Req.Params)
	}
}

func TestLeavingTheTableWhileBulkEditingAppliesTheText(t *testing.T) {
	w := newWorld(t)
	w.click(`text="Bulk edit"`)
	w.kv.BulkArea().SelectAll()
	w.kv.BulkArea().Insert("only: one")
	w.fake.shown = false // another pane shows, as clicking its tab does
	if !w.frame(core.Input{}) {
		t.Fatal("Tick reports that it applied the text")
	}
	if w.kv.BulkArea() != nil || !reflect.DeepEqual(w.names(), []string{"only"}) {
		t.Fatalf("bulk %v params %v: nothing typed may be lost", w.kv.BulkArea() != nil, w.names())
	}
}

func TestTheTogglesAndTheDeleteButtonChangeRowsAsUndoableSteps(t *testing.T) {
	w := newWorld(t)
	w.click(`text="NAME"`) // a header cell has no handler: a click that changes nothing
	if len(w.fake.changed) != 0 {
		t.Fatal("test setup")
	}
	x, y := w.pointer(`tip=Delete row`)
	w.frame(core.Input{X: x, Y: y, Click: true, Held: true})
	w.frame(core.Input{X: x, Y: y})
	if !reflect.DeepEqual(w.names(), []string{"order", "sort"}) {
		t.Fatalf("after Delete: %v", w.names())
	}
	w.kv.Keys(core.Input{Undo: true}, false)
	if !reflect.DeepEqual(w.names(), []string{"q", "order", "sort"}) {
		t.Fatalf("after undo: %v", w.names())
	}
	w.kv.Keys(core.Input{Redo: true}, false)
	if !reflect.DeepEqual(w.names(), []string{"order", "sort"}) {
		t.Fatalf("after redo: %v", w.names())
	}
}

func TestTheHeaderCheckboxEnablesOrDisablesEveryRow(t *testing.T) {
	w := newWorld(t)
	x, y := w.pointer(`tip=Enable or disable every row`)
	click := func() {
		w.frame(core.Input{X: x, Y: y, Click: true, Held: true})
		w.frame(core.Input{X: x, Y: y})
	}
	click()
	if request.AllEnabled(w.tab.Req.Params) || slicesAnyEnabled(w.tab.Req.Params) {
		t.Fatalf("all must be disabled: %+v", w.tab.Req.Params)
	}
	click()
	if !request.AllEnabled(w.tab.Req.Params) {
		t.Fatalf("all enabled again: %+v", w.tab.Req.Params)
	}
}

func slicesAnyEnabled(rows []request.KV) bool {
	for _, row := range rows {
		if row.Enabled {
			return true
		}
	}
	return false
}

func TestDragWithTheGripMovesTheRowAndAReleaseEndsIt(t *testing.T) {
	w := newWorld(t)
	rowY := func(row int) int { return 62 + row*28 + 14 }
	w.frame(core.Input{X: 10, Y: rowY(0), Click: true, Held: true}) // the first row's grip
	if !w.kv.Dragging() {
		t.Fatal("pressing a grip starts a drag")
	}
	w.frame(core.Input{X: 10, Y: rowY(1), Held: true})
	w.frame(core.Input{X: 10, Y: rowY(2), Held: true})
	if !reflect.DeepEqual(w.names(), []string{"order", "sort", "q"}) {
		t.Fatalf("after two rows down: %v", w.names())
	}
	w.frame(core.Input{X: 10, Y: rowY(0)}) // released elsewhere
	if w.kv.Dragging() {
		t.Fatal("a release ends the drag")
	}
	w.frame(core.Input{X: 10, Y: rowY(0), Held: true})
	if !reflect.DeepEqual(w.names(), []string{"order", "sort", "q"}) {
		t.Fatalf("a held button that did not start on a grip must not move rows: %v", w.names())
	}
}

func TestEachTableKeepsItsOwnColumnWidths(t *testing.T) {
	w := newWorld(t)
	w.kv.Widths(logic.Params).Weights = []float64{10, 20, 70}
	if w.kv.Widths(logic.Headers).Weights != nil {
		t.Fatal("the Headers table keeps its default widths")
	}
	if w.kv.Widths(logic.ResponseHeaders) == w.kv.Widths(logic.Headers) {
		t.Fatal("the response Headers table is not the request Headers table")
	}
	if w.kv.Widths(logic.Params) != w.kv.Widths(logic.Params) {
		t.Fatal("asking twice must give the same state")
	}
}

func TestViewShowsTheCaptionAndTheBulkLink(t *testing.T) {
	w := newWorld(t)
	dump := Dump(w.tree())
	for _, want := range []string{`text="QUERY"`, `text="Bulk edit"`, `text="Name"`} {
		if !strings.Contains(dump, want) {
			t.Errorf("dump lacks %s:\n%s", want, dump)
		}
	}
	w.look = (Look{})
	if strings.Contains(Dump(w.tree()), "Bulk edit") {
		t.Error("an empty caption leaves the row, and the link, out")
	}
}

func TestAReadOnlyTableHasNoCheckboxesAndNoEdit(t *testing.T) {
	w := newWorld(t)
	w.look = (Look{Caption: "Headers"})
	dump := Dump(w.tree())
	for _, bad := range []string{"checkB", "Bulk edit", "Drag to move the row", "Delete row"} {
		if strings.Contains(dump, bad) {
			t.Errorf("a read-only table shows %q:\n%s", bad, dump)
		}
	}
	w.click(`text="gioui"`)
	if w.kv.Editing() {
		t.Fatal("a read-only table cannot be edited")
	}
}

func TestDuplicateHeaderNamesAreFlagged(t *testing.T) {
	rows := []request.KV{{Name: "X-Id", Enabled: true}, {Name: "x-id", Enabled: true}, {Name: "Other", Enabled: true}}
	flagged := kvRows(rows, true, true)
	if !flagged[0].Flagged || !flagged[1].Flagged || flagged[2].Flagged {
		t.Fatalf("flags = %v %v %v", flagged[0].Flagged, flagged[1].Flagged, flagged[2].Flagged)
	}
	if kvRows(rows, true, false)[0].Flagged {
		t.Fatal("params may repeat a key: no flag there")
	}
}

func TestKVRows(t *testing.T) {
	rows := []request.KV{
		{Name: "q", Value: "x", Description: "d", Enabled: true},
		{Name: "off", Value: "y"},
	}
	two := kvRows(rows, false, false)
	three := kvRows(rows, true, false)
	if len(two[0].Cells) != 2 || len(three[0].Cells) != 3 || three[0].Cells[2] != "d" {
		t.Fatalf("cells: %v / %v", two[0].Cells, three[0].Cells)
	}
	if !two[0].Checked || two[0].Muted || two[1].Checked || !two[1].Muted {
		t.Fatalf("enabled rows are checked and bright, disabled ones muted: %+v", two)
	}
}

func TestSecretValuesAreMaskedInTheTable(t *testing.T) {
	rows := kvRows([]request.KV{{Name: "token", Value: "hunter2", Secret: true, Enabled: true}, {Name: "a", Value: "visible"}}, true, false)
	if rows[0].Cells[1] == "hunter2" || rows[0].Cells[1] == "" {
		t.Fatalf("secret shown as %q", rows[0].Cells[1])
	}
	if rows[1].Cells[1] != "visible" {
		t.Fatalf("plain value %q", rows[1].Cells[1])
	}
}

func menuLabels(items []widget.MenuItem) []string {
	var out []string
	for _, item := range items {
		out = append(out, item.Label)
	}
	return out
}

func (w *world) rightClick(part string) {
	w.t.Helper()
	x, y := w.pointer(part)
	w.frame(core.Input{X: x, Y: y, RightClick: true})
}

func TestTableRowMenu(t *testing.T) {
	w := newWorld(t)
	w.rightClick(`text="q"`)
	if len(w.fake.menus) != 1 {
		t.Fatalf("%d menus opened", len(w.fake.menus))
	}
	want := []string{"Copy value", "Copy name", "Copy as Name: Value", "", "Insert row above", "Insert row below", "Duplicate", "Move up", "Move down", "", "Enable all", "Disable all", "", "Delete"}
	items := w.fake.menus[0]
	if got := menuLabels(items); !reflect.DeepEqual(got, want) {
		t.Fatalf("row menu = %q", got)
	}
	if !items[7].Disabled || items[8].Disabled {
		t.Fatal("the first row cannot move up, but can move down")
	}
	items[4].Run() // Insert row above
	if len(w.tab.Req.Params) != 4 || w.tab.Req.Params[0].Name != "" || w.tab.Req.Params[1].Name != "q" {
		t.Fatalf("params %+v", w.tab.Req.Params)
	}
}

func TestTableRowMenuActionsUseTheSameUndoableChanges(t *testing.T) {
	w := newWorld(t)
	pick := func(label string) {
		for _, item := range w.fake.menus[len(w.fake.menus)-1] {
			if item.Label == label {
				item.Run()
				return
			}
		}
		t.Fatalf("no %q in the menu", label)
	}
	w.rightClick(`text="order"`)
	pick("Duplicate")
	w.rightClick(`text="q"`)
	pick("Disable all")
	w.rightClick(`text="q"`)
	pick("Delete")
	for range 3 {
		w.kv.Keys(core.Input{Undo: true}, false)
	}
	if got := w.names(); !reflect.DeepEqual(got, []string{"q", "order", "sort"}) {
		t.Fatalf("undoing every menu action restores the table: %v", got)
	}
}

func TestPlaceholderRowHasNoMenu(t *testing.T) {
	w := newWorld(t)
	w.rightClick(`text="Name"`)
	if len(w.fake.menus) != 0 {
		t.Fatal("the empty placeholder row has no row menu")
	}
}

func TestVariablesOfTheActiveEnvironmentOfferTheSecretMark(t *testing.T) {
	w := newWorld(t)
	vars := logic.Table{Tab: w.tab, Kind: logic.Vars}
	w.fake.secret[0] = false
	items := w.kv.rowMenuItems(w.ctx, vars, 0)
	last := items[len(items)-1]
	if last.Label != "Mark as secret" {
		t.Fatalf("last entry %q", last.Label)
	}
	last.Run()
	if !reflect.DeepEqual(w.fake.toggled, []int{0}) {
		t.Fatalf("toggled %v", w.fake.toggled)
	}
	if labels := menuLabels(w.kv.rowMenuItems(w.ctx, w.params, 0)); slicesContains(labels, "Mark as secret") {
		t.Fatal("only the Vars table has the secret mark")
	}
}

func slicesContains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
