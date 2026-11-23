package graphql

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// graphQLBodyStates are the looks of the GraphQL body editor: no schema
// and no variables, a loaded schema, and valid or invalid variables.
var graphQLBodyStates = []struct {
	name  string
	setUp func(world *world)
}{
	{"empty", func(world *world) {}},
	{"schema", func(world *world) { world.pane.state(world.tab).schema = pinnedSchema }},
	{"valid-vars", func(world *world) { world.pane.state(world.tab).vars = widget.NewTextArea(`{"name": "gio"}`) }},
	{"invalid-vars", func(world *world) { world.pane.state(world.tab).vars = widget.NewTextArea(`{"name": `) }},
}

// The GraphQL body editor is pinned as text (element kinds, options,
// rects) for each state at the request pane's width and at the narrowest
// main area, so moving it to a template cannot change it unseen.
func TestTheGraphQLBodyLaysOutAsPinned(t *testing.T) {
	sizes := []struct {
		name  string
		width int
	}{
		{"wide", wideW},
		{"narrow", narrowW},
	}
	for _, state := range graphQLBodyStates {
		for _, size := range sizes {
			name := "graphql-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				world := newWorld(t)
				state.setUp(world)
				root := world.view()
				Layout(world.ctx, root, size.width, paneH)
				want, err := os.ReadFile(filepath.Join("testdata", name+".dump"))
				if err != nil {
					t.Fatal(err)
				}
				if got := Dump(root); got != string(want) {
					t.Fatalf("got:\n%s\nwant:\n%s", got, want)
				}
			})
		}
	}
}

// What Dump does not show of the two areas: the query edits the tab's
// body and the variables its own area. The query is drawn without the
// JSON styler a JSON body leaves on the shared body area.
func TestTheGraphQLAreasEditTheTabsTexts(t *testing.T) {
	world := newWorld(t)
	query := world.fake.Area(world.tab)
	query.Styler = func(line string) []widget.Span { return []widget.Span{{Text: line}} } // what a JSON body leaves behind
	areas := behaviorStates(world.view(), "*widget.areaB")
	if len(areas) != 2 {
		t.Fatalf("%d areas, want 2: %q", len(areas), areas)
	}
	for i, area := range []*widget.TextArea{query, &world.pane.state(world.tab).vars} {
		if want := fmt.Sprintf("%p", area); !strings.Contains(areas[i], want) {
			t.Errorf("area %d lacks %s: %s", i, want, areas[i])
		}
	}
	if query.Styler != nil {
		t.Error("the query area keeps a styler")
	}
}

// The variables area starts with the request's variables.
func TestTheVariablesAreaStartsWithTheRequestsVariables(t *testing.T) {
	world := newWorld(t)
	*world.tab.Req = requestedit.Update(*world.tab.Req, requestedit.VarsTyped{Text: `{"a": 1}`})
	world.view()
	if got := world.pane.state(world.tab).vars.Text; got != `{"a": 1}` {
		t.Fatalf("variables area %q, want the request's", got)
	}
}
