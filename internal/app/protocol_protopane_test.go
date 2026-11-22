package app

import (
	"context"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/exampleserver"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/editor"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// healthProtoSource is the example server's copy of the real health.proto.
var healthProtoSource = filepath.Join("..", "exampleserver", "protos", "grpc", "health", "v1", "health.proto")

// grpcTabOnProtoPane is a test app whose open tab is a clean gRPC request
// with its Proto pane shown, in a collection kept in dir.
func grpcTabOnProtoPane(t *testing.T) (a *App, tab *reqTab, dir string) {
	t.Helper()
	a = testApp(t)
	t.Cleanup(a.WaitWrites)
	dir = t.TempDir()
	a.setWorkspace(dir)
	tab = a.cur()
	editRequest(tab, editor.ProtocolChosen{Protocol: "grpc"})
	tab.MarkSaved()
	clickText(t, a, "Proto", 0)
	return a, tab, dir
}

// protoFileIn copies health.proto into dir/protos, or writes source there
// under name when source is given, and returns its absolute path.
func protoFileIn(t *testing.T, dir, name string, source []byte) string {
	t.Helper()
	if source == nil {
		var err error
		if source, err = os.ReadFile(healthProtoSource); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(dir, "protos", name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, source, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestOnlyAGRPCRequestHasAProtoPane(t *testing.T) {
	for protocol, want := range map[request.Protocol]bool{"grpc": true, "rest": false, "websocket": false, "sse": false, "mcp": false} {
		got := false
		for _, candidate := range requestPanesFor(newReqTab(tabs.New(&request.Request{Protocol: protocol}))) {
			got = got || candidate.label == "Proto"
		}
		if got != want {
			t.Errorf("%s has a Proto pane: %v, want %v", protocol, got, want)
		}
	}
}

func TestTheProtoPaneStartsOnTheServersReflection(t *testing.T) {
	a, _, _ := grpcTabOnProtoPane(t)
	dump := layoutDump(a)
	for _, want := range []string{`text="Reflection"`, `text="Proto files"`, `text="Methods come from the server's reflection"`} {
		if !strings.Contains(dump, want) {
			t.Errorf("the pane lacks %s", want)
		}
	}
	if strings.Contains(dump, `text="Add file..."`) {
		t.Error("the file lists show while the source is reflection")
	}
}

// Choosing Proto files shows the lists; with no file in them the request
// still uses reflection, so it is not edited.
func TestChoosingProtoFilesShowsTheListsAndLeavesTheRequestAsItWas(t *testing.T) {
	a, tab, _ := grpcTabOnProtoPane(t)
	clickText(t, a, "Proto files", 0)
	dump := layoutDump(a)
	// captions are drawn in capitals, as the Query caption of Params
	for _, want := range []string{`text="FILES"`, `text="IMPORT PATHS"`, `text="Add file..."`, `text="Add path"`} {
		if !strings.Contains(dump, want) {
			t.Errorf("the pane lacks %s", want)
		}
	}
	if strings.Contains(dump, "Methods come from the server's reflection") {
		t.Error("the reflection note still shows")
	}
	if tab.Dirty() {
		t.Errorf("choosing the source alone edited the request: %+v", tab.Req.Proto)
	}
}

// Add file... asks for a file; the chosen one is stored absolute and shown
// relative to the collection, as it is saved.
func TestAddFileListsTheChosenFileRelativeToTheCollection(t *testing.T) {
	a, tab, dir := grpcTabOnProtoPane(t)
	health := protoFileIn(t, dir, "health.proto", nil)
	asked := ""
	a.files = fakeDialog{choose: func(_ context.Context, title string, _ filedialog.Mode, _ string) (string, error) {
		asked = title
		return health, nil
	}}
	clickText(t, a, "Proto files", 0)
	clickText(t, a, "Add file...", 0)
	until(t, a, "the chosen file", func() bool { return len(tab.Req.Proto.Files) > 0 })
	if asked == "" {
		t.Error("no file dialog was asked")
	}
	if !reflect.DeepEqual(tab.Req.Proto.Files, []string{health}) || !tab.Dirty() {
		t.Fatalf("files %v dirty %v, want [%s] and an edited tab", tab.Req.Proto.Files, tab.Dirty(), health)
	}
	if input := tab.protoInput(editor.ProtoFiles, 0); input.Text != "protos/health.proto" {
		t.Errorf("the row shows %q, want the path relative to the collection", input.Text)
	}
}

// A path typed into a row is read from the collection's folder when it is
// relative, so the request holds an absolute path.
func TestTypingAPathSetsItFromTheCollectionFolder(t *testing.T) {
	a, tab, dir := grpcTabOnProtoPane(t)
	clickText(t, a, "Proto files", 0)
	clickText(t, a, "Add path", 0)
	if !reflect.DeepEqual(tab.Req.Proto.ImportPaths, []string{""}) {
		t.Fatalf("import paths %q, want one empty row to type into", tab.Req.Proto.ImportPaths)
	}
	// the new row is the last field of the window, after the URL bar's
	clickDumped(t, a, "fieldB", strings.Count(layoutDump(a), "fieldB")-1)
	a.Step(core.Input{X: 5000, Y: 5000, Typed: []rune("protos")})
	if want := []string{filepath.Join(dir, "protos")}; !reflect.DeepEqual(tab.Req.Proto.ImportPaths, want) {
		t.Fatalf("import paths %q, want %q", tab.Req.Proto.ImportPaths, want)
	}
}

func TestRemoveDropsThatRowOnly(t *testing.T) {
	a, tab, dir := grpcTabOnProtoPane(t)
	first, second := protoFileIn(t, dir, "a.proto", nil), protoFileIn(t, dir, "b.proto", nil)
	editRequest(tab, editor.ProtoPathAdded{List: editor.ProtoFiles, Path: first})
	editRequest(tab, editor.ProtoPathAdded{List: editor.ProtoFiles, Path: second})
	clickDumped(t, a, "tip=Remove", 0)
	if !reflect.DeepEqual(tab.Req.Proto.Files, []string{second}) {
		t.Fatalf("files %q, want only %s", tab.Req.Proto.Files, second)
	}
	if input := tab.protoInput(editor.ProtoFiles, 0); input.Text != "protos/b.proto" {
		t.Errorf("the remaining row shows %q", input.Text)
	}
}

// Choosing Reflection drops the files and import paths: the request is
// edited and saved without them.
func TestChoosingReflectionDropsTheFilesAndImportPaths(t *testing.T) {
	a, tab, dir := grpcTabOnProtoPane(t)
	editRequest(tab, editor.ProtoPathAdded{List: editor.ProtoFiles, Path: protoFileIn(t, dir, "health.proto", nil)})
	editRequest(tab, editor.ProtoPathAdded{List: editor.ImportPaths, Path: filepath.Join(dir, "protos")})
	tab.MarkSaved()
	clickText(t, a, "Reflection", 0)
	if !reflect.DeepEqual(tab.Req.Proto, request.ProtoSource{}) || !tab.Dirty() {
		t.Fatalf("proto %+v dirty %v, want none and an edited tab", tab.Req.Proto, tab.Dirty())
	}
	if !strings.Contains(layoutDump(a), "Methods come from the server's reflection") {
		t.Error("the reflection note does not show")
	}
}

// The files are read in the background after an edit; a problem in them
// is shown under the lists at its file, line and column.
func TestAParseErrorShowsWithFileLineAndColumn(t *testing.T) {
	a, tab, dir := grpcTabOnProtoPane(t)
	source, err := os.ReadFile(healthProtoSource)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.SplitAfter(string(source), "\n")
	broken := protoFileIn(t, dir, "broken.proto", []byte(strings.Join(lines[:33], "")))
	editRequest(tab, editor.ProtoPathAdded{List: editor.ProtoFiles, Path: broken})
	want := `text="protos/broken.proto:34:`
	until(t, a, "the parse error", func() bool { return strings.Contains(layoutDump(a), want) })
	for _, line := range strings.Split(layoutDump(a), "\n") {
		if strings.Contains(line, want) && !strings.Contains(line, "fg=error") {
			t.Errorf("the problem is not in the error color: %s", line)
		}
	}
}

// exampleServerURLs serves the example server on free ports for the test
// and returns each service's URL by name.
func exampleServerURLs(t *testing.T) map[string]string {
	t.Helper()
	endpoints, err := exampleserver.Listen(exampleserver.Config{Host: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, stop := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- exampleserver.Serve(ctx, endpoints) }()
	t.Cleanup(func() {
		stop()
		<-served
	})
	urls := map[string]string{}
	for _, endpoint := range endpoints {
		urls[endpoint.Name] = endpoint.URL
	}
	return urls
}

// Task 070's acceptance in the app: the example server's port without
// reflection cannot be called until the Proto pane adds its .proto file,
// and then the call answers.
func TestAServerWithoutReflectionIsCalledFromTheProtoFileAddedInThePane(t *testing.T) {
	a, tab, dir := grpcTabOnProtoPane(t)
	urls := exampleServerURLs(t)
	pointTabAt(tab, urls["grpc-no-reflection"])
	editRequest(tab, editor.MethodChosen{Method: "grpc.health.v1.Health/Check"})
	editRequest(tab, editor.BodyTyped{Text: `{"service": "example"}`})

	tab.Resp = nil
	a.send(tab)
	waitForResponse(t, a, tab)
	if tab.Resp.Err == "" {
		t.Fatalf("the server answered without a proto file: %+v", tab.Resp)
	}

	health := protoFileIn(t, dir, "health.proto", nil)
	a.files = fakeDialog{choose: func(context.Context, string, filedialog.Mode, string) (string, error) { return health, nil }}
	clickText(t, a, "Proto files", 0)
	clickText(t, a, "Add file...", 0)
	until(t, a, "the chosen file", func() bool { return len(tab.Req.Proto.Files) > 0 })
	tab.Resp = nil
	a.send(tab)
	waitForResponse(t, a, tab)
	if tab.Resp.Err != "" || tab.Resp.Status != 200 || !strings.Contains(string(tab.Resp.Body), "SERVING") {
		t.Fatalf("from the proto file: %+v", tab.Resp)
	}
}

// A file that does not parse at send is that call's error, at its line,
// never a fallback to reflection.
func TestSendingWithABrokenProtoFileShowsWhereItBreaks(t *testing.T) {
	a, tab, dir := grpcTabOnProtoPane(t)
	urls := exampleServerURLs(t)
	source, err := os.ReadFile(healthProtoSource)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.SplitAfter(string(source), "\n")
	broken := protoFileIn(t, dir, "broken.proto", []byte(strings.Join(lines[:33], "")))
	pointTabAt(tab, urls["grpc"]) // the port with reflection: the files must still win
	editRequest(tab, editor.MethodChosen{Method: "grpc.health.v1.Health/Check"})
	editRequest(tab, editor.ProtoPathAdded{List: editor.ProtoFiles, Path: broken})
	tab.Resp = nil
	a.send(tab)
	waitForResponse(t, a, tab)
	if !strings.Contains(tab.Resp.Err, broken+":34:") {
		t.Fatalf("Err = %q, want it to point at %s:34", tab.Resp.Err, broken)
	}
}

// Picking a method of a request with proto files reads the files: no
// server has to be entered or reachable.
func TestPickingAMethodFromProtoFilesNeedsNoServer(t *testing.T) {
	a, tab, dir := grpcTabOnProtoPane(t)
	tab.Req.URL = ""
	editRequest(tab, editor.ProtoPathAdded{List: editor.ProtoFiles, Path: protoFileIn(t, dir, "health.proto", nil)})
	a.pickGRPCMethod(tab)
	until(t, a, "the menu of methods", func() bool { return widget.PopupMenu(a.ctx) != nil })
	if got := strings.Join(menuLabels(a), " "); !strings.Contains(got, "grpc.health.v1.Health/Check") {
		t.Fatalf("methods = %s", got)
	}
}
