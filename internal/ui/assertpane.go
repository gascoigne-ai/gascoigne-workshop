package ui

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/assert"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
)

// runTests checks the tab's assertions against its shown response.
func (a *App) runTests(t *reqTab) {
	t.testResults = assert.Run(t.req.Tests, t.resp)
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
		Text("One assertion per line: status == 200, time < 500ms, header Content-Type contains json, body contains \"ok\", json items.length > 0, json items[0].name == \"a\". Lines starting with # are notes.", Small(), Muted()),
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
