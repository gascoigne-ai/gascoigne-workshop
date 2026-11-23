// Package welcome is the dialog with the release notes of this version,
// shown the first time a version runs and again from the version in the
// status bar.
//
// Whether a version was welcomed is the application's to remember (it is
// part of the saved state), so the component only shows the dialog. The
// dialog itself belongs to the window, which holds one at a time: the
// component asks for it through Dialogs and keeps only the scroll position
// of its own content.
package welcome

import (
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

const (
	// welcomeW is wider than a message dialog: the notes are lines of prose.
	welcomeW = 560
	// welcomeChrome is what the dialog needs besides the notes: the title
	// bar, the lead text, the button and the window's margins.
	welcomeChrome = 260
	// welcomeCols is how many characters of a point fit a line beside its bullet.
	welcomeCols  = 64
	welcomeTitle = "Welcome to Gascoigne Workshop 0.1.0 alpha"
	welcomeLead  = "The first alpha. Here is what is in it and what to expect."
)

var template = markup.Must("welcome.vml", source)

// Component is the welcome dialog.
type Component struct {
	deps   Deps
	scroll ScrollState // scroll of the notes
}

// New is the component; nothing is shown until Open.
func New(deps Deps) *Component { return &Component{deps: deps} }

// Open shows the release notes, scrolled to the top.
func (c *Component) Open() {
	c.scroll = ScrollState{}
	var closeDialog func()
	state := DialogState{Title: welcomeTitle, Body: welcomeLead, Buttons: []DialogButton{
		{Label: "Get started", Primary: true, Run: func() { closeDialog() }},
	}}
	closeDialog = c.deps.Dialogs.Open(state, c.content, welcomeW)
}

// content builds the notes between the lead text and the button.
func (c *Component) content(*w.Ctx) Node {
	_, height := c.deps.Window.Size()
	return template.MustBuild(markup.Scope{
		"looseGap": w.LooseGap,
		"gutter":   w.Gutter,
		"nearGap":  w.NearGap,
		"sections": wrapped(welcomeNotes, welcomeCols),
		"scroll":   &c.scroll,
		"maxH":     max(height-welcomeChrome, 120),
	})
}

// Templates, parsed once above.
const source = `<!-- The release notes: a heading and a bulleted list per section, scrolling in a short window. -->
<scroll state="scroll" max-h="{{maxH}}" gap="{{gutter}}">
  <col for="section in sections" gap="{{nearGap}}">
    <text bold>{{section.Title}}</text>
    <row for="item in section.Items" gap="{{looseGap}}" align-items="start">
      <icon name="dot" color="muted" margin-t="{{looseGap}}"/>
      <lines grow="1" color="muted">{{item}}</lines>
    </row>
  </col>
</scroll>
`
