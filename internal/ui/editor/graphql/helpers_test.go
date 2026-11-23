package graphql

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/theme"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// Widths the pane is pinned at: the main column of a 1280px window with the
// sidebar open and the narrowest main area, and the height of the pane.
const (
	wideW   = 1020
	narrowW = 420
	paneH   = 300
)

// introspection is an introspection answer with one operation, shaped from
// the GraphQL specification, not captured from a server.
const introspection = `{"data":{"__schema":{"queryType":{"name":"Query"},"mutationType":null,"types":[
 {"kind":"OBJECT","name":"Query","fields":[
  {"name":"repo","args":[{"name":"name","type":{"kind":"NON_NULL","name":null,"ofType":{"kind":"SCALAR","name":"String","ofType":null}}}],"type":{"kind":"OBJECT","name":"Repo","ofType":null}}]},
 {"kind":"OBJECT","name":"Repo","fields":[{"name":"name","args":[],"type":{"kind":"SCALAR","name":"String","ofType":null}}]}]}}}`

// pinnedSchema is a loaded schema with two operations, so the Schema
// button shows its count.
var pinnedSchema = &send.Schema{Operations: []send.Operation{
	{Label: "query repo(name: String!): Repo", Query: "query($name: String!) { repo(name: $name) { name } }", Variables: `{"name": ""}`},
	{Label: "query viewer: User", Query: "{ viewer { login } }"},
}}

// menu is one menu the component opened.
type menu struct {
	posX, posY int
	items      []widget.MenuItem
}

// sent is one request the component asked the host to send, and for which
// tab.
type sent struct {
	tab *tabs.Tab
	req request.Request
}

// fake is the host of the component: real tabs, the body's query areas, and
// a record of what the component asked of it.
type fake struct {
	set   *tabs.Set
	areas map[*tabs.Tab]*widget.TextArea
	sends []sent
	menus []menu
}

func (fake *fake) Current() *tabs.Tab        { return fake.set.Current() }
func (fake *fake) IsOpen(tab *tabs.Tab) bool { return slices.Contains(fake.set.Tabs(), tab) }
func (fake *fake) Area(tab *tabs.Tab) *widget.TextArea {
	area, ok := fake.areas[tab]
	if !ok {
		made := widget.NewTextArea(tab.Req.Body)
		area = &made
		fake.areas[tab] = area
	}
	return area
}
func (fake *fake) Dispatch(tab *tabs.Tab, req request.Request) {
	fake.sends = append(fake.sends, sent{tab, req})
}
func (fake *fake) Open(posX, posY int, items []widget.MenuItem) {
	fake.menus = append(fake.menus, menu{posX, posY, items})
}

// world is the component over a fake, with one open request whose body is
// GraphQL.
type world struct {
	t    *testing.T
	ctx  *core.Ctx
	fake *fake
	pane *Component
	tab  *tabs.Tab
}

func newWorld(t *testing.T) *world {
	t.Helper()
	ctx, err := core.NewCtx(theme.ThemeWith(settings.Default()))
	if err != nil {
		t.Fatal(err)
	}
	fake := &fake{set: tabs.NewSet(), areas: map[*tabs.Tab]*widget.TextArea{}}
	fake.set.Open(&request.Request{Name: "R", Protocol: request.ProtocolREST, Method: "POST", URL: "http://example.test/graphql"})
	pane := New(Deps{Tabs: fake, Query: fake, Sends: fake, Window: fake})
	tab := fake.set.Current()
	*tab.Req = requestedit.Update(*tab.Req, requestedit.BodyTypeChosen{Type: request.BodyGraphQL})
	return &world{t: t, ctx: ctx, fake: fake, pane: pane, tab: tab}
}

// view builds the GraphQL editor of the open tab.
func (world *world) view() Node { return world.pane.View(world.ctx, world.tab) }

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

// buttonLabelled is the element whose only child is the text label, nil
// when there is none.
func buttonLabelled(root Node, label string) Node {
	if kids := root.Children(); len(kids) == 1 && strings.HasPrefix(Dump(kids[0]), `textB text="`+label+`" `) {
		return root
	}
	for _, child := range root.Children() {
		if found := buttonLabelled(child, label); found != nil {
			return found
		}
	}
	return nil
}

// click presses the button labelled label and lets go, each frame on a
// freshly built and laid out editor, as the window does.
func (world *world) click(label string) {
	world.t.Helper()
	root := world.view()
	Layout(world.ctx, root, wideW, paneH)
	target := buttonLabelled(root, label)
	if target == nil {
		world.t.Fatalf("no button %q in:\n%s", label, Dump(root))
	}
	at := target.Rect()
	x, y := at.X+at.W/2, at.Y+at.H/2
	for _, in := range []core.Input{{X: x, Y: y, Click: true, Held: true}, {X: x, Y: y}} {
		root = world.view()
		Layout(world.ctx, root, wideW, paneH)
		world.ctx.In = in
		Update(world.ctx, root)
	}
}
