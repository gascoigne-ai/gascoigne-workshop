package body

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// bodyPaneStates are the looks of the body pane: each editor kind, and
// for JSON a valid, an invalid (a long parse error) and an empty body.
// The Form look is pinned by the app, with the real table. File is last in
// the chip row, so at the narrow width it pins the row scrolled to keep the
// chosen chip in view.
var bodyPaneStates = []struct {
	name     string
	bodyType request.BodyType
	text     string
}{
	{"none", request.BodyNone, ""},
	{"json-valid", request.BodyJSON, "{\n  \"hello\": \"world\"\n}"},
	{"json-invalid", request.BodyJSON, `{"hello": "world", "list": [1, 2`},
	{"json-empty", request.BodyJSON, ""},
	{"text", request.BodyText, "hello"},
	{"file", request.BodyFile, ""},
}

// The body pane is pinned as text (element kinds, options, rects) for
// each state at two widths, so moving it to a template cannot change it
// unseen.
func TestTheBodyPaneLaysOutAsPinned(t *testing.T) {
	for _, state := range bodyPaneStates {
		for _, size := range paneSizes {
			name := "bodypane-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				world := open(t, state.bodyType, state.text)
				assertDumpPinned(t, world.ctx, world.view(), name, size.width, paneH)
			})
		}
	}
}

// What Dump does not show of the chip row: it scrolls sideways with the
// tab's own state.
func TestTheBodyTypeChipsScrollWithTheTabsState(t *testing.T) {
	world := open(t, request.BodyJSON, "")
	chips := findWidget(world.view(), "*widget.hscrollB")
	if chips == nil {
		t.Fatal("the body pane has no sideways scroll")
	}
	want := fmt.Sprintf("%p", &world.pane.state(world.tab).typesX)
	if settings := fmt.Sprintf("%+v", chips.Behavior()); !strings.Contains(settings, want) {
		t.Fatalf("the chip row lacks the tab's state %s: %s", want, settings)
	}
}

