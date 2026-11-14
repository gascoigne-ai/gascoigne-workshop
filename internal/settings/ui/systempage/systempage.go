// Package systempage is the System page of the settings dialog: what the
// program runs on, the display and the GPU rendering, read-only, with a
// button that copies the report as text. The numbers are read when the page
// is built, so they are as fresh as the last redraw.
package systempage

import (
	"embed"

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

//go:embed systempage.vml
var templates embed.FS

var pageTemplate = markup.MustLoad(templates, "systempage.vml")

// View is the page: a table of labelled values for each section of the
// report, then Copy report.
func View(host Host) Node {
	report := host.System()
	return pageTemplate.MustBuild(markup.Scope{
		"sections": report.Sections,
		"copy":     func() { host.CopyText(report.Text()) },
		"labelH":   w.LabelH,
		"rowH":     w.RowH,
	})
}
