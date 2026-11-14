package mcpform

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Widths the form is pinned at: the dialog's own, and so narrow that the
// descriptions are cut.
const (
	wideW, narrowW = 500, 240
	formH          = 600
)

func newCtx(t *testing.T) *core.Ctx {
	t.Helper()
	ctx, err := core.NewCtx(core.Theme{Colors: map[core.Token]color.RGBA{core.Text: core.RGB(0xffffff)}, FontSize: 14, SmallSize: 12, MonoSize: 13})
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

// everyKind is a tool with a field of every kind, required and optional.
func everyKind() model.MCPTool {
	return model.MCPTool{Name: "every", Description: "A field of every kind, required and optional", Schema: map[string]any{
		"type":     "object",
		"required": []any{"query", "strict", "order"},
		"properties": map[string]any{
			"query":   map[string]any{"type": "string", "description": "What to find, a long description that a narrow form cuts"},
			"strict":  map[string]any{"type": "boolean", "description": "Exact matches only"},
			"order":   map[string]any{"type": "string", "enum": []any{"asc", "desc"}},
			"limit":   map[string]any{"type": "integer", "default": float64(10)},
			"safe":    map[string]any{"type": "boolean"},
			"sort":    map[string]any{"type": "string", "enum": []any{"name", "date"}, "description": "Sort key"},
			"filters": map[string]any{"type": "object"},
		},
	}}
}

// manyFields is a tool with more fields than the form shows unscrolled.
func manyFields() model.MCPTool {
	properties := map[string]any{}
	for i := range 12 {
		properties[fmt.Sprintf("field%02d", i)] = map[string]any{"type": "string"}
	}
	return model.MCPTool{Name: "many", Schema: map[string]any{"type": "object", "properties": properties}}
}

// The laid-out form is pinned as text (testdata/*.dump, captured from the
// form built in Go), one case per state and width.
func TestTheFormLaysOutAsPinned(t *testing.T) {
	cases := []struct {
		name    string
		width   int
		tool    model.MCPTool
		prepare func(form *Form)
	}{
		{"every-kind-wide", wideW, everyKind(), nil},
		{"every-kind-narrow", narrowW, everyKind(), nil},
		{"chosen-wide", wideW, everyKind(), func(form *Form) {
			form.Choices["strict"] = "true"
			form.Choices["safe"] = "false"
			form.Choices["order"] = "desc"
			form.Choices["sort"] = "date"
		}},
		{"problem-narrow", narrowW, searchTool(), func(form *Form) { form.Problem = "limit must be a number, not a word like lots" }},
		{"no-description-wide", wideW, model.MCPTool{Name: "plain", Schema: map[string]any{"type": "object", "properties": map[string]any{"text": map[string]any{"type": "string"}}}}, nil},
		{"many-wide", wideW, manyFields(), nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			form := New(tc.tool)
			if tc.prepare != nil {
				tc.prepare(form)
			}
			root := form.View(&fakeHost{})
			Layout(newCtx(t), root, tc.width, formH)
			got := Dump(root)
			want, err := os.ReadFile(filepath.Join("testdata", tc.name+".dump"))
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Fatalf("got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// findAll is every element under root, in tree order, whose own line of
// Dump has all the parts.
func findAll(root *El, parts ...string) []*El {
	var found []*El
	line, _, _ := strings.Cut(Dump(root), "\n")
	matches := true
	for _, part := range parts {
		matches = matches && strings.Contains(line, part)
	}
	if matches {
		found = append(found, root)
	}
	for _, child := range root.Children() {
		found = append(found, findAll(child, parts...)...)
	}
	return found
}

// formDriver runs the form frame by frame as the dialog does: rebuild the
// view, lay it out, deliver the input.
type formDriver struct {
	t    *testing.T
	ctx  *core.Ctx
	form *Form
	host *fakeHost
}

func newDriver(t *testing.T, tool model.MCPTool) *formDriver {
	return &formDriver{t: t, ctx: newCtx(t), form: New(tool), host: &fakeHost{}}
}

// view is the form as the next frame lays it out.
func (driver *formDriver) view() *El {
	root := driver.form.View(driver.host)
	Layout(driver.ctx, root, wideW, formH)
	return root
}

// step runs one frame; input is built from the frame's laid-out view.
func (driver *formDriver) step(input func(root *El) core.Input) {
	root := driver.view()
	driver.ctx.In = input(root)
	Update(driver.ctx, root)
}

// clickOn clicks the middle of the index-th element matching parts.
func (driver *formDriver) clickOn(index int, parts ...string) {
	driver.t.Helper()
	driver.step(func(root *El) core.Input {
		found := findAll(root, parts...)
		if len(found) <= index {
			driver.t.Fatalf("%d elements match %q, want index %d:\n%s", len(found), parts, index, Dump(root))
		}
		rect := found[index].Rect()
		return core.Input{X: rect.X + rect.W/2, Y: rect.Y + rect.H/2, Click: true}
	})
}

// Clicking a boolean's row flips it: an optional one goes from unset to
// true, then to false and back to true.
func TestClickingABooleanRowFlipsIt(t *testing.T) {
	driver := newDriver(t, searchTool())
	steps := []struct{ hint, want string }{{"not sent", "true"}, {"sent as true", "false"}, {"sent as false", "true"}}
	for _, step := range steps {
		driver.clickOn(0, fmt.Sprintf("text=%q", step.hint))
		if got := driver.form.Choices["safe"]; got != step.want {
			t.Fatalf("after clicking %q safe is %q, want %q", step.hint, got, step.want)
		}
	}
}

// A choice button picks its option; an optional enum's "none" unsets it.
func TestClickingAChoicePicksItsOption(t *testing.T) {
	driver := newDriver(t, searchTool())
	driver.clickOn(0, `text="desc"`)
	if got := driver.form.Choices["sort"]; got != "desc" {
		t.Fatalf("sort is %q, want desc", got)
	}
	driver.clickOn(0, `text="none"`)
	if got := driver.form.Choices["sort"]; got != "" {
		t.Fatalf("sort is %q, want unset", got)
	}
}

// A required enum has no "none" and starts at its first option.
func TestARequiredEnumHasNoNone(t *testing.T) {
	driver := newDriver(t, everyKind())
	if got := driver.form.Choices["order"]; got != "asc" {
		t.Fatalf("order starts at %q, want asc", got)
	}
	if found := findAll(driver.view(), `text="none"`); len(found) != 1 {
		t.Fatalf("%d none buttons, want only the optional sort's:\n%s", len(found), Dump(driver.view()))
	}
}

// Enter in a text field submits the form through the host.
func TestEnterInAFieldSubmits(t *testing.T) {
	driver := newDriver(t, searchTool())
	driver.clickOn(0, "fieldB")
	driver.step(func(*El) core.Input { return core.Input{X: 5000, Y: 5000, Typed: []rune("go"), Enter: true} })
	if driver.host.submitted != 1 {
		t.Fatalf("submitted %d times, want once", driver.host.submitted)
	}
	if got := driver.form.Inputs["query"].Text; got != "go" {
		t.Fatalf("query %q, want what was typed", got)
	}
}

// What Dump does not print: which field edits which input, its text menu
// and Enter handler, and whether a box is ticked.
func TestTheFormsUnprintedParts(t *testing.T) {
	form := New(everyKind())
	form.Choices["strict"] = "true"
	form.Choices["safe"] = "false"
	root := form.View(&fakeHost{})
	var fields, boxes []string
	var walk func(node *El)
	walk = func(node *El) {
		switch kind := fmt.Sprintf("%T", node.Behavior()); kind {
		case "*widget.fieldB":
			fields = append(fields, fmt.Sprintf("%+v", node.Behavior()))
		case "widget.checkB", "*widget.checkB":
			boxes = append(boxes, fmt.Sprintf("%+v", node.Behavior()))
		}
		for _, child := range node.Children() {
			walk(child)
		}
	}
	walk(root)
	order := []string{"query", "filters", "limit"} // the text fields in form order
	if len(fields) != len(order) {
		t.Fatalf("%d fields, want %d:\n%s", len(fields), len(order), Dump(root))
	}
	for i, name := range order {
		for _, want := range []string{fmt.Sprintf("%p", form.Inputs[name]), "onEnter:0x"} {
			if !strings.Contains(fields[i], want) {
				t.Errorf("field %d (%s) = %s, want %q in it", i, name, fields[i], want)
			}
		}
	}
	if len(boxes) != 2 || !strings.Contains(boxes[0], "on:true") || !strings.Contains(boxes[1], "on:false") {
		t.Errorf("boxes %q, want strict ticked and safe not", boxes)
	}
	if found := findAll(root, `text="sent as false"`); len(found) != 1 {
		t.Errorf("safe's hint is not shown:\n%s", Dump(root))
	}
}
