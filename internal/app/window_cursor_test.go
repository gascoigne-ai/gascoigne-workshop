package app

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"

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
		{"a table row being dragged", func(app *App) { app.dragRow = &rowSelection{} }, true, core.CursorMove},
		{"a tab being dragged", func(app *App) { app.list.Drag = 0 }, true, core.CursorMove},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := testApp(t)
			a.step(core.Input{X: 5000, Y: 5000, Held: test.held}) // settle the first frame
			test.setup(a)
			a.step(core.Input{X: 5000, Y: 5000, Held: test.held})
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
			a.dragNode, a.dropTarget = request.Node{}, request.Node{}
			if test.drag != nil {
				a.dragNode = request.Of(test.drag)
			}
			if test.target != nil {
				a.dropTarget = request.Of(test.target)
			}
			if got := a.treeDragging(); got != test.want {
				t.Fatalf("treeDragging = %v, want %v", got, test.want)
			}
		})
	}
}