// What Dump does not show of the chip row: only the chosen type's chip is
// kept in view, so choosing a type elsewhere scrolls to it.
func TestOnlyTheChosenBodyTypeChipIsKeptInView(t *testing.T) {
	choices := []struct {
		bodyType request.BodyType
		chip     int // its place in the row: None, JSON, Text, Form, Multipart, GraphQL, File
	}{
		{request.BodyNone, 0},
		{request.BodyFile, 6},
	}
	for _, chosen := range choices {
		t.Run(string(chosen.bodyType), func(t *testing.T) {
			world := open(t, chosen.bodyType, "")
			chips := findWidget(world.view(), "*widget.hscrollB")
			var kept []int
			for i, chip := range chips.Children() {
				if chip.KeepsInView() {
					kept = append(kept, i)
				}
			}
			if !slices.Equal(kept, []int{chosen.chip}) {
				t.Fatalf("chips kept in view %v, want only %d", kept, chosen.chip)
			}
		})
	}
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

// The Wrap chip of the JSON and Text editors switches wrap for the tab's
// body, as the response's Wrap chip does for the response; the other
// editors have no long text lines to wrap and show no chip.
func TestTheBodyWrapChipWrapsTheTabsBodyEditor(t *testing.T) {
	for _, state := range bodyPaneStates {
		t.Run(state.name, func(t *testing.T) {
			world := open(t, state.bodyType, state.text)
			root := world.view()
			Layout(world.ctx, root, 600, paneH)
			chip := chipLabelled(root, "Wrap")
			editable := state.bodyType == request.BodyJSON || state.bodyType == request.BodyText
			if (chip != nil) != editable {
				t.Fatalf("Wrap chip shown %v, want %v", chip != nil, editable)
			}
			if !editable {
				return
			}
			at := chip.Rect()
			world.ctx.In = core.Input{X: at.X + at.W/2, Y: at.Y + at.H/2, Click: true}
			Update(world.ctx, root)
			if !world.pane.Area(world.tab).Wrap {
				t.Fatal("the Wrap chip left the body unwrapped")
			}
			root = world.view()
			Layout(world.ctx, root, 600, paneH)
			world.ctx.In = core.Input{X: 5000, Y: 5000}
			Update(world.ctx, root)
			if !world.pane.Area(world.tab).Wrap {
				t.Fatal("the next frame's editor dropped the wrap the chip chose")
			}
		})
	}
}

// What Dump does not show of the text editors: JSON and Text edit the
// tab's body area, colored by a styler of their own.
func TestTheBodyAreaEditsTheTabsBody(t *testing.T) {
	for _, bodyType := range []request.BodyType{request.BodyJSON, request.BodyText} {
		t.Run(string(bodyType), func(t *testing.T) {
			world := open(t, bodyType, "{}")
			areas := behaviorStates(world.view(), "*widget.areaB")
			if len(areas) != 1 {
				t.Fatalf("%d areas, want 1: %q", len(areas), areas)
			}
			if want := fmt.Sprintf("%p", world.pane.Area(world.tab)); !strings.Contains(areas[0], want) {
				t.Errorf("the area lacks the tab's body %s: %s", want, areas[0])
			}
			if world.pane.Area(world.tab).Styler == nil {
				t.Error("the body area has no styler")
			}
		})
	}
}

func TestTheFormAndGraphQLBodiesComeFromTheHostsSlots(t *testing.T) {
	for _, test := range []struct {
		bodyType request.BodyType
		want     string
	}{
		{request.BodyForm, "form table"},
		{request.BodyMultipart, "form table"},
		{request.BodyGraphQL, "graphql body"},
	} {
		world := open(t, test.bodyType, "")
		if dump := Dump(world.view()); !strings.Contains(dump, `text="`+test.want+`"`) {
			t.Errorf("%s: the slot %q is missing:\n%s", test.bodyType, test.want, dump)
		}
	}
}

// fileBodyStates are the looks of the file body editor: no path yet, a
// path with nothing behind it and a path of a file.
var fileBodyStates = []struct {
	name string
	path func(t *testing.T) string
}{
	{"empty", func(t *testing.T) string { return "" }},
	{"missing", func(t *testing.T) string { return filepath.Join(t.TempDir(), "missing.bin") }},
	{"found", func(t *testing.T) string {
		path := filepath.Join(t.TempDir(), "found.bin")
		if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}},
}

// The file body editor is pinned as text (element kinds, options, rects)
// for no path, a missing one and a found one, at two widths. The fake
// asks the disk itself: a ui package may not import the file adapter.
func TestTheFileBodyLaysOutAsPinned(t *testing.T) {
	for _, state := range fileBodyStates {
		for _, size := range paneSizes {
			name := "filebody-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				world := newWorld(t, &request.Request{Name: "R", BodyType: request.BodyFile})
				world.fake.statFiles = true
				*world.pane.File(world.tab) = widget.NewTextField(state.path(t))
				world.pane.fileBody(world.ctx, world.tab) // asks the disk about the path
				assertDumpPinned(t, world.ctx, world.pane.fileBody(world.ctx, world.tab), name, size.width, paneH)
			})
		}
	}
}

// What Dump does not show of the path field: it edits the tab's own file
// field, with the placeholder and no Enter handler.
func TestTheFileBodyFieldEditsTheTabsPath(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R", BodyType: request.BodyFile})
	fields := behaviorStates(world.pane.fileBody(world.ctx, world.tab), "*widget.fieldB")
	if len(fields) != 1 {
		t.Fatalf("%d fields, want 1: %q", len(fields), fields)
	}
	for _, want := range []string{"field:" + fmt.Sprintf("%p", world.pane.File(world.tab)), "placeholder:Path of the file to send", "onEnter:<nil>"} {
		if !strings.Contains(fields[0], want) {
			t.Errorf("path field lacks %q: %s", want, fields[0])
		}
	}
}

func TestChoosingAFileSetsThePathAsTheBody(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R", BodyType: request.BodyFile})
	world.pane.choose(world.tab)
	if want := []string{"pick Choose the file to send"}; !reflect.DeepEqual(world.fake.asked, want) {
		t.Fatalf("asked %v, want %v", world.fake.asked, want)
	}
	world.fake.picks[0]("/data/picked.bin")
	if got := world.pane.File(world.tab).Text; got != "/data/picked.bin" || world.tab.Req.Body != got {
		t.Fatalf("field %q body %q, want the picked path in both", got, world.tab.Req.Body)
	}
}

