package app

import (
	kvlogic "github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable/logic"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestTheCursorIsAGrabWhileSomethingIsBeingDragged(t *testing.T) {
	tests := []struct {
		name  string
		setup func(app *App)
		held  bool
		want  core.Cursor
	}{
		{"nothing grabbed", func(*App) {}, true, core.CursorDefault},
		{"a table row being dragged", func(app *App) { app.kv.GrabRow(kvlogic.Table{Tab: app.cur().Tab, Kind: kvlogic.Params}, 0) }, true, core.CursorMove},
		{"a tab being dragged", func(app *App) { app.tabs.Apply(tabs.DragBegin{Index: 0}) }, true, core.CursorMove},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := testApp(t)
			a.Step(core.Input{X: 5000, Y: 5000, Held: test.held}) // settle the first frame
			test.setup(a)
			a.Step(core.Input{X: 5000, Y: 5000, Held: test.held})
			if a.ctx.Cursor != test.want {
				t.Fatalf("cursor = %v, want %v", a.ctx.Cursor, test.want)
			}
		})
	}
}

// A tree row is grabbed once the pointer is over another row; the drop
// target is set during the frame by that row, so it is checked directly.
func TestATreeRowIsGrabbedOnlyOnceItIsOverAnotherRow(t *testing.T) {
	a := testApp(t)
	first, second := a.col().Folders[0], a.col().Folders[1]
	tests := []struct {
		name         string
		drag, target any
		want         bool
	}{
		{"only pressed", first, nil, false},
		{"over itself", first, first, false},
		{"over another row", first, second, true},
		{"nothing pressed", nil, second, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var drag, target request.Node
			if test.drag != nil {
				drag = request.Of(test.drag)
			}
			if test.target != nil {
				target = request.Of(test.target)
			}
			holdDrag(a, drag, target)
			if got := a.treeDragging(); got != test.want {
				t.Fatalf("treeDragging = %v, want %v", got, test.want)
			}
		})
	}
}
