package kvtable

import (
	"image/color"
	"slices"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/kvtable/logic"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// fake is every dependency of the tables at once, recording what it was asked.
type fake struct {
	open    []*tabs.Tab
	visible logic.Table
	shown   bool
	rows    map[logic.Table]*[]request.KV
	changed []logic.Table
	menus   [][]widget.MenuItem
	secret  map[int]bool // rows of the Vars table that can be marked, and whether they are
	toggled []int
}

type host struct{ *fake }

func (host host) Rows(table logic.Table) *[]request.KV {
	if rows, ok := host.rows[table]; ok {
		return rows
	}
	return &table.Tab.Req.Params
}
func (host host) Changed(table logic.Table) { host.changed = append(host.changed, table) }
func (host host) IsOpen(tab *tabs.Tab) bool { return slices.Contains(host.open, tab) }
func (host host) Visible() (logic.Table, bool) {
	return host.visible, host.shown
}
func (host host) Open(posX, posY int, items []widget.MenuItem) {
	host.menus = append(host.menus, items)
}
func (host host) Secret(table logic.Table, row int) (secret, ok bool) {
	secret, ok = host.fake.secret[row]
	return secret, ok && table.Kind == logic.Vars
}
func (host host) ToggleSecret(row int) { host.toggled = append(host.toggled, row) }
func (host host) CellStyler(*request.Request) func(string) []widget.Span {
	return nil
}

// world is a component over one open tab whose Params are q, order and sort.
type world struct {
	t      *testing.T
	ctx    *core.Ctx
	kv     *Component
	fake   *fake
	tab    *tabs.Tab
	params logic.Table
	look   Look
	scroll widget.ScrollState
}

func newWorld(t *testing.T) *world {
	t.Helper()
	ctx, err := core.NewCtx(core.Theme{Colors: map[core.Token]color.RGBA{core.Text: core.RGB(0xffffff)}, FontSize: 14, SmallSize: 12, MonoSize: 13})
	if err != nil {
		t.Fatal(err)
	}
	set := tabs.NewSet()
	set.Open(&request.Request{Name: "search", Params: []request.KV{
		{Name: "q", Value: "gioui", Description: "repo name", Enabled: true},
		{Name: "order", Value: "desc", Enabled: true},
		{Name: "sort", Value: "stars", Enabled: true},
	}})
	tab := set.Current()
	params := logic.Table{Tab: tab, Kind: logic.Params}
	fake := &fake{open: []*tabs.Tab{tab}, visible: params, shown: true, secret: map[int]bool{}}
	var h = host{fake}
	kv := New(Deps{Tables: h, Tabs: h, Menus: h, Secrets: h, Styles: h})
	return &world{t: t, ctx: ctx, kv: kv, fake: fake, tab: tab, params: params, look: paramsLook}
}

const (
	tableW, tableH = 640, 400
	paramsCaption  = "Query"
)

var paramsLook = Look{Caption: paramsCaption, Editable: true}

// tree builds the Params table and lays it out.
func (w *world) tree() Node {
	root := w.kv.View(w.ctx, w.params, w.look, &w.scroll, 0)
	Layout(w.ctx, root, tableW, tableH)
	return root
}

// frame is one frame: build, react to input, then the per-frame step.
func (w *world) frame(in core.Input) bool {
	w.t.Helper()
	w.ctx.Tick++
	w.ctx.In = in
	w.kv.BeginFrame(in)
	root := w.tree()
	Update(w.ctx, root)
	return w.kv.Tick()
}

// pointer is the center of the first element whose own Dump line has part.
func (w *world) pointer(part string) (int, int) {
	w.t.Helper()
	node := find(w.tree(), part)
	if node == nil {
		w.t.Fatalf("no element with %s in:\n%s", part, Dump(w.tree()))
	}
	rect := node.Rect()
	return rect.X + rect.W/2, rect.Y + rect.H/2
}

// click presses and releases on the element with part, in two frames.
func (w *world) click(part string) {
	w.t.Helper()
	x, y := w.pointer(part)
	w.frame(core.Input{X: x, Y: y, Click: true, Held: true})
	w.frame(core.Input{X: x, Y: y})
}

// typeInto replaces the text of the cell being edited.
func (w *world) typeInto(text string) {
	w.t.Helper()
	if w.kv.edit == nil {
		w.t.Fatal("no cell is being edited")
	}
	w.kv.edit.field.SelectAll()
	w.kv.edit.field.Insert(text)
}

func (w *world) names() []string {
	var out []string
	for _, row := range w.tab.Req.Params {
		out = append(out, row.Name)
	}
	return out
}

func find(node Node, part string) Node {
	line, _, _ := strings.Cut(Dump(node), "\n")
	if strings.Contains(line, part) {
		return node
	}
	for _, child := range node.Children() {
		if found := find(child, part); found != nil {
			return found
		}
	}
	return nil
}
