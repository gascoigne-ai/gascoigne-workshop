package app

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	kvlogic "github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// The tree reads Enter and the arrows while it has the keyboard, and the
// shortcut table reads the same keys with a modifier. Neither claims the key
// from the other: both run on one press.
func TestKeysTheTreeTakesLeaveTheShortcutsRunning(t *testing.T) {
	tests := []struct {
		name  string
		input core.Input
		check func(t *testing.T, a *App, selectedBefore request.Node)
	}{
		{"Ctrl+Enter opens the row and sends", combo(ebiten.KeyEnter, true, false, false), func(t *testing.T, a *App, selectedBefore request.Node) {
			if a.cur().Req.Name != "Repository Tags" {
				t.Fatalf("the tree did not open the selected row, current is %q", a.cur().Req.Name)
			}
			waitForResponse(t, a, a.cur())
			if a.cur().Resp.Status != 200 {
				t.Fatalf("the send shortcut did not run: %+v", a.cur().Resp)
			}
		}},
		{"Alt+Down steps the tree and moves the selected row", combo(ebiten.KeyArrowDown, false, false, true), func(t *testing.T, a *App, selectedBefore request.Node) {
			if a.side.Tree().Selected() == selectedBefore {
				t.Fatalf("the tree did not step down from %v", selectedBefore)
			}
			if got := paramNames(a); len(got) < 2 || got[0] != "order" || got[1] != "q" {
				t.Fatalf("params %v: the selected row did not move down", got)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := testApp(t)
			findReq(a, "Repository Tags").URL = jsonServer(t).URL
			click(a, 100, treeRowY(rowInfo)) // the tree has the keyboard, Repository Info is open
			a.cur().Req.Params = []request.KV{{Name: "q", Enabled: true}, {Name: "order", Enabled: true}}
			a.editor.Tables.Select(kvlogic.Table{Tab: a.cur().Tab, Kind: kvlogic.Params}, 0)
			a.Step(combo(ebiten.KeyArrowDown, false, false, false)) // selects the next row, opens nothing
			if a.cur().Req.Name != "Repository Info" {
				t.Fatal("test setup: an arrow only selects")
			}
			selectedBefore := a.side.Tree().Selected()
			a.Step(tt.input)
			tt.check(t, a, selectedBefore)
		})
	}
}

// A user binding on Ctrl+Z fires whoever else wants the key: a text field
// keeps its own undo, and a showing table undoes as well.
func TestAUserBindingOnUndoStillFires(t *testing.T) {
	tests := []struct {
		name  string
		setup func(a *App)
		undo  bool // whether the table undoes too
	}{
		{"a URL field has the focus", func(a *App) { a.editor.URLBar.Field(a.cur().Tab).Focused = true }, false},
		{"a table shows and no text has the focus", func(a *App) { click(a, 739, paramsRow0Y) }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := testApp(t)
			a.settings.Shortcuts = map[string]string{"settings": "Ctrl+Z"}
			req := a.cur().Req
			tt.setup(a)
			params := len(req.Params)
			a.Step(core.Input{Keys: []ebiten.Key{ebiten.KeyZ}, Ctrl: true, Undo: true})
			if !a.over.Settings.Open() {
				t.Fatal("the binding on Ctrl+Z did not fire")
			}
			if undone := len(req.Params) != params; undone != tt.undo {
				t.Fatalf("table undid: %v, want %v", undone, tt.undo)
			}
		})
	}
}
