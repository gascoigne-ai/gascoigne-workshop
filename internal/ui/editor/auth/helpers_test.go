package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Widths the pane is pinned at: the main column of a 1280px window with the
// default sidebar, and the narrowest the dividers allow; the height the
// request pane has in the test window.
const (
	wideW   = 1020
	narrowW = 420
	paneH   = 300
)

// findWidgets is every node under root whose behavior has the Go type
// kind, in tree order.
func findWidgets(root Node, kind string) []Node {
	var found []Node
	if fmt.Sprintf("%T", root.Behavior()) == kind {
		found = append(found, root)
	}
	for _, child := range root.Children() {
		found = append(found, findWidgets(child, kind)...)
	}
	return found
}

// behaviorStates prints the behavior of every node under root whose
// behavior has the Go type kind, in order.
func behaviorStates(root Node, kind string) []string {
	var states []string
	for _, node := range findWidgets(root, kind) {
		states = append(states, fmt.Sprintf("%+v", node.Behavior()))
	}
	return states
}

// chipLabelled is the element whose only child is the text label, nil
// when there is none.
func chipLabelled(root Node, label string) Node {
	if kids := root.Children(); len(kids) == 1 && strings.HasPrefix(Dump(kids[0]), `textB text="`+label+`" `) {
		return root
	}
	for _, child := range root.Children() {
		if found := chipLabelled(child, label); found != nil {
			return found
		}
	}
	return nil
}

// assertDumpPinned lays root out in a window of width by height and
// compares its dump with testdata/name.dump.
func assertDumpPinned(t *testing.T, ctx *core.Ctx, root Node, name string, width, height int) {
	t.Helper()
	Layout(ctx, root, width, height)
	want, err := os.ReadFile(filepath.Join("testdata", name+".dump"))
	if err != nil {
		t.Fatal(err)
	}
	if got := Dump(root); got != string(want) {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// view builds the Auth pane of the open tab.
func (world *world) view() Node {
	var scroll widget.ScrollState
	return world.pane.View(world.ctx, world.tab, &scroll)
}

// frame builds the pane, lays it out at the wide size and runs one frame of
// input over it, as the window does each frame; it returns the pane's tree.
func (world *world) frame(in core.Input, pick func(root Node) Node) Node {
	world.t.Helper()
	root := world.view()
	Layout(world.ctx, root, wideW, paneH)
	if pick != nil {
		target := pick(root)
		if target == nil {
			world.t.Fatal("nothing to point at in the pane")
		}
		at := target.Rect()
		in.X, in.Y = at.X+at.W/2, at.Y+at.H/2
	}
	world.ctx.In = in
	Update(world.ctx, root)
	return root
}

// clickOn presses the node pick finds and lets go.
func (world *world) clickOn(pick func(Node) Node) {
	world.t.Helper()
	world.frame(core.Input{Click: true, Held: true}, pick)
	world.frame(core.Input{}, nil)
}

// click presses the chip labelled label and lets go.
func (world *world) click(label string) {
	world.t.Helper()
	world.clickOn(func(root Node) Node { return chipLabelled(root, label) })
}

// toggleShowSecrets clicks the Show secrets check.
func (world *world) toggleShowSecrets() {
	world.t.Helper()
	world.clickOn(func(root Node) Node { return findWidgets(root, "widget.checkB")[0] })
}

// fieldAt is the pick of the nth text field of the pane.
func fieldAt(n int) func(Node) Node {
	return func(root Node) Node {
		fields := findWidgets(root, "*widget.fieldB")
		if n >= len(fields) {
			return nil
		}
		return fields[n]
	}
}

// typeInto clicks the nth text field and types text in it.
func (world *world) typeInto(n int, text string) {
	world.t.Helper()
	world.frame(core.Input{Click: true, Held: true}, fieldAt(n))
	world.frame(core.Input{}, nil)
	world.frame(core.Input{Typed: []rune(text)}, nil)
}
