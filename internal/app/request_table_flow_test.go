package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"reflect"
	"strings"
	"testing"
)

func TestTypingInTheURLFieldUpdatesParams(t *testing.T) {
	a := testApp(t)
	req := a.cur().Req
	click(a, 1100, urlBarY) // inside the field, past the end of the text: cursor at the end
	a.step(core.Input{Typed: []rune("&page=2")})

	if !strings.HasSuffix(req.URL, "&page=2") {
		t.Fatalf("url = %q", req.URL)
	}
	last := req.Params[len(req.Params)-1]
	if last.Name != "page" || last.Value != "2" || !last.Enabled {
		t.Fatalf("params = %+v", req.Params)
	}
	if got := a.cur().url.Text; got != req.URL {
		t.Fatalf("field %q and request %q disagree", got, req.URL)
	}
}

func TestTogglingAParamRewritesTheURL(t *testing.T) {
	a := testApp(t)
	req := a.cur().Req
	click(a, 310, paramsRow0Y) // checkbox of the first Params row (q)
	if req.Params[0].Enabled || strings.Contains(req.URL, "q=") {
		t.Fatalf("q must be disabled and gone from %q", req.URL)
	}
	if !strings.Contains(req.URL, "order=desc") {
		t.Fatalf("other params must stay in %q", req.URL)
	}
	click(a, 310, paramsRow0Y)
	if !req.Params[0].Enabled || !strings.Contains(req.URL, "q=gioui") {
		t.Fatalf("toggling back must restore q in %q", req.URL)
	}
}

func TestDeletingAParamRewritesTheURL(t *testing.T) {
	a := testApp(t)
	req := a.cur().Req
	click(a, 739, paramsRow0Y) // delete button of the first Params row
	if len(req.Params) != 2 || req.Params[0].Name != "order" || strings.Contains(req.URL, "q=") {
		t.Fatalf("params %+v url %q", req.Params, req.URL)
	}
}

func TestTabItemsCountEnabledRowsAndHeaders(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	items := a.tabItems(requestPanes, tab)
	want := map[string]int{"Params": 3, "Body": 0, "Headers": 0, "Auth": 0, "Vars": 1}
	for _, it := range items {
		if it.Badge != want[it.Label] {
			t.Errorf("%s badge = %d, want %d", it.Label, it.Badge, want[it.Label])
		}
	}
	tab.Req.Params[0].Enabled = false
	if got := a.tabItems(requestPanes, tab)[0].Badge; got != 2 {
		t.Errorf("disabled params must not count, badge = %d", got)
	}
}

func TestEditingAParamValueUpdatesTheURL(t *testing.T) {
	a := testApp(t)
	click(a, valueCellX, paramsRow0Y)
	if a.edit == nil || a.edit.row != 0 || a.edit.col != 1 {
		t.Fatalf("edit = %+v", a.edit)
	}
	a.typeText("rust") // the old text is selected, so typing replaces it
	a.step(core.Input{Enter: true})

	req := a.cur().Req
	if a.edit != nil || req.Params[0].Value != "rust" || !strings.Contains(req.URL, "q=rust") {
		t.Fatalf("edit %+v params %+v url %q", a.edit, req.Params, req.URL)
	}
	if a.cur().url.Text != req.URL {
		t.Fatal("the URL field must show the rewritten URL")
	}
}

func TestEditingAParamNameAndDescription(t *testing.T) {
	a := testApp(t)
	click(a, nameCellX, paramsRow0Y)
	a.typeText("search")
	a.step(core.Input{Enter: true})
	click(a, descCellX, paramsRow0Y)
	a.typeText("what to find")
	a.step(core.Input{Enter: true})

	p := a.cur().Req.Params[0]
	if p.Name != "search" || p.Description != "what to find" || !strings.Contains(a.cur().Req.URL, "search=gioui") {
		t.Fatalf("param %+v url %q", p, a.cur().Req.URL)
	}
}

func TestEscapeCancelsACellEdit(t *testing.T) {
	a := testApp(t)
	click(a, valueCellX, paramsRow0Y)
	a.typeText("zzz")
	a.step(core.Input{Escape: true})
	if a.edit != nil || a.cur().Req.Params[0].Value != "gioui" {
		t.Fatalf("edit %+v value %q", a.edit, a.cur().Req.Params[0].Value)
	}
}

