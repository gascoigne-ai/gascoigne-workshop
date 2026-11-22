package app

import (
	"context"
	"errors"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/desktop/filedialog"
	"reflect"
	"strings"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/editor"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// protoPane is the Proto tab of a gRPC request: where its methods come
// from, the server's reflection or .proto files (the user's choice of a tab
// of its own). Files are picked with the file dialog; import paths are
// typed, since the dialogs pick files only and a folder mode would touch
// every dialog program.
var protoPane = pane{label: "Proto", kind: paneProto}

// grpcRequestPanes are the request side of gRPC: the HTTP-like panes with
// Proto after the Body, which holds the message the files describe.
var grpcRequestPanes = []pane{paramsPane, bodyPane, protoPane, schemaPane, headersPane, authPane, varsPane, testsPane, settingsPane}

// protoState is what a tab keeps for its Proto pane between frames.
type protoState struct {
	filesChosen bool                       // the file lists show before a file is added
	inputs      map[protoRowKey]*TextField // path fields, made on first use
	dir         string                     // the folder of the tab's collection, as the pane last saw it
	check       protoCheck                 // what reading the proto files found
}

// protoRowKey names one path row of the Proto pane.
type protoRowKey struct {
	list  editor.ProtoList
	index int
}

// protoCheck is what reading a request's proto files in the background
// found, and for which source.
type protoCheck struct {
	source   request.ProtoSource
	problems []string
	running  bool
}

// protoSourceChip and protoListView are what the pane's template shows.
type protoSourceChip struct {
	Label  string
	Width  int
	Chosen bool
	Pick   func()
}

type protoListView struct {
	Caption  string
	Rows     []protoRowView
	AddLabel string
	AddWidth int
	Add      func()
}

type protoRowView struct {
	Input  *TextField
	Sync   func()
	Remove func()
}

// Widths of the pane's chips and buttons, sized to their labels.
const (
	reflectionChipW = 92
	protoFilesChipW = 96
	addFileButtonW  = 96
	addPathButtonW  = 84
)

// protoFilesShown says the pane shows the file lists: the request has
// paths, or the user chose Proto files and has not added one yet (which
// only the tab remembers: a request without files uses reflection).
func protoFilesShown(tab *reqTab) bool {
	return tab.proto.filesChosen || len(tab.Req.Proto.Files) > 0 || len(tab.Req.Proto.ImportPaths) > 0
}

// protoInput is the text field of a path row, made on first use with the
// path as the collection saves it: relative to its folder when inside.
func (t *reqTab) protoInput(list editor.ProtoList, index int) *TextField {
	path := protoPaths(t.Req.Proto, list)[index]
	key := protoRowKey{list, index}
	if input, ok := t.proto.inputs[key]; ok && pathFromText(t.proto.dir, input.Text) == path {
		return input
	}
	// A path changed by something other than typing (a removed row before
	// it, a renamed collection) gets a fresh field showing it.
	input := new(TextField)
	*input = NewTextField(request.StoredPath(t.proto.dir, path))
	if t.proto.inputs == nil {
		t.proto.inputs = map[protoRowKey]*TextField{}
	}
	t.proto.inputs[key] = input
	return input
}

// pathFromText is the path a row's text stands for: a relative one is
// taken from the collection's folder, and an empty row is no path yet.
func pathFromText(dir, text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	return request.LoadedPath(dir, text)
}

func protoPaths(source request.ProtoSource, list editor.ProtoList) []string {
	if list == editor.ImportPaths {
		return source.ImportPaths
	}
	return source.Files
}

// protoPane is the Proto tab: the source chips, then the reflection note
// or the lists of files and import paths with what reading them found.
func (a *App) protoPane(_ pane, tab *reqTab, scroll *ScrollState) Node {
	entry, _ := a.entryFor(tab.Req)
	tab.proto.dir = entry.Dir
	shown := protoFilesShown(tab)
	if shown {
		a.checkProtoFiles(tab)
	}
	chooseReflection := func() {
		tab.proto.filesChosen = false
		if !reflect.DeepEqual(tab.Req.Proto, request.ProtoSource{}) {
			editRequest(tab, editor.ReflectionChosen{})
		}
	}
	sources := []protoSourceChip{
		{Label: "Reflection", Width: reflectionChipW, Chosen: !shown, Pick: chooseReflection},
		{Label: "Proto files", Width: protoFilesChipW, Chosen: shown, Pick: func() { tab.proto.filesChosen = true }},
	}
	lists := []protoListView{
		a.protoList(tab, editor.ProtoFiles, "Files", "Add file...", addFileButtonW, func() { a.addProtoFile(tab) }),
		a.protoList(tab, editor.ImportPaths, "Import paths", "Add path", addPathButtonW, func() {
			editRequest(tab, editor.ProtoPathAdded{List: editor.ImportPaths})
		}),
	}
	return protoPaneTemplate.MustBuild(a.scope(markup.Scope{
		"scroll":     scroll,
		"sources":    sources,
		"files":      shown,
		"reflection": !shown,
		"lists":      lists,
		"problems":   tab.proto.check.problems,
	}))
}

func (a *App) protoList(tab *reqTab, list editor.ProtoList, caption, addLabel string, addWidth int, add func()) protoListView {
	paths := protoPaths(tab.Req.Proto, list)
	rows := make([]protoRowView, 0, len(paths))
	for index := range paths {
		input := tab.protoInput(list, index)
		sync := func() {
			editRequest(tab, editor.ProtoPathSet{List: list, Index: index, Path: pathFromText(tab.proto.dir, input.Text)})
		}
		remove := func() { editRequest(tab, editor.ProtoPathRemoved{List: list, Index: index}) }
		rows = append(rows, protoRowView{Input: input, Sync: sync, Remove: remove})
	}
	return protoListView{Caption: caption, Rows: rows, AddLabel: addLabel, AddWidth: addWidth, Add: add}
}

// addProtoFile asks for a .proto file and adds it to the request.
func (a *App) addProtoFile(tab *reqTab) {
	a.pickFile("Choose a .proto file", filedialog.OpenFile, "", func(path string) {
		tab.proto.filesChosen = true
		editRequest(tab, editor.ProtoPathAdded{List: editor.ProtoFiles, Path: path})
	})
}

// checkProtoFiles reads the request's proto files in the background when
// they changed since the last read, so a problem shows while the user edits
// and not only at send. Answers for an older source are dropped by the
// comparison: the next build starts a read of the current one.
func (a *App) checkProtoFiles(tab *reqTab) {
	source := tab.Req.Proto
	if len(source.Files) == 0 {
		tab.proto.check = protoCheck{source: source}
		return
	}
	if tab.proto.check.running || reflect.DeepEqual(tab.proto.check.source, source) {
		return
	}
	tab.proto.check.running = true
	req := tab.Req.Clone()
	dir := tab.proto.dir
	timeout := a.timeoutFor(req) // read here: the goroutine must not touch App fields
	a.bus.Run(context.Background(), func(ctx context.Context) bus.Msg {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		_, err := a.listGRPCMethods(ctx, req)
		return func() { // runs on the UI goroutine
			tab.proto.check = protoCheck{source: req.Proto, problems: protoProblems(err, dir)}
		}
	})
}

// protoReading reports whether t shows its Proto pane while the files are
// still being read, so a capture waits for what the read finds.
func protoReading(tab *reqTab) bool {
	panes := requestPanesFor(tab)
	if tab.paneTab < 0 || tab.paneTab >= len(panes) || panes[tab.paneTab].kind != paneProto || len(tab.Req.Proto.Files) == 0 {
		return false
	}
	return tab.proto.check.running || !reflect.DeepEqual(tab.proto.check.source, tab.Req.Proto)
}

// protoProblems are the lines that tell what is wrong with the files, each
// located problem at its file (as the collection names it), line and
// column.
func protoProblems(err error, dir string) []string {
	if err == nil {
		return nil
	}
	parts := []error{err}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		parts = joined.Unwrap()
	}
	problems := make([]string, 0, len(parts))
	for _, part := range parts {
		var located request.ProtoError
		if !errors.As(part, &located) {
			problems = append(problems, part.Error())
			continue
		}
		located.Path = request.StoredPath(dir, located.Path)
		problems = append(problems, located.Error())
	}
	return problems
}

