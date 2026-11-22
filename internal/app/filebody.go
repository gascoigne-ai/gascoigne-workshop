package app

import (
	"context"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/port"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/controller/editor"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/bus"

	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

var fileBodyTemplate = markup.Must("filebody.vml", filebodySource)

// bodyFileState is what is known of the file a file body names.
type bodyFileState int

const (
	bodyFileUnknown bodyFileState = iota // not asked yet, or the answer is on its way
	bodyFileFound                        // a file, not a folder
	bodyFileMissing                      // nothing there, a folder, or a path that cannot be looked at
)

// bodyFileLooks is how each state is shown under the path; no color is the
// muted text.
var bodyFileLooks = map[bodyFileState]struct {
	text  string
	color w.Token
}{
	bodyFileUnknown: {"", ""},
	bodyFileFound:   {"found", ""},
	bodyFileMissing: {"not found", w.Error},
}

// bodyFileCheck is the last question about a file body's path. The editor
// is built every frame, so the file system is asked once per path, off the
// UI goroutine, and the editor shows the answer it has.
type bodyFileCheck struct {
	path  string // the path asked about
	state bodyFileState
}

// checkBodyFile asks the file system about the tab's path when it is not
// the one last asked about; an answer for an older path is dropped.
func (a *App) checkBodyFile(tab *reqTab) {
	path := tab.file.Text
	if path == "" || path == tab.fileCheck.path {
		return
	}
	tab.fileCheck = bodyFileCheck{path: path}
	files := a.folders
	a.bus.Run(context.Background(), func(ctx context.Context) bus.Msg {
		state := bodyFileMissing
		if files.IsFile(ctx, path) {
			state = bodyFileFound
		}
		return func() { // on the UI goroutine
			if tab.fileCheck.path == path {
				tab.fileCheck.state = state
			}
		}
	})
}

// fileBody is the editor of a file body: the path, and a button to pick it.
func (a *App) fileBody(tab *reqTab) Node {
	sync := func() { editRequest(tab, editor.BodyTyped{Text: tab.file.Text}) }
	a.checkBodyFile(tab)
	look := bodyFileLooks[bodyFileUnknown]
	if tab.file.Text != "" { // asked about just above
		look = bodyFileLooks[tab.fileCheck.state]
	}
	size, sizeColor := look.text, look.color
	choose := func() {
		a.pickFile("Choose the file to send", port.OpenFile, "", func(path string) {
			tab.file = NewTextField(path)
			sync()
		})
	}
	return fileBodyTemplate.MustBuild(a.scope(markup.Scope{
		"file":      &tab.file,
		"sync":      sync,
		"choose":    choose,
		"size":      size,
		"sizeColor": sizeColor,
	}))
}

// Templates, parsed once above.
const (
	filebodySource = `<col grow="1" gap="{{looseGap}}">
  <row h="{{rowH}}" gap="{{tightGap}}">
    <field state="file" grow="1" placeholder="Path of the file to send" on-change="sync" text-menu="file"/>
    <button w="{{chooseBtnW}}" on-click="choose">Choose...</button>
  </row>
  <!-- always built, even empty, so the hint keeps its place -->
  <text small muted color="{{sizeColor}}">{{size}}</text>
  <text small muted>The file is sent as it is, with a content type from its extension.</text>
</col>
`
)
