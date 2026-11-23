package body

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
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

// findWidget is the first node findWidgets finds, nil when there is none.
func findWidget(root Node, kind string) Node {
	if found := findWidgets(root, kind); len(found) > 0 {
		return found[0]
	}
	return nil
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
