package grpc

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
)

// paneStates are the looks of the Proto pane: the reflection note, and the
// file lists with two files, an import path and a problem the files were
// found to have.
var paneStates = []struct {
	name  string
	setUp func(world *world)
}{
	{"reflection", func(world *world) {}},
	{"files", func(world *world) {
		for _, name := range []string{"a.proto", "b.proto"} {
			editRequest(world.tab, requestedit.ProtoPathAdded{List: requestedit.ProtoFiles, Path: world.protoPath(name)})
		}
		editRequest(world.tab, requestedit.ProtoPathAdded{List: requestedit.ImportPaths, Path: filepath.Join(world.dir, "include")})
		world.view() // asks for the files to be read
		world.answer(nil, request.ProtoError{Path: world.protoPath("b.proto"), Line: 4, Col: 9, Message: "unexpected token"})
	}},
}

// The Proto pane is pinned as text (element kinds, options, rects) at two
// widths; the goldens are the ones the pane had in the app before it moved
// here.
func TestTheProtoPaneLaysOutAsPinned(t *testing.T) {
	sizes := []struct {
		name  string
		width int
	}{
		{"wide", wideW},
		{"narrow", narrowW},
	}
	for _, state := range paneStates {
		for _, size := range sizes {
			name := "protopane-" + state.name + "-" + size.name
			t.Run(name, func(t *testing.T) {
				world := newWorld(t)
				state.setUp(world)
				assertDumpPinned(t, world.ctx, world.view(), name, size.width, paneH)
			})
		}
	}
}

func TestTheProtoPaneStartsOnTheServersReflection(t *testing.T) {
	world := newWorld(t)
	dump := world.text()
	for _, want := range []string{`text="Reflection"`, `text="Proto files"`, `text="Methods come from the server's reflection"`} {
		if !strings.Contains(dump, want) {
			t.Errorf("the pane lacks %s", want)
		}
	}
	if strings.Contains(dump, `text="Add file..."`) {
		t.Error("the file lists show while the source is reflection")
	}
	if len(world.fake.listed) != 0 {
		t.Errorf("the files were read with none chosen: %v", world.fake.listed)
	}
}

// Choosing Proto files shows the lists; with no file in them the request
// still uses reflection, so it is not edited.
func TestChoosingProtoFilesShowsTheListsAndLeavesTheRequestAsItWas(t *testing.T) {
	world := newWorld(t)
	world.click("Proto files")
	dump := world.text()
	// captions are drawn in capitals, as the Query caption of Params
	for _, want := range []string{`text="FILES"`, `text="IMPORT PATHS"`, `text="Add file..."`, `text="Add path"`} {
		if !strings.Contains(dump, want) {
			t.Errorf("the pane lacks %s", want)
		}
	}
	if strings.Contains(dump, "Methods come from the server's reflection") {
		t.Error("the reflection note still shows")
	}
	if world.tab.Dirty() {
		t.Errorf("choosing the source alone edited the request: %+v", world.tab.Req.Proto)
	}
}

// Add file... asks for a file; the chosen one is stored absolute and shown
// relative to the collection, as it is saved.
func TestAddFileListsTheChosenFileRelativeToTheCollection(t *testing.T) {
	world := newWorld(t)
	health := world.protoPath("health.proto")
	world.click("Proto files")
	world.click("Add file...")
	if len(world.fake.titles) != 1 {
		t.Fatalf("file dialogs asked: %v, want one", world.fake.titles)
	}
	world.fake.chosen[0](health)
	if !reflect.DeepEqual(world.tab.Req.Proto.Files, []string{health}) || !world.tab.Dirty() {
		t.Fatalf("files %v dirty %v, want [%s] and an edited tab", world.tab.Req.Proto.Files, world.tab.Dirty(), health)
	}
	world.view()
	if input := world.pane.Input(world.tab, requestedit.ProtoFiles, 0); input.Text != "protos/health.proto" {
		t.Errorf("the row shows %q, want the path relative to the collection", input.Text)
	}
}