func TestClickingElsewhereCommitsACellEdit(t *testing.T) {
	a := testApp(t)
	click(a, valueCellX, paramsRow0Y)
	a.typeText("go")
	click(a, farAwayX, farAwayY)
	a.step(core.Input{})
	if a.edit != nil || a.cur().Req.Params[0].Value != "go" {
		t.Fatalf("edit %+v value %q", a.edit, a.cur().Req.Params[0].Value)
	}
}

func TestClickingAnotherCellCommitsAndMovesTheEdit(t *testing.T) {
	a := testApp(t)
	click(a, valueCellX, paramsRow0Y)
	a.typeText("go")
	click(a, valueCellX, paramsRow0Y+core.RowH) // the order row
	if a.cur().Req.Params[0].Value != "go" || a.edit == nil || a.edit.row != 1 {
		t.Fatalf("value %q edit %+v", a.cur().Req.Params[0].Value, a.edit)
	}
}

func TestTabMovesToTheNextCellAndShiftTabBack(t *testing.T) {
	a := testApp(t)
	click(a, nameCellX, paramsRow0Y)
	a.typeText("k")
	a.step(core.Input{Tab: true})
	if a.edit == nil || a.edit.row != 0 || a.edit.col != 1 || a.cur().Req.Params[0].Name != "k" {
		t.Fatalf("after Tab: edit %+v name %q", a.edit, a.cur().Req.Params[0].Name)
	}
	a.step(core.Input{Tab: true})
	a.step(core.Input{Tab: true}) // description, then the next row's name
	if a.edit == nil || a.edit.row != 1 || a.edit.col != 0 {
		t.Fatalf("Tab past the last column must wrap to the next row, edit %+v", a.edit)
	}
	a.step(core.Input{Tab: true, Shift: true})
	if a.edit == nil || a.edit.row != 0 || a.edit.col != 2 {
		t.Fatalf("Shift+Tab goes back, edit %+v", a.edit)
	}
}

func TestTypingInThePlaceholderRowAddsARow(t *testing.T) {
	a := testApp(t)
	click(a, nameCellX, paramsRow3Y)
	a.typeText("page")
	a.step(core.Input{Tab: true})
	a.typeText("2")
	a.step(core.Input{Enter: true})

	req := a.cur().Req
	if len(req.Params) != 4 {
		t.Fatalf("params = %+v", req.Params)
	}
	if got := req.Params[3]; got.Name != "page" || got.Value != "2" || !got.Enabled {
		t.Fatalf("new row = %+v", got)
	}
	if !strings.HasSuffix(req.URL, "&page=2") {
		t.Fatalf("url = %q", req.URL)
	}
}

func TestLeavingThePlaceholderRowEmptyAddsNothing(t *testing.T) {
	a := testApp(t)
	click(a, nameCellX, paramsRow3Y)
	a.step(core.Input{Enter: true})
	if len(a.cur().Req.Params) != 3 || a.edit != nil {
		t.Fatalf("params %+v edit %+v", a.cur().Req.Params, a.edit)
	}
}

func TestHeadersTableIsEditableToo(t *testing.T) {
	a := testApp(t)
	a.cur().paneTab = headersTabIdx
	a.step(core.Input{})
	click(a, nameCellX, paramsRow0Y) // headers have no rows: the placeholder is the first row
	a.typeText("X-Test")
	a.step(core.Input{Tab: true})
	a.typeText("1")
	a.step(core.Input{Enter: true})
	h := a.cur().Req.Headers
	if len(h) != 1 || h[0].Name != "X-Test" || h[0].Value != "1" || !h[0].Enabled {
		t.Fatalf("headers = %+v", h)
	}
}

func TestResponseHeadersAreNotEditable(t *testing.T) {
	srv := jsonServer(t)
	a := testApp(t)
	tab := a.cur()
	pointTabAt(tab, srv.URL)
	a.send(tab)
	waitForResponse(t, a, tab)
	tab.respTab = 1 // Headers
	a.step(core.Input{})
	click(a, 1000, paramsRow0Y)
	if a.edit != nil {
		t.Fatalf("response headers must stay read-only, edit %+v", a.edit)
	}
}

