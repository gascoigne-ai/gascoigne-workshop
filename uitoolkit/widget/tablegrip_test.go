package widget

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// gripRig builds a three-row table with a drag handle and records what
// the handle reports.
type gripRig struct {
	c       *core.Ctx
	root    *El
	grabbed []int
	over    []int
}

func newGripRig(t *testing.T) *gripRig {
	rig := &gripRig{c: testCtx(t)}
	spec := editableSpec()
	spec.Rows = append(spec.Rows, TableRow{Cells: []string{"z", "w"}})
	spec.OnGrabRow = func(row int) { rig.grabbed = append(rig.grabbed, row) }
	spec.OnDragOverRow = func(row int) { rig.over = append(rig.over, row) }
	rig.root = Col(Table(spec))
	Layout(rig.c, rig.root, 400, 400)
	return rig
}

func (rig *gripRig) input(in core.Input) {
	rig.c.In = in
	Update(rig.c, rig.root)
}

const gripX = tableGripW / 2

func TestPressingARowGripStartsADragOfThatRow(t *testing.T) {
	rig := newGripRig(t)
	rig.input(core.Input{X: gripX, Y: cellY(1), Click: true, Held: true})
	if len(rig.grabbed) != 1 || rig.grabbed[0] != 1 {
		t.Fatalf("grabbed = %v, want [1]", rig.grabbed)
	}
}

func TestDraggingOverARowReportsThatRow(t *testing.T) {
	rig := newGripRig(t)
	rig.input(core.Input{X: gripX, Y: cellY(0), Click: true, Held: true})
	rig.input(core.Input{X: gripX, Y: cellY(2), Held: true})
	if len(rig.over) == 0 || rig.over[len(rig.over)-1] != 2 {
		t.Fatalf("over = %v, want the last to be 2", rig.over)
	}
}
