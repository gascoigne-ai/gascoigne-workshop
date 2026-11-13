package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Sizes the key/value table is pinned at: the request pane of the test
// window, a narrow one, and a height that leaves room under the rows.
const (
	tableWideW   = winW - sidebarW
	tableNarrowW = minMainW
	tableH       = 400
)

// tableViewState is a named look of a key/value table, how to give the
// open tab that look, and which pane and padding show it.
type tableViewState struct {
	name  string
	pane  pane
	pad   int
	setUp func(t *testing.T, a *App, tab *reqTab)
}

// longParamValue does not fit its cell at either width.
const longParamValue = "gpu-rendered-desktop-clients-written-in-go-with-custom-widgets-and-three-panes"

// tableViewStates are the looks of the table: the sample's params, a
// disabled row and a long value, no rows, a selected row, a cell being
// edited, headers with a repeated name, the variables, the form fields
// (no padding), a response's headers (read-only, no caption) and the
// rows as text in bulk mode.
var tableViewStates = []tableViewState{
	{"params", paramsPane, core.Gutter, func(t *testing.T, a *App, tab *reqTab) {}},
	{"params-disabled-long", paramsPane, core.Gutter, func(t *testing.T, a *App, tab *reqTab) {
		tab.Req.Params[0].Value = longParamValue
		tab.Req.Params[1].Enabled = false
	}},
	{"params-empty", paramsPane, core.Gutter, func(t *testing.T, a *App, tab *reqTab) {
		tab.Req.Params = nil
	}},
	{"params-selected", paramsPane, core.Gutter, func(t *testing.T, a *App, tab *reqTab) {
		a.rowSel = &rowSelection{pane: paramsPane, tab: tab, row: 1}
	}},
	{"params-editing", paramsPane, core.Gutter, func(t *testing.T, a *App, tab *reqTab) {
		a.beginEdit(paramsPane, tab, 0, 1)
	}},
	{"headers-duplicate", headersPane, core.Gutter, func(t *testing.T, a *App, tab *reqTab) {
		tab.Req.Headers = []model.KV{
			{Name: "X-Id", Value: "1", Enabled: true},
			{Name: "x-id", Value: "2", Enabled: true},
			{Name: "Accept", Value: "application/json", Enabled: true},
		}
	}},
	{"vars", varsPane, core.Gutter, func(t *testing.T, a *App, tab *reqTab) {}},
	{"form", formPane, 0, func(t *testing.T, a *App, tab *reqTab) {
		tab.Req.Form = []model.KV{{Name: "name", Value: "gopher", Enabled: true}}
	}},
	{"response-headers", responsePanes[1], core.Gutter, func(t *testing.T, a *App, tab *reqTab) {
		tab.Resp = &model.Response{Headers: []model.KV{
			{Name: "Content-Type", Value: "text/plain; charset=utf-8", Enabled: true},
			{Name: "Content-Length", Value: "82", Enabled: true},
		}}
	}},
	{"bulk", paramsPane, core.Gutter, func(t *testing.T, a *App, tab *reqTab) {
		a.toggleBulk(paramsPane, tab)
	}},
}

// The key/value table is pinned as text (element kinds, options,
// handlers, rectangles) at two widths in each of its looks, so moving it
// to markup cannot change it unseen.
func TestTheKeyValueTableLaysOutAsPinned(t *testing.T) {
	sizes := []struct {
		name  string
		width int
	}{
		{"wide", tableWideW},
		{"narrow", tableNarrowW},
	}
	for _, state := range tableViewStates {
		for _, size := range sizes {
			name := "tableview-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				a, _ := newTestApp(t)
				t.Cleanup(a.waitWrites)
				tab := a.cur()
				state.setUp(t, a, tab)
				root := Stack(a.tableView(state.pane, tab, &widget.ScrollState{}, state.pad))
				Layout(a.ctx, root, size.width, tableH)
				want, err := os.ReadFile(filepath.Join("testdata", name+".dump"))
				if err != nil {
					t.Fatal(err)
				}
				if got := Dump(root); got != string(want) {
					t.Fatalf("got:\n%s\nwant:\n%s", got, want)
				}
			})
		}
	}
}

// Right-clicking the text of bulk mode opens the text menu, as in every
// other text area.
func TestTheBulkTextHasTheTextMenu(t *testing.T) {
	a, _ := newTestApp(t)
	t.Cleanup(a.waitWrites)
	click(a, bulkBtnX, chipsY)
	if a.bulk == nil {
		t.Fatal("test setup: the Bulk edit link must open bulk mode")
	}
	rightClick(a, valueCellX, chipsY+3*core.RowH)
	if widget.PopupMenu(a.ctx) == nil || menuLabels(a)[0] != "Undo" {
		t.Fatalf("menu = %v", widget.PopupMenu(a.ctx))
	}
}
