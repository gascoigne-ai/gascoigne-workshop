package app

import (
	kvlogic "github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable/logic"
	"reflect"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// tableEditing, tableBulk and typeBulk read the table state the way the
// window sees it, so the pins below do not name where it lives.
func tableEditing(a *App) bool { return a.kv.Editing() }

// editCell is the cell being edited, the zero cell when none.
func editCell(a *App) kvlogic.Cell {
	cell, _ := a.kv.EditCell()
	return cell
}
func tableBulk(a *App) bool { return a.kv.BulkArea() != nil }
func typeBulk(a *App, text string) {
	a.kv.BulkArea().SelectAll()
	a.kv.BulkArea().Insert(text)
}

// A second tab, and the first one active again, so closing the first one
// leaves another tab to show.
func withTwoParamsTabs(t *testing.T) *App {
	t.Helper()
	a := testApp(t)
	a.newRequest()
	a.updateTabs(tabs.Activate{Index: 0})
	a.Step(core.Input{})
	return a
}

// These pin what happens to a table edit that is open when its tab closes,
// before the table state moved into ui/editor/kvtable: they must pass on
// both sides of the move.
func TestClosingATabDropsItsCellEditWithoutWriting(t *testing.T) {
	a := withTwoParamsTabs(t)
	req := a.cur().Req
	want := req.Params[0].Value
	click(a, valueCellX, paramsRow0Y)
	a.typeText("zzz")
	if !tableEditing(a) {
		t.Fatal("test setup: a cell is being edited")
	}

	a.closeTab(0)
	a.Step(core.Input{})

	if tableEditing(a) || req.Params[0].Value != want {
		t.Fatalf("editing %v, value %q: the text of a closed tab's edit is dropped", tableEditing(a), req.Params[0].Value)
	}
}

func TestClosingATabWhileBulkEditingAppliesTheText(t *testing.T) {
	a := withTwoParamsTabs(t)
	req := a.cur().Req
	click(a, bulkBtnX, chipsY)
	typeBulk(a, "only: one")

	a.closeTab(0)
	a.Step(core.Input{})

	if got := []string{req.Params[0].Name}; len(req.Params) != 1 || !reflect.DeepEqual(got, []string{"only"}) {
		t.Fatalf("params %+v: nothing typed may be lost when the tab closes", req.Params)
	}
	if tableBulk(a) {
		t.Fatal("bulk mode ends with the tab")
	}
}

func TestUndoGoesToTheURLWhileTheURLFieldIsFocused(t *testing.T) {
	a := testApp(t)
	req := a.cur().Req
	click(a, 739, paramsRow0Y) // a table change that could be undone
	if len(req.Params) != 2 {
		t.Fatal("test setup")
	}
	click(a, 1100, urlBarY)
	a.Step(core.Input{Ctrl: true, Undo: true})
	if len(req.Params) != 2 {
		t.Fatalf("params %+v: with the URL field focused Ctrl+Z belongs to its text", req.Params)
	}
}

// The history of a table is keyed by the label of its pane, and the pane
// names its table by kind: the two must agree, or the undo history of one
// table would be shared with another's.
func TestEachTablePaneLabelMatchesItsKind(t *testing.T) {
	for kind := kvlogic.Params; kind < kvlogic.KindCount; kind++ {
		shown := tablePanes[kind]
		if shown.kind != paneTable || shown.table != kind || shown.label != kind.Label() {
			t.Errorf("kind %d: pane %q (kind %d, table %d), want label %q", kind, shown.label, shown.kind, shown.table, kind.Label())
		}
	}
	for _, list := range [][]pane{requestPanes, connectionRequestPanes, grpcRequestPanes, responsePanes} {
		for _, shown := range list {
			if (shown.kind == paneTable) != (shown.table != kvlogic.NoTable) {
				t.Errorf("pane %q: kind %d with table %d", shown.label, shown.kind, shown.table)
			}
		}
	}
}
