package ui

import (
	"testing"

	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"
)

// The inline status bar template renders exactly what the Go view it replaced did.
func TestStatusBarTemplateMatchesTheGoView(t *testing.T) {
	a, _ := newTestApp(t)
	a.status = "Sent 200 OK in 120ms"
	goView := Row(H(statusH), PadX(w.Pad), Gap(w.Pad), Bg(w.Frame), Control("statusbar"), BorderTop(w.Border),
		Text(a.status, Small(), Muted(), Ellipsis(), Grow(1)),
		Text(version, Small(), Muted()),
	)
	tplView := a.statusBar()
	Layout(a.ctx, goView, 900, statusH)
	Layout(a.ctx, tplView, 900, statusH)
	if got, want := Dump(tplView), Dump(goView); got != want {
		t.Fatalf("template:\n%s\nGo:\n%s", got, want)
	}
}