var protoPaneTemplate = markup.Must("protopane.vml", protopaneSource)

// Templates, parsed once above.
const (
	protopaneSource = `<scroll state="scroll" pad="{{gutter}}" gap="{{looseGap}}">
  <row h="{{rowH}}" gap="{{tightGap}}">
    <segment-chip for="chip in sources" selected="{{chip.Chosen}}" w="{{chip.Width}}" on-click="chip.Pick">{{chip.Label}}</segment-chip>
  </row>
  <lines if="reflection" color="muted">Methods come from the server's reflection</lines>
  <col if="files" gap="{{looseGap}}">
    <col for="list in lists" gap="{{tightGap}}">
      <row h="{{rowH}}">
        <caption>{{list.Caption}}</caption>
      </row>
      <row for="row in list.Rows" h="{{rowH}}" gap="{{tightGap}}" align-items="center">
        <field state="row.Input" grow="1" h="{{rowH}}" bg="sunken" on-change="row.Sync" text-menu="row.Input"/>
        <icon-button icon="trash" w="{{iconBtnW}}" h="{{compactH}}" on-click="row.Remove" tooltip="Remove"/>
      </row>
      <row h="{{rowH}}">
        <button w="{{list.AddWidth}}" on-click="list.Add">{{list.AddLabel}}</button>
      </row>
    </col>
    <text for="problem in problems" small color="error">{{problem}}</text>
  </col>
</scroll>
`
)