func TestSchemaRowsShowEveryFieldWithItsNesting(t *testing.T) {
	address := &request.MessageSchema{Name: "Address", Fields: []request.SchemaField{{Name: "street", Type: "string"}}}
	node := &request.MessageSchema{Name: "Node", Cut: true}
	schema := request.MessageSchema{Name: "Person", Fields: []request.SchemaField{
		{Name: "name", Type: "string"},
		{Name: "tags", Type: "string", Repeated: true},
		{Name: "status", Type: "enum Status", EnumValues: []string{"UNKNOWN", "SERVING"}},
		{Name: "labels", Type: "map<string, string>", MapKey: "string"},
		{Name: "email", Type: "string", Oneof: "contact"},
		{Name: "home", Type: "message Address", Message: address},
		{Name: "next", Type: "message Node", Message: node},
	}}
	want := []schemaRow{
		{Depth: 0, Text: "name: string"},
		{Depth: 0, Text: "tags: repeated string"},
		{Depth: 0, Text: "status: enum Status (UNKNOWN, SERVING)"},
		{Depth: 0, Text: "labels: map<string, string>"},
		{Depth: 0, Text: "email: string, oneof contact"},
		{Depth: 0, Text: "home: message Address"},
		{Depth: 1, Text: "street: string"},
		{Depth: 0, Text: "next: message Node (recursive)"},
	}
	if got := schemaRows(schema); !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %+v\nwant   %+v", got, want)
	}
}

func TestTheSchemaViewAsksForAMethodFirst(t *testing.T) {
	world := newWorld(t, &request.Request{Name: "R", Protocol: "grpc"})
	dump := Dump(world.schemaView())
	if !strings.Contains(dump, "Pick a method") || strings.Contains(dump, "Fill body") {
		t.Errorf("want only the prompt to pick a method:\n%s", dump)
	}
	if len(world.fake.describes) != 0 {
		t.Error("asked for a message with no method")
	}
}

func TestTheSchemaViewShowsTheFieldsOfTheAnsweredMessage(t *testing.T) {
	world := grpcWorld(t)
	if dump := Dump(world.schemaView()); !strings.Contains(dump, "Loading...") {
		t.Errorf("no loading line while the answer is on its way:\n%s", dump)
	}
	world.fake.describes[0](healthSchema(), nil)
	dump := Dump(world.schemaView())
	for _, want := range []string{`text="HealthCheckRequest"`, `text="service: string"`, `text="Fill body"`} {
		if !strings.Contains(dump, want) {
			t.Errorf("the view lacks %s:\n%s", want, dump)
		}
	}
}

// Variables stand out in a JSON or Text body as they do in the URL: green
// when the request resolves them, red when not.
func TestBodyEditorColorsVariablesByWhetherTheyResolve(t *testing.T) {
	world := open(t, request.BodyJSON, "")
	world.fake.vars = []request.KV{{Name: "known", Value: "1", Enabled: true}}
	world.view()
	spans := world.pane.Area(world.tab).Styler(`{"a": "{{known}}", "b": "{{missing}}"}`)
	var known, missing core.Token
	var plain int
	for _, span := range spans {
		switch span.Text {
		case "{{known}}":
			known = span.Style
		case "{{missing}}":
			missing = span.Style
		default:
			plain++
		}
	}
	if known != core.OK || missing != core.Error {
		t.Fatalf("known %q missing %q in %+v", known, missing, spans)
	}
	if plain == 0 {
		t.Fatal("the text around the variables must stay")
	}
	joined := ""
	for _, span := range spans {
		joined += span.Text
	}
	if joined != `{"a": "{{known}}", "b": "{{missing}}"}` {
		t.Fatalf("styling must not change the text: %q", joined)
	}
	world.pane.ChooseType(world.tab, request.BodyText)
	world.view()
	if got := world.pane.Area(world.tab).Styler("no variables here"); len(got) != 1 {
		t.Fatalf("a line without variables is one span: %+v", got)
	}
}