func paramNames(app *App) []string {
	var out []string
	for _, p := range app.cur().Req.Params {
		out = append(out, p.Name)
	}
	return out
}

func TestCtrlEnterWhileEditingInsertsARowBelowAndEditsIt(t *testing.T) {
	a := testApp(t)
	click(a, valueCellX, paramsRow0Y)
	a.step(combo(ebiten.KeyEnter, true, false, false))
	req := a.cur().Req
	if len(req.Params) != 4 || req.Params[1].Name != "" || !req.Params[1].Enabled || req.Params[0].Name != "q" {
		t.Fatalf("params = %+v", req.Params)
	}
	if a.edit == nil || a.edit.row != 1 || a.edit.col != 1 {
		t.Fatalf("edit = %+v: the new row is edited in the same column", a.edit)
	}
	if strings.Contains(req.URL, "&=") || strings.Contains(req.URL, "?=") {
		t.Fatalf("an empty name must stay out of the URL: %q", req.URL)
	}
}

func TestCtrlDDuplicatesTheEditedRow(t *testing.T) {
	a := testApp(t)
	click(a, valueCellX, paramsRow0Y)
	a.step(combo(ebiten.KeyD, true, false, false))
	p := a.cur().Req.Params
	if len(p) != 4 || p[1] != p[0] || p[1].Name != "q" {
		t.Fatalf("params = %+v", p)
	}
	if a.edit == nil || a.edit.row != 1 {
		t.Fatalf("the copy is the edited row, edit = %+v", a.edit)
	}
}

func TestAltArrowsMoveTheRowAndTheEditFollows(t *testing.T) {
	a := testApp(t)
	click(a, valueCellX, paramsRow0Y)
	a.typeText("rust") // an uncommitted edit travels with the row
	a.step(combo(ebiten.KeyArrowDown, false, false, true))
	if got := paramNames(a); !reflect.DeepEqual(got, []string{"order", "q", "sort"}) {
		t.Fatalf("params = %v", got)
	}
	if a.edit == nil || a.edit.row != 1 || a.cur().Req.Params[1].Value != "rust" {
		t.Fatalf("edit %+v value %q", a.edit, a.cur().Req.Params[1].Value)
	}
	if !strings.Contains(a.cur().Req.URL, "order=desc&q=rust&sort=stars") {
		t.Fatalf("the URL follows the order: %q", a.cur().Req.URL)
	}
	a.step(combo(ebiten.KeyArrowUp, false, false, true))
	a.step(combo(ebiten.KeyArrowUp, false, false, true)) // already at the top: stays
	if got := paramNames(a); !reflect.DeepEqual(got, []string{"q", "order", "sort"}) {
		t.Fatalf("params = %v", got)
	}
}

// The drag handle sits in the first 20 px of the table, which starts at x=272.
const gripCellX = 282

func TestDraggingARowByItsGripMovesItAndAFurtherDragKeepsMovingIt(t *testing.T) {
	a := testApp(t)
	rowY := func(row int) int { return paramsRow0Y + row*core.RowH }
	a.step(core.Input{X: gripCellX, Y: rowY(0), Click: true, Held: true}) // press the first row's grip
	a.step(core.Input{X: gripCellX, Y: rowY(1), Held: true})
	if got := paramNames(a); !reflect.DeepEqual(got, []string{"order", "q", "sort"}) {
		t.Fatalf("after one row down: %v", got)
	}
	a.step(core.Input{X: gripCellX, Y: rowY(2), Held: true})
	if got := paramNames(a); !reflect.DeepEqual(got, []string{"order", "sort", "q"}) {
		t.Fatalf("after two rows down: %v", got)
	}
	if !strings.Contains(a.cur().Req.URL, "order=desc&sort=stars&q=gioui") {
		t.Fatalf("the URL follows the order: %q", a.cur().Req.URL)
	}
	a.step(core.Input{X: gripCellX, Y: rowY(0)}) // release elsewhere
	a.step(core.Input{X: gripCellX, Y: rowY(0), Held: true})
	if got := paramNames(a); !reflect.DeepEqual(got, []string{"order", "sort", "q"}) {
		t.Fatalf("a held button that did not start on a grip must not move rows: %v", got)
	}
}