// A path typed into a row is read from the collection's folder when it is
// relative.
func TestTypingAPathSetsItFromTheCollectionFolder(t *testing.T) {
	world := newWorld(t)
	world.click("Proto files")
	world.click("Add path")
	if !reflect.DeepEqual(world.tab.Req.Proto.ImportPaths, []string{""}) {
		t.Fatalf("import paths %q, want one empty row to type into", world.tab.Req.Proto.ImportPaths)
	}
	world.typeInto(0, "protos")
	if want := []string{filepath.Join(world.dir, "protos")}; !reflect.DeepEqual(world.tab.Req.Proto.ImportPaths, want) {
		t.Fatalf("import paths %q, want %q", world.tab.Req.Proto.ImportPaths, want)
	}
}

func TestRemoveDropsThatRowOnly(t *testing.T) {
	world := newWorld(t)
	first, second := world.protoPath("a.proto"), world.protoPath("b.proto")
	editRequest(world.tab, requestedit.ProtoPathAdded{List: requestedit.ProtoFiles, Path: first})
	editRequest(world.tab, requestedit.ProtoPathAdded{List: requestedit.ProtoFiles, Path: second})
	world.clickOn(removeButton(0))
	if !reflect.DeepEqual(world.tab.Req.Proto.Files, []string{second}) {
		t.Fatalf("files %q, want only %s", world.tab.Req.Proto.Files, second)
	}
	world.view()
	if input := world.pane.Input(world.tab, requestedit.ProtoFiles, 0); input.Text != "protos/b.proto" {
		t.Errorf("the remaining row shows %q", input.Text)
	}
}

// Choosing Reflection drops the files and import paths: the request is
// edited and saved without them.
func TestChoosingReflectionDropsTheFilesAndImportPaths(t *testing.T) {
	world := newWorld(t)
	editRequest(world.tab, requestedit.ProtoPathAdded{List: requestedit.ProtoFiles, Path: world.protoPath("health.proto")})
	editRequest(world.tab, requestedit.ProtoPathAdded{List: requestedit.ImportPaths, Path: filepath.Join(world.dir, "protos")})
	world.tab.MarkSaved()
	world.click("Reflection")
	if !reflect.DeepEqual(world.tab.Req.Proto, request.ProtoSource{}) || !world.tab.Dirty() {
		t.Fatalf("proto %+v dirty %v, want none and an edited tab", world.tab.Req.Proto, world.tab.Dirty())
	}
	if !strings.Contains(world.text(), "Methods come from the server's reflection") {
		t.Error("the reflection note does not show")
	}
}

// The files are read once for a source, not again every frame.
func TestTheFilesAreReadOnceForASource(t *testing.T) {
	world := newWorld(t)
	editRequest(world.tab, requestedit.ProtoPathAdded{List: requestedit.ProtoFiles, Path: world.protoPath("a.proto")})
	world.view()
	world.view()
	if len(world.fake.listed) != 1 {
		t.Fatalf("listings asked: %d, want 1 while the first is not answered", len(world.fake.listed))
	}
	world.answer(nil, nil)
	world.view()
	if len(world.fake.listed) != 1 {
		t.Fatalf("listings asked: %d, want 1 once the source is read", len(world.fake.listed))
	}
	editRequest(world.tab, requestedit.ProtoPathAdded{List: requestedit.ProtoFiles, Path: world.protoPath("b.proto")})
	world.view()
	if len(world.fake.listed) != 2 {
		t.Fatalf("listings asked: %d, want a second for the edited source", len(world.fake.listed))
	}
}

