package ui

import (
	"embed"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/controller/editor"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/request/model/assert"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// runTests checks the tab's assertions against its shown response.
func (a *App) runTests(t *reqTab) {
	t.testResults = assert.Run(t.Req.Tests, t.Resp)
	for _, c := range assert.Captures(t.Req.Tests, t.Resp) {
		a.setVariable(c.Name, c.Value)
	}
	if len(t.testResults) == 0 {
		return
	}
	if sum := assert.Summary(t.testResults); sum != "" && t.Resp != nil && t.Resp.Err == "" {
		a.status += "  Tests: " + sum
	}
}

//go:embed assertions.vml results.vml results-none.vml
var assertTemplates embed.FS

var (
	assertionsTemplate = markup.MustLoad(assertTemplates, "assertions.vml")
	resultsTemplate    = markup.MustLoad(assertTemplates, "results.vml")
	noResultsTemplate  = markup.MustLoad(assertTemplates, "results-none.vml")
)

// assertionsHelp says how assertions are written; it goes through the
// scope, as a template would read its {{token}} as a variable. Text does
// not wrap by itself, so the lines are broken to fit the narrowest pane.
const assertionsHelp = "One assertion per line: status == 200, time < 500ms,\n" +
	"header Content-Type contains json, body contains \"ok\",\n" +
	"json items.length > 0, json items[0].name == \"a\".\n" +
	"set token = json access_token keeps a value as {{token}}\n" +
	"for later requests. Lines starting with # are notes."

// assertionsPane is where the assertions are written, one per line.
func (a *App) assertionsPane(_ pane, t *reqTab, _ *ScrollState) Node {
	sync := func() {
		editRequest(t, editor.TestsTyped{Text: t.tests.Text})
		t.testResults = assert.Run(t.Req.Tests, t.Resp) // live, against the response on screen
	}
	return assertionsTemplate.MustBuild(markup.Scope{
		"gutter":    w.Gutter,
		"help":      assertionsHelp,
		"tests":     &t.tests,
		"testsText": &t.tests.TextField,
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
func (a *App) resultsPane(_ pane, t *reqTab, _ *ScrollState) Node {
	if len(t.testResults) == 0 {
		return noResultsTemplate.MustBuild(markup.Scope{"gutter": w.Gutter})
	}
	rows := make([]resultRow, 0, len(t.testResults))
	for _, r := range t.testResults {
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
	return resultsTemplate.MustBuild(markup.Scope{
		"gutter":  w.Gutter,
		"scroll":  &t.respScroll,
		"summary": assert.Summary(t.testResults),
		"rows":    rows,
	})
}

// setVariable stores a captured value in the active environment, or in the
// collection's variables when none is active (model.UpsertVar).
func (a *App) setVariable(name, value string) {
	if env := a.activeEnv(); env != nil {
		env.Vars = model.UpsertVar(env.Vars, name, value)
		return
	}
	a.col.Vars = model.UpsertVar(a.col.Vars, name, value)
}
