package assertions

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/assert"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/send/requestedit"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/tabs"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"

	// The view DSL is dot-imported so views read like markup.
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

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

// View is the request's Tests pane of tab: how assertions are written, and
// the area they are typed in.
func (c *Component) View(_ *w.Ctx, tab *tabs.Tab, _ *widget.ScrollState) Node {
	state := c.state(tab)
	sync := func() {
		c.edit(tab, requestedit.TestsTyped{Text: state.tests.Text})
		state.results = assert.Run(tab.Req.Tests, tab.Resp) // live, against the response on screen
	}
	return assertionsTemplate.MustBuild(markup.Scope{
		"looseGap":  w.LooseGap,
		"gutter":    w.Gutter,
		"help":      assertionsHelp,
		"tests":     &state.tests,
		"testsText": &state.tests.TextField,
		"sync":      sync,
	})
}

// resultRow is what the results template shows of one assertion: its
// line marked as passed or failed, in the matching color.
type resultRow struct {
	Text  string
	Color w.Token
}

// ResultsView is the response's Tests pane of tab: what the assertions
// found, scrolled with scroll.
func (c *Component) ResultsView(ctx *w.Ctx, tab *tabs.Tab, scroll *widget.ScrollState) Node {
	results := c.Results(tab)
	if len(results) == 0 {
		return noResultsTemplate.MustBuild(markup.Scope{"gutter": w.Gutter})
	}
	rows := make([]resultRow, 0, len(results))
	for _, r := range results {
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
	return sizes.Build(ctx, resultsTemplate, markup.Scope{
		"gutter":  w.Gutter,
		"scroll":  scroll,
		"summary": assert.Summary(results),
		"rows":    rows,
	})
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
