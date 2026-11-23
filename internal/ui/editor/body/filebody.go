package body

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"

	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

var fileTemplate = markup.Must("filebody.vml", fileSource)

// fileKnowledge is what is known of the file a file body names.
type fileKnowledge int

const (
	fileUnknown fileKnowledge = iota // not asked yet, or the answer is on its way
	fileFound                        // a file, not a folder
	fileMissing                      // nothing there, a folder, or a path that cannot be looked at
)

// fileLooks is how each state is shown under the path; no color is the
// muted text.
var fileLooks = map[fileKnowledge]struct {
	text  string
	color w.Token
}{
	fileUnknown: {"", ""},
	fileFound:   {"found", ""},
	fileMissing: {"not found", w.Error},
}

// fileCheck is the last question about a file body's path. The editor is
// built every frame, so the disk is asked once per path, off the UI
// goroutine, and the editor shows the answer it has.
type fileCheck struct {
	path  string // the path asked about
	state fileKnowledge
}

// fileState is what the disk said about the path of tab's file body.
func (c *Component) fileState(tab *tabs.Tab) fileKnowledge { return c.state(tab).fileCheck.state }

// checkFile asks the disk about the tab's path when it is not the one last
// asked about; an answer for an older path is dropped.
func (c *Component) checkFile(tab *tabs.Tab) {
	state := c.state(tab)
	path := state.file.Text
	if path == "" || path == state.fileCheck.path {
		return
	}
	state.fileCheck = fileCheck{path: path}
	c.deps.Files.Check(path, func(found bool) {
		if state.fileCheck.path != path {
			return
		}
		state.fileCheck.state = fileMissing
		if found {
			state.fileCheck.state = fileFound
		}
	})
}

// fileBody is the editor of a file body: the path, and a button to pick it.
func (c *Component) fileBody(ctx *w.Ctx, tab *tabs.Tab) Node {
	state := c.state(tab)
	c.checkFile(tab)
	look := fileLooks[fileUnknown]
	if state.file.Text != "" { // asked about just above
		look = fileLooks[state.fileCheck.state]
	}
	return fileTemplate.MustBuild(sizes.Scope(ctx, markup.Scope{
		"file":      &state.file,
		"sync":      func() { c.pathTyped(tab) },
		"choose":    func() { c.choose(tab) },
		"size":      look.text,
		"sizeColor": look.color,
	}))
}

// pathTyped makes the text of the path field the request's body.
func (c *Component) pathTyped(tab *tabs.Tab) {
	c.edit(tab, requestedit.BodyTyped{Text: c.state(tab).file.Text})
}

// choose lets the user pick the file and makes its path the body.
func (c *Component) choose(tab *tabs.Tab) {
	c.deps.Files.Pick("Choose the file to send", func(path string) {
		*c.File(tab) = widget.NewTextField(path)
		c.pathTyped(tab)
	})
}

// Template, parsed once above.
const fileSource = `<col grow="1" gap="{{looseGap}}">
  <row h="{{rowH}}" gap="{{tightGap}}">
    <field state="file" grow="1" placeholder="Path of the file to send" on-change="sync" text-menu="file"/>
    <button w="{{chooseBtnW}}" on-click="choose">Choose...</button>
  </row>
  <!-- always built, even empty, so the hint keeps its place -->
  <text small muted color="{{sizeColor}}">{{size}}</text>
  <text small muted>The file is sent as it is, with a content type from its extension.</text>
</col>
`