func TestDeleteRemovesTheSelectedRowWhenNothingIsBeingTyped(t *testing.T) {
	a := testApp(t)
	click(a, valueCellX, paramsRow0Y)
	a.step(core.Input{Enter: true}) // commit; the row stays selected
	a.step(combo(ebiten.KeyDelete, false, false, false))
	if got := paramNames(a); !reflect.DeepEqual(got, []string{"order", "sort"}) || strings.Contains(a.cur().Req.URL, "q=") {
		t.Fatalf("params %v url %q", got, a.cur().Req.URL)
	}
	a.step(combo(ebiten.KeyDelete, false, false, false)) // the selection moved on: a second Delete removes the next row
	if len(a.cur().Req.Params) > 2 {
		t.Fatal("unreachable")
	}
}

func TestDeleteDoesNothingWithoutASelectedRowOrWhileTyping(t *testing.T) {
	a := testApp(t)
	a.step(combo(ebiten.KeyDelete, false, false, false))
	if len(a.cur().Req.Params) != 3 {
		t.Fatal("no row is selected")
	}
	click(a, valueCellX, paramsRow0Y) // editing: Delete is a text key
	a.step(combo(ebiten.KeyDelete, false, false, false))
	if len(a.cur().Req.Params) != 3 {
		t.Fatal("while a cell is edited Delete belongs to the text")
	}
}

func TestHeaderCheckboxEnablesOrDisablesEveryRow(t *testing.T) {
	a := testApp(t)
	req := a.cur().Req
	click(a, checkAllX, paramsHeaderY)
	for _, p := range req.Params {
		if p.Enabled {
			t.Fatalf("all must be disabled: %+v", req.Params)
		}
	}
	if strings.Contains(req.URL, "?") {
		t.Fatalf("no enabled params, no query: %q", req.URL)
	}
	click(a, checkAllX, paramsHeaderY)
	if !request.AllEnabled(req.Params) || !strings.Contains(req.URL, "q=gioui") {
		t.Fatalf("all enabled again: %+v url %q", req.Params, req.URL)
	}
}

func TestUndoAndRedoOfTableChanges(t *testing.T) {
	a := testApp(t)
	req := a.cur().Req
	click(a, 739, paramsRow0Y) // delete the first row
	if len(req.Params) != 2 {
		t.Fatal("test setup")
	}
	a.step(core.Input{Ctrl: true, Undo: true})
	if len(req.Params) != 3 || req.Params[0].Name != "q" || !strings.Contains(req.URL, "q=gioui") {
		t.Fatalf("undo: %+v url %q", req.Params, req.URL)
	}
	a.step(core.Input{Ctrl: true, Redo: true})
	if len(req.Params) != 2 || strings.Contains(req.URL, "q=") {
		t.Fatalf("redo: %+v url %q", req.Params, req.URL)
	}
	a.step(core.Input{Ctrl: true, Undo: true})
	click(a, 310, row1Y) // a new change after an undo clears redo
	a.step(core.Input{Ctrl: true, Redo: true})
	if req.Params[1].Enabled {
		t.Fatalf("redo must be gone after a new change: %+v", req.Params)
	}
	a.step(core.Input{Ctrl: true, Undo: true})
	a.step(core.Input{Ctrl: true, Undo: true})
	a.step(core.Input{Ctrl: true, Undo: true}) // more undos than changes: stops at the start
	if len(req.Params) != 3 {
		t.Fatalf("params = %+v", req.Params)
	}
}

func TestUndoGoesToTheCellTextWhileACellIsEdited(t *testing.T) {
	a := testApp(t)
	click(a, 739, paramsRow0Y) // a table change to undo
	click(a, valueCellX, paramsRow0Y)
	a.typeText("zz")
	a.step(core.Input{Ctrl: true, Undo: true})
	if len(a.cur().Req.Params) != 2 {
		t.Fatal("Ctrl+Z inside a cell undoes the typing, not the table")
	}
}

