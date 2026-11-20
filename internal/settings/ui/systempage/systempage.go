// Package systempage is the System page of the settings dialog: what the
// program runs on, the display and the GPU rendering, read-only, with a
// button that copies the report as text. The numbers are read when the page
// is built, so they are as fresh as the last redraw. The page was the
// user's request. The Ebitengine side comes in through port.SystemInfo, so
// tests need no display.
package systempage

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/sysinfo"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/markup"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

// Host is what the page needs of the application.
type Host interface {
	// System is the report as it is now.
	System() sysinfo.Report
	// CopyText puts text on the clipboard.
	CopyText(text string)
}

var pageTemplate = markup.Must("systempage.vml", systempageSource)

// View is the page: a table of labelled values for each section of the
// report, then Copy report.
func View(host Host) Node {
	report := host.System()
	return pageTemplate.MustBuild(markup.Scope{
		"looseGap": w.LooseGap,
		"sections": report.Sections,
		"copy":     func() { host.CopyText(report.Text()) },
		"labelH":   w.LabelH,
		"rowH":     w.RowH,
	})
}

// Templates, parsed once above.
const (
	systempageSource = `<!-- The System page: a table of labelled values for each section of the
     report, then the button that copies it all as text. -->
<col gap="{{looseGap}}">
  <col for="section in sections">
    <caption h="{{labelH}}">{{section.Title}}</caption>
    <row for="row in section.Rows" h="{{rowH}}" gap="{{looseGap}}">
      <text small w="150" color="muted">{{row.Label}}</text>
      <text small ellipsis grow="1">{{row.Value}}</text>
    </row>
  </col>
  <row h="{{rowH}}"><button w="110" on-click="copy">Copy report</button></row>
</col>
`
)
