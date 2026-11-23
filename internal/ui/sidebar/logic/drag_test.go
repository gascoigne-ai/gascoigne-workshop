package logic

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/collections/transfer"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
)

func TestDragTransitions(t *testing.T) {
	fix := newFixture()
	apiRow := collection.Row{Name: "api", Folder: true, Dir: fix.api}
	rootRow := collection.Row{Name: "root", Folder: true, Dir: fix.root}
	homeRow := collection.Row{Name: "home", Req: fix.home}
	apiNode, rootNode, homeNode := apiRow.Node(), rootRow.Node(), homeRow.Node()

	tests := []struct {
		name  string
		drive func(tree *Tree) bool // returns the alt flag of the release
		want  Outcome
	}{
		{"press a request and release on itself does nothing", func(tree *Tree) bool {
			tree.Press(homeRow, false)
			tree.Over(homeNode)
			return false
		}, Outcome{}},
		{"press a folder and release on itself toggles it", func(tree *Tree) bool {
			tree.Press(apiRow, false)
			tree.Over(apiNode)
			return false
		}, Outcome{Kind: Toggle}},
		{"press a root and release on itself toggles it", func(tree *Tree) bool {
			tree.Press(rootRow, true)
			tree.Over(rootNode)
			return false
		}, Outcome{Kind: Toggle}},
		{"press then over another row moves", func(tree *Tree) bool {
			tree.Press(homeRow, false)
			tree.Over(apiNode)
			return false
		}, Outcome{Kind: Drop, Node: homeNode, Target: apiNode, Mode: transfer.Move}},
		{"the same with alt copies", func(tree *Tree) bool {
			tree.Press(homeRow, false)
			tree.Over(apiNode)
			return true
		}, Outcome{Kind: Drop, Node: homeNode, Target: apiNode, Mode: transfer.Copy}},
		{"back over the pressed folder toggles it", func(tree *Tree) bool {
			tree.Press(apiRow, false)
			tree.Over(homeNode)
			tree.Over(apiNode)
			return false
		}, Outcome{Kind: Toggle}},
		{"a cleared target does nothing", func(tree *Tree) bool {
			tree.Press(homeRow, false)
			tree.Over(apiNode)
			tree.Clear()
			return false
		}, Outcome{}},
		{"a root cannot be dropped anywhere", func(tree *Tree) bool {
			tree.Press(rootRow, true)
			tree.Over(apiNode)
			return false
		}, Outcome{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tree := New()
			alt := test.drive(tree)
			got := tree.Release(alt)
			if got != test.want {
				t.Errorf("outcome %+v, want %+v", got, test.want)
			}
			if tree.Drag().Phase() != Idle {
				t.Errorf("phase %v after the release, want Idle", tree.Drag().Phase())
			}
			if !tree.Focused() {
				t.Error("a press must give the tree the keyboard")
			}
		})
	}
}

func TestPressSelectsTheRowAndAToggleOpensTheFolder(t *testing.T) {
	fix := newFixture()
	apiRow := collection.Row{Name: "api", Folder: true, Dir: fix.api}
	tree := New()
	tree.Press(apiRow, false)
	if tree.Selected() != apiRow.Node() {
		t.Errorf("selected %v", tree.Selected())
	}
	tree.Over(apiRow.Node())
	tree.Release(false)
	if !tree.Expanded()[fix.api] {
		t.Error("the release on the pressed folder did not open it")
	}
}

func TestPhaseIsComputedFromTheDrag(t *testing.T) {
	fix := newFixture()
	home := request.Node{Req: fix.home}
	api := request.Node{Dir: fix.api}
	tests := []struct {
		name string
		drag Drag
		want Phase
	}{
		{"nothing held", Drag{}, Idle},
		{"pressed with no target", Drag{Node: home}, Pressed},
		{"over its own row", Drag{Node: home, Target: home}, Pressed},
		{"over another row", Drag{Node: home, Target: api}, Dragging},
		{"a root press over a row is never a drag", Drag{Pressed: fix.root, Target: api}, Pressed},
	}
	for _, test := range tests {
		if got := test.drag.Phase(); got != test.want {
			t.Errorf("%s: phase %v, want %v", test.name, got, test.want)
		}
	}
}

func TestHoverOpensAClosedFolderAfter30Ticks(t *testing.T) {
	fix := newFixture()
	homeRow := collection.Row{Name: "home", Req: fix.home}
	press := func(tree *Tree, over request.Node) {
		tree.Press(homeRow, false)
		tree.Over(over)
	}
	t.Run("29 ticks change nothing, the 30th opens", func(t *testing.T) {
		tree := New()
		press(tree, request.Node{Dir: fix.api})
		for tick := 1; tick < 30; tick++ {
			if tree.Tick() || tree.Expanded()[fix.api] {
				t.Fatalf("tick %d opened the folder", tick)
			}
		}
		if !tree.Tick() || !tree.Expanded()[fix.api] {
			t.Fatal("tick 30 did not open the folder")
		}
	})
	t.Run("an open folder is left alone", func(t *testing.T) {
		tree := New()
		tree.Expand(fix.api)
		press(tree, request.Node{Dir: fix.api})
		for tick := 0; tick < 40; tick++ {
			if tree.Tick() {
				t.Fatal("an open folder reported a change")
			}
		}
	})
	t.Run("the dragged folder itself opens nothing", func(t *testing.T) {
		tree := New()
		tree.Press(collection.Row{Name: "api", Folder: true, Dir: fix.api}, false)
		tree.Over(request.Node{Dir: fix.api})
		for tick := 0; tick < 40; tick++ {
			tree.Tick()
		}
		if tree.Expanded()[fix.api] {
			t.Fatal("hovering the dragged folder opened it")
		}
	})
	t.Run("with no dragged node nothing opens", func(t *testing.T) {
		tree := New()
		tree.Over(request.Node{Dir: fix.api})
		for tick := 0; tick < 40; tick++ {
			tree.Tick()
		}
		if tree.Expanded()[fix.api] {
			t.Fatal("a hover without a drag opened the folder")
		}
	})
}
