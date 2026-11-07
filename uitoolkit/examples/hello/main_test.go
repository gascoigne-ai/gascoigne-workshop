package main

import (
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/theming"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// The example's view builds, lays out and reacts to its state.
func TestHelloViewFollowsItsState(t *testing.T) {
	ctx, err := core.NewCtx(theming.Dark())
	if err != nil {
		t.Fatal(err)
	}
	g := &hello{ctx: ctx, name: NewTextField("")}
	root := g.view()
	Layout(ctx, root, 480, 320)
	if d := Dump(root); !strings.Contains(d, `"Nobody yet"`) {
		t.Fatalf("empty state:\n%s", d)
	}
	g.name = NewTextField("Ada")
	g.greet()
	if d := Dump(g.view()); !strings.Contains(d, `"Hello, Ada!"`) || strings.Contains(d, "Nobody yet") {
		t.Fatalf("after greeting:\n%s", d)
	}
}