func TestDeletingRowsUnderAnOpenCellEditDropsTheEditAndRewritesTheURL(t *testing.T) {
	a := testApp(t)
	click(a, valueCellX, paramsRow0Y+2*core.RowH) // edit row 2
	if a.edit == nil || a.edit.row != 2 {
		t.Fatal("test setup")
	}
	a.applyRows(requestPanes[0], a.cur(), func(rows []request.KV) []request.KV { return request.DeleteRow(request.DeleteRow(rows, 2), 1) })
	if a.edit != nil {
		t.Fatalf("an edit pointing at a row that is gone must be dropped: %+v", a.edit)
	}
	if got := paramNames(a); !reflect.DeepEqual(got, []string{"q"}) || strings.Contains(a.cur().Req.URL, "order") {
		t.Fatalf("params %v url %q", got, a.cur().Req.URL)
	}
}

func TestBulkEditShowsLinesAndAppliesThemBack(t *testing.T) {
	a := testApp(t)
	req := a.cur().Req
	req.Params[1].Enabled = false
	click(a, bulkBtnX, chipsY)
	if a.bulk == nil || a.bulk.area.Text != "q: gioui\n# order: desc\nsort: stars" {
		t.Fatalf("bulk = %+v", a.bulk)
	}
	a.bulk.area.SelectAll()
	a.bulk.area.Insert("q: rust\npage: 2\n# off: 1")
	click(a, bulkBtnX, chipsY) // back to the table
	if a.bulk != nil {
		t.Fatal("the button leaves bulk mode")
	}
	want := []request.KV{
		{Name: "q", Value: "rust", Description: "repo name", Enabled: true},
		{Name: "page", Value: "2", Enabled: true},
		{Name: "off", Value: "1", Enabled: false},
	}
	if !reflect.DeepEqual(req.Params, want) || !strings.HasSuffix(req.URL, "?q=rust&page=2") {
		t.Fatalf("params %+v url %q", req.Params, req.URL)
	}
	a.step(core.Input{Ctrl: true, Undo: true})
	if len(req.Params) != 3 || req.Params[0].Value != "gioui" {
		t.Fatalf("a bulk edit is one undo step: %+v", req.Params)
	}
}

