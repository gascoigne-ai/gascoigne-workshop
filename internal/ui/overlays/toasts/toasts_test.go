package toasts

import (
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request/severity"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// fakeWorld is both Deps in one: a clock the test moves and a window that
// counts what the toasts asked of it.
type fakeWorld struct {
	now            time.Time
	stale, redrawn int
	width, height  int
}

func (world *fakeWorld) Now() time.Time   { return world.now }
func (world *fakeWorld) Size() (int, int) { return world.width, world.height }
func (world *fakeWorld) Stale()           { world.stale++ }
func (world *fakeWorld) Redraw()          { world.redrawn++ }

func newWorld() (*Component, *fakeWorld) {
	world := &fakeWorld{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC), width: 1280, height: 800}
	return New(Deps{Clock: world, Window: world}), world
}

func TestShowMarksStaleAndRedrawsOnlyWhenTheStackChanged(t *testing.T) {
	c, world := newWorld()
	c.Show("Disk full", severity.Error)
	if world.stale != 1 || world.redrawn != 1 {
		t.Fatalf("first toast: stale %d redrawn %d, want 1 and 1", world.stale, world.redrawn)
	}
	c.Show("Disk full", severity.Error) // the same failure again keeps the stack as it is
	if world.stale != 2 || world.redrawn != 1 {
		t.Fatalf("same toast again: stale %d redrawn %d, want 2 and 1", world.stale, world.redrawn)
	}
}

func TestPruneDropsExpiredInfoToasts(t *testing.T) {
	c, world := newWorld()
	c.Show("Saved", severity.Info)
	c.Show("Disk full", severity.Error)
	if c.Prune() || len(c.Stack()) != 2 {
		t.Fatalf("a young info toast was dropped: %+v", c.Stack())
	}
	world.now = world.now.Add(10 * time.Second)
	if !c.Prune() {
		t.Fatal("Prune did not report the expired toast")
	}
	if stack := c.Stack(); len(stack) != 1 || stack[0].Kind != widget.ToastError {
		t.Fatalf("after the time only the error is left: %+v", stack)
	}
	world.now = world.now.Add(time.Hour)
	if c.Prune() || len(c.Stack()) != 1 {
		t.Fatal("an error toast stays until it is clicked")
	}
}

// A new severity level must get a toast kind: the table may not miss one,
// or the toast would silently look like an info message.
func TestEveryLevelMapsToAToastKind(t *testing.T) {
	for _, level := range severity.All() {
		kind, ok := kinds[level]
		if !ok {
			t.Fatalf("level %v has no toast kind", level)
		}
		c, _ := newWorld()
		c.Show("message", level)
		if stack := c.Stack(); len(stack) != 1 || stack[0].Kind != kind {
			t.Errorf("level %v: toasts %+v, want kind %v", level, stack, kind)
		}
	}
	if kinds[severity.Info] == kinds[severity.Error] {
		t.Error("info and error look the same")
	}
}
