package app

import (
	"strings"

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

// welcomeSection is one block of the release notes: a heading and its points.
type welcomeSection struct {
	Title string
	Items []string
}

// welcomeNotes are the release notes of this version, shown by the welcome
// dialog; CHANGELOG.md holds the same list for the release page.
var welcomeNotes = []welcomeSection{
	{"Early access", []string{
		"0.1.0 is an alpha: ready to use every day, and still growing fast, so details will keep improving.",
		"Your collections are plain files: read them, diff them and keep them in git.",
		"Your feedback shapes what comes next. Tell us what you find.",
	}},
	{"Requests", []string{
		"REST, WebSocket, SSE, MCP (tools explorer, streamable HTTP and stdio) and gRPC (reflection or .proto files, unary calls and streams) share one window.",
		"Params, headers, bodies (JSON, text, form, multipart, file), auth and assertions.",
		"{{variables}} from the collection, its folders and environments.",
	}},
	{"Collections", []string{
		"A collection is a folder of JSON files; secrets stay in a separate file out of version control.",
		"Open several side by side; create, rename, save, reorder, close and delete them from the sidebar.",
		"The tree supports rename, duplicate, move and delete; tabs and tree roots show the collection's color.",
	}},
	{"Responses", []string{
		"Pretty, raw and compact views, with foldable and searchable JSON.",
		"History, a timeline, a call log and a log of the requests as sent.",
	}},
	{"Import and export", []string{
		"Import cURL commands, OpenAPI specs and Postman collections; export to Postman.",
		"Client configs (.workshop.yaml, .json, .toml) in release builds.",
	}},
	{"Make it yours", []string{
		"Dark and light themes, JSON theme files and a colors page.",
		"Settings in one settings.json, with a form and a raw editor; every shortcut can be rebound.",
		"Send with Ctrl+Enter, jump to the URL with Ctrl+L, close a collection with Ctrl+Shift+W.",
		"Native file dialogs on macOS, Windows and Linux (zenity, kdialog, yad, qarma or matedialog), else an in-app file browser.",
	}},
}

// wrapped is sections with every point broken into lines of at most cols
// characters: a Lines column draws what it is given and never wraps.
func wrapped(sections []welcomeSection, cols int) []welcomeSection {
	out := make([]welcomeSection, len(sections))
	for i, section := range sections {
		out[i].Title = section.Title
		for _, item := range section.Items {
			out[i].Items = append(out[i].Items, wrap(item, cols))
		}
	}
	return out
}

// wrap breaks text at spaces into lines of at most limit characters; a word
// longer than that keeps a line of its own.
func wrap(text string, limit int) string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(text) {
		switch {
		case line == "":
			line = word
		case len(line)+1+len(word) <= limit:
			line += " " + word
		default:
			lines = append(lines, line)
			line = word
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

var welcomeTemplate = markup.Must("welcome.vml", welcomeSource)

// openWelcome shows the release notes and remembers that this version was
// welcomed, so the dialog opens by itself only once per version.
func (a *App) openWelcome() {
	a.welcomed = version
	d := &dialog{width: welcomeW, DialogState: DialogState{Title: welcomeTitle, Body: welcomeLead, Buttons: []DialogButton{
		{Label: "Get started", Primary: true, Run: a.closeDialog},
	}}}
	d.content = func() Node {
		return welcomeTemplate.MustBuild(markup.Scope{
			"looseGap": w.LooseGap,
			"sections": wrapped(welcomeNotes, welcomeCols),
			"scroll":   &d.scroll,
			"maxH":     max(a.h-welcomeChrome, 120),
		})
	}
	a.dialog = d
}

// Templates, parsed once above.
const welcomeSource = `<!-- The release notes: a heading and a bulleted list per section, scrolling in a short window. -->
<scroll state="scroll" max-h="{{maxH}}" gap="12">
  <col for="section in sections" gap="4">
    <text bold>{{section.Title}}</text>
    <row for="item in section.Items" gap="{{looseGap}}" align-items="start">
      <text muted>•</text>
      <lines grow="1" color="muted">{{item}}</lines>
    </row>
  </col>
</scroll>
`
