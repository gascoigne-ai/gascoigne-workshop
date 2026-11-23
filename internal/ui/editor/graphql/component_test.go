package graphql

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
)

// Typing is true only while the variables area of the shown tab has the
// keyboard, and asking makes no state.
func TestTypingIsTheFocusOfTheVariablesOfTheShownTab(t *testing.T) {
	t.Run("a tab that never showed the pane", func(t *testing.T) {
		world := newWorld(t)
		if world.pane.Typing() {
			t.Fatal("typing without a state")
		}
		if len(world.pane.states) != 0 {
			t.Fatal("asking made a state")
		}
	})
	t.Run("focus in the variables", func(t *testing.T) {
		world := newWorld(t)
		world.pane.state(world.tab).vars.Focused = true
		if !world.pane.Typing() {
			t.Fatal("the variables have the keyboard but Typing is false")
		}
	})
	t.Run("focus in a hidden tab", func(t *testing.T) {
		world := newWorld(t)
		world.pane.state(world.tab).vars.Focused = true
		world.fake.set.Open(&request.Request{Name: "Other"})
		if world.pane.Typing() {
			t.Fatal("the focus of a hidden tab counts")
		}
	})
	t.Run("no tab shown", func(t *testing.T) {
		world := newWorld(t)
		world.fake.set = tabs.NewSet()
		if world.pane.Typing() {
			t.Fatal("typing with no tab")
		}
	})
}

// A closed tab's state, its variables and schema with it, goes in Tick.
func TestTickDropsTheStateOfAClosedTab(t *testing.T) {
	world := newWorld(t)
	world.pane.state(world.tab).schema = pinnedSchema
	if world.pane.Tick() || len(world.pane.states) != 1 {
		t.Fatalf("an open tab lost its state: %d states", len(world.pane.states))
	}
	world.fake.set.Close(0)
	if world.pane.Tick() {
		t.Fatal("Tick reports a change on screen")
	}
	if len(world.pane.states) != 0 {
		t.Fatalf("%d states left after the tab closed", len(world.pane.states))
	}
}