func TestLeavingTheTabWhileBulkEditingAppliesTheText(t *testing.T) {
	a := testApp(t)
	click(a, bulkBtnX, chipsY)
	a.bulk.area.SelectAll()
	a.bulk.area.Insert("only: one")
	a.cur().paneTab = headersTabIdx // as clicking the Headers tab does
	a.step(core.Input{})
	if a.bulk != nil || !reflect.DeepEqual(paramNames(a), []string{"only"}) {
		t.Fatalf("bulk %v params %v: nothing typed may be lost", a.bulk != nil, paramNames(a))
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

func TestTableRowMenu(t *testing.T) {
	a := testApp(t)
	rightClick(a, 400, paramsRow0Y)
	want := []string{"Copy value", "Copy name", "Copy as Name: Value", "", "Insert row above", "Insert row below", "Duplicate", "Move up", "Move down", "", "Enable all", "Disable all", "", "Delete"}
	if got := menuLabels(a); !reflect.DeepEqual(got, want) {
		t.Fatalf("row menu = %q", got)
	}
	if !itemByLabel(t, a, "Move up").Disabled || itemByLabel(t, a, "Move down").Disabled {
		t.Fatal("the first row cannot move up, but can move down")
	}
	pickMenu(t, a, "Insert row above")
	req := a.cur().Req
	if len(req.Params) != 4 || req.Params[0].Name != "" || req.Params[1].Name != "q" {
		t.Fatalf("params = %+v", req.Params)
	}
}

func TestTableRowMenuActionsUseTheSameUndoableChanges(t *testing.T) {
	a := testApp(t)
	req := a.cur().Req
	rightClick(a, 400, row1Y) // the "order" row
	pickMenu(t, a, "Duplicate")
	if got := paramNames(a); !reflect.DeepEqual(got, []string{"q", "order", "order", "sort"}) {
		t.Fatalf("duplicate: %v", got)
	}
	rightClick(a, 400, row1Y)
	pickMenu(t, a, "Move down")
	rightClick(a, 400, paramsRow0Y)
	pickMenu(t, a, "Disable all")
	if request.AllEnabled(req.Params) || strings.Contains(req.URL, "?") {
		t.Fatalf("disable all: %+v url %q", req.Params, req.URL)
	}
	rightClick(a, 400, paramsRow0Y)
	pickMenu(t, a, "Delete")
	if len(req.Params) != 3 {
		t.Fatalf("delete: %v", paramNames(a))
	}
	for range 4 {
		a.step(core.Input{Ctrl: true, Undo: true})
	}
	if got := paramNames(a); !reflect.DeepEqual(got, []string{"q", "order", "sort"}) || !strings.Contains(req.URL, "q=gioui") {
		t.Fatalf("undoing every menu action restores the table: %v url %q", got, req.URL)
	}
}

func TestPlaceholderRowHasNoMenu(t *testing.T) {
	a := testApp(t)
	rightClick(a, 400, paramsRow3Y)
	if widget.PopupMenu(a.ctx) != nil {
		t.Fatal("the empty placeholder row has no row menu")
	}
}

// The Params table: columns name|value|description at weights 28/40/32;
// the name|value divider is at x=438, the header row at y=234.
const (
	nameValueDividerX = 438
	headersPaneTabIdx = 2
)

func TestDraggingAParamsDividerResizesItsColumns(t *testing.T) {
	a := testApp(t)
	drag(a, nameValueDividerX, 480, paramsHeaderY)
	w := a.widthsFor(requestPanes[0])
	if w == nil || len(w.Weights) != 3 || w.Weights[0] <= 28 || w.Weights[1] >= 40 {
		t.Fatalf("weights = %+v", w)
	}
	if sum := w.Weights[0] + w.Weights[1] + w.Weights[2]; sum < 99.9 || sum > 100.1 {
		t.Fatalf("the total must stay 100, got %v", sum)
	}
	click(a, 450, paramsRow0Y) // was in the value column; now it is the wider name column
	if a.edit == nil || a.edit.col != 0 {
		t.Fatalf("a click at x=450 edits %+v: the name column must have grown over it", a.edit)
	}
}

func TestEachTableKeepsItsOwnColumnWidths(t *testing.T) {
	a := testApp(t)
	drag(a, nameValueDividerX, 480, paramsHeaderY)
	if a.widthsFor(requestPanes[0]).Weights == nil {
		t.Fatal("test setup")
	}
	if a.widthsFor(requestPanes[headersPaneTabIdx]).Weights != nil {
		t.Fatal("the Headers table must keep its default widths")
	}
	if a.widthsFor(responsePanes[1]) == a.widthsFor(requestPanes[headersPaneTabIdx]) {
		t.Fatal("the response Headers table is not the request Headers table")
	}
	if a.widthsFor(requestPanes[0]) != a.widthsFor(requestPanes[0]) {
		t.Fatal("asking twice must give the same state")
	}
}

func TestDoubleClickOnAParamsDividerResetsIt(t *testing.T) {
	a := testApp(t)
	drag(a, nameValueDividerX, 480, paramsHeaderY)
	a.step(core.Input{X: 480, Y: paramsHeaderY, Click: true, Held: true, Clicks: 2})
	a.step(core.Input{})
	if a.widthsFor(requestPanes[0]).Weights != nil {
		t.Fatal("double click resets the widths")
	}
}

func TestNarrowColumnsKeepAMinimum(t *testing.T) {
	a := testApp(t)
	drag(a, nameValueDividerX, 300, paramsHeaderY) // far left
	w := a.widthsFor(requestPanes[0])
	// the name column is at least MinColumnPx of the 393px of columns
	if min := float64(widget.MinColumnPx) / 393 * 100; w.Weights[0] < min-0.5 {
		t.Fatalf("name weight %v is below the minimum share %v", w.Weights[0], min)
	}
}

// Both panes start their first row at the same height, so the two halves
// of the window line up across the divider.
func TestRequestAndResponsePanesShareTheirFirstRowBand(t *testing.T) {
	a := jsonResponse(t, `{"ok":true}`)
	clip := &fakeClip{}
	a.ctx.Clipboard = clip
	click(a, bulkBtnX, chipsY) // the request pane's first row
	if a.bulk == nil {
		t.Fatal("the Bulk edit link is in the first row of the request pane")
	}
	click(a, copyBtnX, chipsY+10) // the lower edge of the shared band, below the old toolbar
	if clip.text != `{"ok":true}` {
		t.Fatalf("the Copy button is not in the same band as Bulk edit, clipboard %q", clip.text)
	}
}
