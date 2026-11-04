package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/assert"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
)

// runTests checks the tab's assertions against its shown response.
func (a *App) runTests(t *reqTab) {
	t.testResults = assert.Run(t.req.Tests, t.resp)
	for _, c := range assert.Captures(t.req.Tests, t.resp) {
		a.setVariable(c.Name, c.Value)
	}
	if len(t.testResults) == 0 {
		return
	}
	if sum := assert.Summary(t.testResults); sum != "" && t.resp != nil && t.resp.Err == "" {
		a.status += "  Tests: " + sum
	}
}

// assertionsPane is where the assertions are written, one per line.
func (a *App) assertionsPane(_ pane, t *reqTab, _ *ScrollState) Node {
	sync := func() {
		t.req.Tests = t.tests.Text
		t.testResults = assert.Run(t.req.Tests, t.resp) // live, against the response on screen
	}
	return Col(Grow(1), Pad(w.Gutter), Gap(8),
		Text("One assertion per line: status == 200, time < 500ms, header Content-Type contains json, body contains \"ok\", json items.length > 0, json items[0].name == \"a\". set token = json access_token keeps a value as {{token}} for later requests. Lines starting with # are notes.", Small(), Muted()),
		Area(&t.tests, Grow(1), Bg(w.Sunken), OnChange(sync), a.textMenuOpt(&t.tests.TextField, sync)),
	)
}

// resultsPane lists what the assertions found.
func (a *App) resultsPane(_ pane, t *reqTab, _ *ScrollState) Node {
	if len(t.testResults) == 0 {
		return Col(Grow(1), Pad(w.Gutter),
			Text("No assertions. Write them in the request's Tests tab.", Muted()))
	}
	rows := make([]Item, 0, len(t.testResults)+1)
	rows = append(rows, Text(assert.Summary(t.testResults), Bold()))
	for _, r := range t.testResults {
		mark, color := "\u2713 ", w.OK
		text := r.Line
		if !r.OK {
			mark, color = "\u2717 ", w.Error
			if r.Detail != "" {
				text += "  \u2014  " + r.Detail
			}
		}
		rows = append(rows, Text(mark+text, Color(color), Mono()))
	}
	return Scroll(&t.respScroll, append([]Item{Pad(w.Gutter), Gap(4)}, rows...)...)
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
