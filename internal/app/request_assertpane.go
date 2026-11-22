package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/kernel/request"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/assert"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/editor"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// runTests checks the tab's assertions against its shown response.
func (a *App) runTests(tab *reqTab) {
	tab.testResults = assert.Run(tab.Req.Tests, tab.Resp)
	for _, c := range assert.Captures(tab.Req.Tests, tab.Resp) {
		a.setVariable(tab.Req, c.Name, c.Value)
	}
	if len(tab.testResults) == 0 {
		return
	}
	if sum := assert.Summary(tab.testResults); sum != "" && tab.Resp != nil && tab.Resp.Err == "" {
		a.status += "  Tests: " + sum
	}
}

var (
	assertionsTemplate = markup.Must("assertions.vml", assertionsSource)
	resultsTemplate    = markup.Must("results.vml", resultsSource)
	noResultsTemplate  = markup.Must("results-none.vml", resultsNoneSource)
)

// assertionsHelp says how assertions are written; it goes through the
// scope, as a template would read its {{token}} as a variable. Text does
// not wrap by itself, so the lines are broken to fit the narrowest pane
// (396 px inside 420) in the small font.
const assertionsHelp = "One assertion per line: status == 200, time < 500ms, header\n" +
	"Content-Type contains json, body contains \"ok\", json items.length > 0,\n" +
	"json items[0].name == \"a\". set token = json access_token keeps a value\n" +
	"as {{token}} for later requests. Lines starting with # are notes."

// assertionsPane is where the assertions are written, one per line.
func (a *App) assertionsPane(_ pane, tab *reqTab, _ *ScrollState) Node {
	sync := func() {
		editRequest(tab, editor.TestsTyped{Text: tab.tests.Text})
		tab.testResults = assert.Run(tab.Req.Tests, tab.Resp) // live, against the response on screen
	}
	return assertionsTemplate.MustBuild(markup.Scope{
		"looseGap":  w.LooseGap,
		"gutter":    w.Gutter,
		"help":      assertionsHelp,
		"tests":     &tab.tests,
		"testsText": &tab.tests.TextField,
		"sync":      sync,
	})
}

// resultRow is what the results template shows of one assertion: its
// line marked as passed or failed, in the matching color.
type resultRow struct {
	Text  string
	Color w.Token
}

// resultsPane lists what the assertions found.
func (a *App) resultsPane(_ pane, tab *reqTab, _ *ScrollState) Node {
	if len(tab.testResults) == 0 {
		return noResultsTemplate.MustBuild(markup.Scope{"gutter": w.Gutter})
	}
	rows := make([]resultRow, 0, len(tab.testResults))
	for _, r := range tab.testResults {
		mark, color := "\u2713 ", w.OK
		text := r.Line
		if !r.OK {
			mark, color = "\u2717 ", w.Error
			if r.Detail != "" {
				text += "  \u2014  " + r.Detail
			}
		}
		rows = append(rows, resultRow{Text: mark + text, Color: color})
	}
	return resultsTemplate.MustBuild(a.scope(markup.Scope{
		"gutter":  w.Gutter,
		"scroll":  &tab.respScroll,
		"summary": assert.Summary(tab.testResults),
		"rows":    rows,
	}))
}

// setVariable stores a value captured from the response to req in the
// active environment of req's collection, or in that collection's
// variables when none is active (request.UpsertVar).
func (a *App) setVariable(req *request.Request, name, value string) {
	entry, ok := a.entryFor(req)
	if !ok {
		return
	}
	if env := entry.Environment(); env != nil {
		env.Vars = request.UpsertVar(env.Vars, name, value)
		return
	}
	entry.Collection.Vars = request.UpsertVar(entry.Collection.Vars, name, value)
}

// Templates, parsed once above.
const (
	assertionsSource = `<!-- The request's Tests pane: how assertions are written, and the editor.
     The help comes from the scope, as it shows a variable's braces. -->
<col grow="1" pad="{{gutter}}" gap="{{looseGap}}">
  <lines color="muted" small>{{help}}</lines>
  <area state="tests" grow="1" bg="sunken" on-change="sync" text-menu="testsText"/>
</col>
`

	resultsSource = `<!-- The response's Tests pane: the summary, then one line per assertion,
     marked and colored by whether it passed. -->
<scroll state="scroll" pad="{{gutter}}" gap="{{nearGap}}">
  <text bold>{{summary}}</text>
  <text for="row in rows" color="{{row.Color}}" mono>{{row.Text}}</text>
</scroll>
`

	resultsNoneSource = `<!-- The response's Tests pane while the request has no assertions. -->
<col grow="1" pad="{{gutter}}">
  <text muted>No assertions. Write them in the request's Tests tab.</text>
</col>
`
)
