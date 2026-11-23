package response

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/viewer"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

func TestStateIsMadeOnFirstUseWithTheSettingsView(t *testing.T) {
	cases := []struct {
		name     string
		setting  string
		wantMode viewer.Mode
	}{
		{"pretty", "pretty", viewer.Pretty},
		{"raw", "raw", viewer.Raw},
		{"compact", "compact", viewer.Compact},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			world := newWorld(t)
			world.fake.mode = tc.setting
			look := world.pane.Look(world.tab)
			if look.Mode != tc.wantMode || look.Match != -1 || look.FindOpen {
				t.Fatalf("look %+v, want mode %v, no match, no find bar", look, tc.wantMode)
			}
			if world.pane.Code(world.tab) != world.pane.Code(world.tab) || world.pane.Field(world.tab) != world.pane.Field(world.tab) {
				t.Fatal("the state must keep its place between calls: views hold pointers into it")
			}
		})
	}
}

func TestTickDropsTheStateOfAClosedTab(t *testing.T) {
	world := newWorld(t)
	first := world.tab
	world.fake.set.Open(&request.Request{Name: "second"})
	world.pane.Field(first).Insert("typed")
	world.fake.set.Close(0)
	if world.pane.Tick() {
		t.Fatal("dropping a state changes nothing on screen")
	}
	if len(world.pane.states) != 0 {
		t.Fatalf("%d states left, want none", len(world.pane.states))
	}
}

func TestKeysClaimsTypingAndUndoOnlyWhileTheFindFieldIsFocused(t *testing.T) {
	cases := []struct {
		name    string
		focused bool
		hidden  bool // the focused field belongs to a tab that is not shown
		input   core.Input
		want    bool
	}{
		{"letter while unfocused", false, false, core.Input{Keys: []ebiten.Key{ebiten.KeyQ}}, false},
		{"letter while focused", true, false, core.Input{Keys: []ebiten.Key{ebiten.KeyQ}}, true},
		{"undo while focused", true, false, core.Input{Keys: []ebiten.Key{ebiten.KeyZ}, Ctrl: true, Undo: true}, true},
		{"redo while focused", true, false, core.Input{Keys: []ebiten.Key{ebiten.KeyY}, Ctrl: true, Redo: true}, true},
		{"undo while unfocused", false, false, core.Input{Keys: []ebiten.Key{ebiten.KeyZ}, Ctrl: true, Undo: true}, false},
		{"escape while focused", true, false, core.Input{Keys: []ebiten.Key{ebiten.KeyEscape}, Escape: true}, false},
		{"ctrl+enter while focused", true, false, core.Input{Keys: []ebiten.Key{ebiten.KeyEnter}, Ctrl: true}, false},
		{"ctrl+f while focused", true, false, core.Input{Keys: []ebiten.Key{ebiten.KeyF}, Ctrl: true}, false},
		{"no keys while focused", true, false, core.Input{}, false},
		{"letter, field of a hidden tab", true, true, core.Input{Keys: []ebiten.Key{ebiten.KeyQ}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			world := newWorld(t)
			world.pane.Field(world.tab).Focused = tc.focused
			if tc.hidden {
				world.fake.set.Open(&request.Request{Name: "second"})
			}
			if got := world.pane.Keys(tc.input); got != tc.want {
				t.Fatalf("Keys = %v, want %v", got, tc.want)
			}
			if got := world.pane.Typing(); got != (tc.focused && !tc.hidden) {
				t.Fatalf("Typing = %v with focused %v hidden %v", got, tc.focused, tc.hidden)
			}
		})
	}
}

func TestTypingAsksWithoutMakingState(t *testing.T) {
	world := newWorld(t)
	if world.pane.Typing() || world.pane.Keys(core.Input{Keys: []ebiten.Key{ebiten.KeyQ}}) {
		t.Fatal("nothing is focused at the start")
	}
	if len(world.pane.states) != 0 {
		t.Fatal("asking whether the user types must not make state")
	}
}
