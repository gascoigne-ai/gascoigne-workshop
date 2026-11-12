package markup

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// The values `if` has always read keep their meaning.
func TestIfReadsTheScopeValuesItAlwaysRead(t *testing.T) {
	tests := []struct {
		name  string
		value any
		shown bool
	}{
		{"untyped nil", nil, false},
		{"true", true, true},
		{"false", false, false},
		{"text", "a", true},
		{"empty text", "", false},
		{"whole number", 2, true},
		{"zero", 0, false},
		{"number", 0.5, true},
		{"zero number", 0.0, false},
		{"list", []any{1}, true},
		{"empty list", []any{}, false},
		{"text list", []string{"a"}, true},
		{"empty text list", []string{}, false},
		{"map list", []map[string]any{{}}, true},
		{"empty map list", []map[string]any{}, false},
		{"map", map[string]any{}, true},
		{"func", func() {}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			node := build(t, `<row><text if="value">x</text></row>`, Scope{"value": test.value})
			if shown := strings.Contains(view.Dump(node), `text="x"`); shown != test.shown {
				t.Errorf("if on %#v shows the text: %v, want %v", test.value, shown, test.shown)
			}
		})
	}
}

type listRequest struct{ Method string }

type listMode string

type listChild struct{ Label string }

// listRow is a typed list item as a view model hands it to a template:
// plain fields, a pointer that may be nil, a named string, a nested list
// and the row's own handler.
type listRow struct {
	Name     string
	Selected bool
	Req      *listRequest
	Mode     listMode
	Children []listChild
	Open     func()
}

// rowHeight is the height of one row of the list views below.
const rowHeight = 22

// goListRow is the Go view listTemplate stands for.
func goListRow(index int, row listRow) view.Node {
	method := ""
	if row.Req != nil {
		method = row.Req.Method
	}
	return view.Row(view.H(rowHeight), view.OnClick(row.Open),
		view.Text(fmt.Sprintf("%d %s", index, row.Name)),
		view.If(row.Selected, view.Text("selected", view.Bold())),
		view.If(row.Req != nil, view.Text(method, view.Small())),
		view.If(row.Mode != "", view.Text(string(row.Mode), view.Muted())),
		view.For(row.Children, func(_ int, child listChild) view.Node { return view.Text(child.Label) }),
	)
}

const listTemplate = `
	<col>
	  <row for="row in rows" h="22" on-click="row.Open">
	    <text>{{$index}} {{row.Name}}</text>
	    <text if="row.Selected" bold>selected</text>
	    <text if="row.Req" small>{{row.Req.Method}}</text>
	    <text if="row.Mode" muted>{{row.Mode}}</text>
	    <text for="child in row.Children">{{child.Label}}</text>
	  </row>
	</col>`

// A for over a slice of structs, or of pointers to them, builds what
// view.For builds: fields are read by name; if hides on a nil pointer, an
// empty named string and a false bool; a nested for walks a field; and
// each row's handler is its own.
func TestForOverTypedStructsMatchesFor(t *testing.T) {
	var opened []string
	rows := []listRow{
		{Name: "users", Selected: true, Req: &listRequest{Method: "GET"}, Children: []listChild{{"a"}, {"b"}}},
		{Name: "orders", Mode: "draft"},
	}
	for i := range rows {
		name := rows[i].Name
		rows[i].Open = func() { opened = append(opened, name) }
	}
	pointers := []*listRow{&rows[0], &rows[1]}
	for _, list := range []struct {
		name  string
		value any
	}{{"values", rows}, {"pointers", pointers}} {
		t.Run(list.name, func(t *testing.T) {
			opened = nil
			ctx := testCtx(t)
			template := build(t, listTemplate, Scope{"rows": list.value})
			sameLayout(t, ctx, template, view.Col(view.For(rows, goListRow)), 400, 200)
			ctx.In = core.Input{X: 5, Y: rowHeight + rowHeight/2, Click: true}
			view.Update(ctx, template)
			if len(opened) != 1 || opened[0] != "orders" {
				t.Errorf("a click on the second row opened %v, want [orders]", opened)
			}
		})
	}
}

// A struct path that cannot be read is a build error naming the path:
// a field the struct does not export, one it does not have, and a nil
// pointer met on the way.
func TestUnreadableStructPathIsABuildError(t *testing.T) {
	type hidden struct{ name string }
	tests := []struct {
		name  string
		src   string
		scope Scope
		want  string
	}{
		{"unexported field", `<text>{{row.name}}</text>`, Scope{"row": hidden{"a"}}, `unknown name "row.name"`},
		{"missing field", `<text>{{row.Title}}</text>`, Scope{"row": listRow{}}, `unknown name "row.Title"`},
		{"nil pointer on the way", `<text>{{row.Req.Method}}</text>`, Scope{"row": listRow{}}, `"row.Req.Method": row.Req is nil`},
		{"nil struct pointer", `<text>{{row.Name}}</text>`, Scope{"row": (*listRow)(nil)}, `"row.Name": row is nil`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := buildError(t, test.src, test.scope); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Errorf("err = %v, want %q", err, test.want)
			}
		})
	}
}
