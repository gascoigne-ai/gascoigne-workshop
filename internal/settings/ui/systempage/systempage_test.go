package systempage

import (
	"image/color"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/settings/model/sysinfo"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

type fakeHost struct {
	report sysinfo.Report
	copied string
}

func (h *fakeHost) System() sysinfo.Report { return h.report }
func (h *fakeHost) CopyText(text string)   { h.copied = text }

func newHost() *fakeHost {
	return &fakeHost{report: sysinfo.Report{Sections: []sysinfo.Section{
		{Title: "Rendering", Rows: []sysinfo.Row{{Label: "Graphics library", Value: "Metal"}, {Label: "Vsync", Value: "on"}}},
		{Title: "System", Rows: []sysinfo.Row{{Label: "OS", Value: "darwin/arm64"}}},
	}}}
}

func newCtx(t *testing.T) *w.Ctx {
	t.Helper()
	ctx, err := w.NewCtx(w.Theme{Colors: map[w.Token]color.RGBA{w.Text: w.RGB(0xffffff)}, FontSize: 14, SmallSize: 12, MonoSize: 13})
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

func TestThePageShowsEverySectionWithItsRows(t *testing.T) {
	root := View(newHost())
	Layout(newCtx(t), root, 600, 600)
	dump := Dump(root)
	for _, want := range []string{"Rendering", "Graphics library", "Metal", "Vsync", "System", "OS", "darwin/arm64", "Copy report"} {
		if !strings.Contains(strings.ToLower(dump), strings.ToLower(want)) { // captions are shown in capitals
			t.Errorf("page has no %q:\n%s", want, dump)
		}
	}
}

func TestCopyPutsTheWholeReportOnTheClipboard(t *testing.T) {
	host := newHost()
	root := View(host)
	ctx := newCtx(t)
	Layout(ctx, root, 600, 600)
	var button *El
	var find func(e *El)
	find = func(e *El) {
		if strings.Contains(strings.SplitN(Dump(e), "\n", 2)[0], "onClick=true") {
			button = e
			return
		}
		for _, k := range e.Children() {
			if button == nil {
				find(k)
			}
		}
	}
	find(root)
	if button == nil {
		t.Fatalf("no button in:\n%s", Dump(root))
	}
	r := button.Rect()
	ctx.In = w.Input{X: r.X + r.W/2, Y: r.Y + r.H/2, Click: true}
	Update(ctx, root)
	if host.copied != host.report.Text() {
		t.Fatalf("copied %q, want the report text %q", host.copied, host.report.Text())
	}
}
