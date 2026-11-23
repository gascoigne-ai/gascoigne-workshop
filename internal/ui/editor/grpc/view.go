package grpc

import (
	"reflect"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// chip is what the pane's template shows of one source: Reflection or
// Proto files.
type chip struct {
	Label  string
	Width  int
	Chosen bool
	Pick   func()
}

// listView is one list of paths, files or import paths.
type listView struct {
	Caption  string
	Rows     []rowView
	AddLabel string
	AddWidth int
	Add      func()
}

type rowView struct {
	Input  *widget.TextField
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

// View is the Proto tab of a gRPC request: where its methods come from, the
// server's reflection or .proto files (the user's choice of a tab of its
// own). Files are picked with the file dialog; import paths are typed, since
// the dialogs pick files only and a folder mode would touch every dialog
// program. Drawing the pane while files show starts the read of them, so a
// problem shows while the user edits; there is no other frame work.
func (c *Component) View(ctx *core.Ctx, tab *tabs.Tab, scroll *widget.ScrollState) Node {
	st := c.state(tab)
	st.dir = c.deps.Vars.Folder(tab.Req)
	shown := st.filesShown(tab.Req)
	if shown {
		c.startCheck(st, tab)
	}
	chooseReflection := func() {
		st.filesChosen = false
		if !reflect.DeepEqual(tab.Req.Proto, request.ProtoSource{}) {
			editRequest(tab, requestedit.ReflectionChosen{})
		}
	}
	chips := []chip{
		{Label: "Reflection", Width: reflectionChipW, Chosen: !shown, Pick: chooseReflection},
		{Label: "Proto files", Width: protoFilesChipW, Chosen: shown, Pick: func() { st.filesChosen = true }},
	}
	lists := []listView{
		c.list(st, tab, requestedit.ProtoFiles, "Files", "Add file...", addFileButtonW, func() { c.addFile(st, tab) }),
		c.list(st, tab, requestedit.ImportPaths, "Import paths", "Add path", addPathButtonW, func() {
			editRequest(tab, requestedit.ProtoPathAdded{List: requestedit.ImportPaths})
		}),
	}
	return paneTemplate.MustBuild(sizes.Scope(ctx, markup.Scope{
		"scroll":     scroll,
		"sources":    chips,
		"files":      shown,
		"reflection": !shown,
		"lists":      lists,
		"problems":   st.problems(tab.Req.Proto),
	}))
}

func (c *Component) list(st *state, tab *tabs.Tab, list requestedit.ProtoList, caption, addLabel string, addWidth int, add func()) listView {
	paths := protoPaths(tab.Req.Proto, list)
	rows := make([]rowView, 0, len(paths))
	for index := range paths {
		field := input(st, tab.Req.Proto, list, index)
		sync := func() {
			editRequest(tab, requestedit.ProtoPathSet{List: list, Index: index, Path: pathFromText(st.dir, field.Text)})
		}
		remove := func() { editRequest(tab, requestedit.ProtoPathRemoved{List: list, Index: index}) }
		rows = append(rows, rowView{Input: field, Sync: sync, Remove: remove})
	}
	return listView{Caption: caption, Rows: rows, AddLabel: addLabel, AddWidth: addWidth, Add: add}
}

var paneTemplate = markup.Must("protopane.vml", paneSource)

// Templates, parsed once above.
const (
	paneSource = `<scroll state="scroll" pad="{{gutter}}" gap="{{looseGap}}">
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