// The check reads the files, not the server: the request goes as it is
// written, with its {{variables}}.
func TestTheCheckAsksForTheRequestUnexpanded(t *testing.T) {
	world := newWorld(t)
	world.fake.vars = []request.KV{{Name: "host", Value: "example.test:1", Enabled: true}}
	world.tab.Req.URL = "grpc://{{host}}"
	editRequest(world.tab, requestedit.ProtoPathAdded{List: requestedit.ProtoFiles, Path: world.protoPath("a.proto")})
	world.view()
	if len(world.fake.listed) != 1 || world.fake.listed[0].URL != "grpc://{{host}}" {
		t.Fatalf("listed %+v, want the URL as written", world.fake.listed)
	}
}

// A problem shows under the lists, its file relative to the collection;
// an error that is not a located one shows as it is.
func TestProblemsNameTheFileRelativeToTheCollection(t *testing.T) {
	cases := []struct {
		name string
		err  func(world *world) error
		want []string
	}{
		{"located", func(world *world) error {
			return request.ProtoError{Path: world.protoPath("a.proto"), Line: 3, Col: 1, Message: "boom"}
		}, []string{"protos/a.proto:3:1: boom"}},
		{"plain", func(*world) error { return errors.New("no gRPC client") }, []string{"no gRPC client"}},
		{"several", func(world *world) error {
			return errors.Join(
				request.ProtoError{Path: world.protoPath("a.proto"), Line: 3, Col: 1, Message: "boom"},
				request.ProtoError{Path: world.protoPath("b.proto"), Line: 5, Col: 2, Message: "bang"},
			)
		}, []string{"protos/a.proto:3:1: boom", "protos/b.proto:5:2: bang"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			world := newWorld(t)
			editRequest(world.tab, requestedit.ProtoPathAdded{List: requestedit.ProtoFiles, Path: world.protoPath("a.proto")})
			world.view()
			world.answer(nil, test.err(world))
			dump := world.text()
			for _, want := range test.want {
				if !strings.Contains(dump, `text="`+want+`"`) {
					t.Errorf("the pane lacks the problem %q:\n%s", want, dump)
				}
			}
			for _, line := range strings.Split(dump, "\n") {
				if strings.Contains(line, test.want[0]) && !strings.Contains(line, "fg=error") {
					t.Errorf("the problem is not in the error color: %s", line)
				}
			}
		})
	}
}

// What the files of an older source were found to have says nothing of the
// source now: the problem is not shown, and the new source is read.
func TestAnAnswerForAnOlderSourceIsNotShown(t *testing.T) {
	world := newWorld(t)
	editRequest(world.tab, requestedit.ProtoPathAdded{List: requestedit.ProtoFiles, Path: world.protoPath("a.proto")})
	world.view()
	editRequest(world.tab, requestedit.ProtoPathAdded{List: requestedit.ProtoFiles, Path: world.protoPath("b.proto")})
	world.answer(nil, request.ProtoError{Path: world.protoPath("a.proto"), Line: 1, Col: 1, Message: "old"})
	if dump := world.text(); strings.Contains(dump, "old") {
		t.Errorf("the problem of the older source shows:\n%s", dump)
	}
	if len(world.fake.listed) != 2 || !reflect.DeepEqual(world.fake.listed[1].Proto.Files, world.tab.Req.Proto.Files) {
		t.Fatalf("listed %d times, want a second read of the current files", len(world.fake.listed))
	}
}

// A listing that ends after its tab was closed has nowhere to be shown: it
// must not make a state that nothing prunes again.
func TestAnAnswerAfterTheTabClosedMakesNoState(t *testing.T) {
	world := newWorld(t)
	editRequest(world.tab, requestedit.ProtoPathAdded{List: requestedit.ProtoFiles, Path: world.protoPath("a.proto")})
	world.view()
	world.fake.set.Close(0)
	world.pane.Tick()
	world.answer(nil, errors.New("late"))
	if len(world.pane.states) != 0 {
		t.Fatalf("states %d, want none for a closed tab", len(world.pane.states))
	}
}
