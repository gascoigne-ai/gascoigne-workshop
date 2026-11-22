// Package systempage is the System page of the settings dialog: what the
// program runs on, the display and the GPU rendering, read-only, with a
// button that copies the report as text. The numbers are read when the page
// is built, so they are as fresh as the last redraw. The page was the
// user's request. The Ebitengine side comes in through port.SystemInfo, so
// tests need no display.
package systempage

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/settings"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Host is what the page needs of the application.
type Host interface {
	// System is the report as it is now.
	System() settings.Report
	// CopyText puts text on the clipboard.
	CopyText(text string)
}

// Widths of the page, by role.
const (
	labelColW = 150 // the label column of the report table
	copyBtnW  = 110 // the Copy report button
)

var pageTemplate = markup.Must("systempage.vml", systempageSource)

// scope is the sizes of the faces the context draws with plus the page's own
// entries, so a template holds no size as a number.
func scope(sizes w.Sizes, own markup.Scope) markup.Scope {
	merged := sizes.Scope()
	for name, value := range own {
		merged[name] = value
	}
	return merged
}

// View is the page: a table of labelled values for each section of the
// report, then Copy report.
func View(ctx *w.Ctx, host Host) Node {
	report := host.System()
	return pageTemplate.MustBuild(scope(ctx.Sizes, markup.Scope{
		"sections":  report.Sections,
		"copy":      func() { host.CopyText(report.Text()) },
		"labelColW": labelColW,
		"copyBtnW":  copyBtnW,
	}))
}

// Templates, parsed once above.
const (
	systempageSource = `<!-- The System page: a table of labelled values for each section of the
     report, then the button that copies it all as text. -->
<col gap="{{looseGap}}">
  <col for="section in sections">
    <caption h="{{labelH}}">{{section.Title}}</caption>
    <row for="row in section.Rows" h="{{rowH}}" gap="{{looseGap}}">
      <text small w="{{labelColW}}" color="muted">{{row.Label}}</text>
      <text small ellipsis grow="1">{{row.Value}}</text>
    </row>
  </col>
  <row h="{{rowH}}"><button w="{{copyBtnW}}" on-click="copy">Copy report</button></row>
</col>
`
)
