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
	goView := Row(H(statusH), PadX(w.Gutter), Gap(w.Gutter), Bg(w.Frame), Control("statusbar"), BorderT(w.Border),
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

// The header template renders exactly what the Go view it replaced did.
func TestHeaderTemplateMatchesTheGoView(t *testing.T) {
	a, _ := newTestApp(t)
	toggle := OnClick(func() { a.menu = !a.menu })
	goView := Row(H(headerH), PadX(w.Gutter), Bg(w.Hero), Control("hero"),
		Text(a.col.Name, Bold(), Color(w.HeroText)),
		Spacer(),
		a.envChip(),
		Spacer(W(w.Gutter)),
		Row(W(headerH), toggle, Tooltip(a.hint("Settings", "settings")),
			DotsIcon(Grow(1), Color(w.HeroMuted), HoverColor(w.HeroText)),
		),
	)
	tplView := a.header()
	Layout(a.ctx, goView, 900, headerH)
	Layout(a.ctx, tplView, 900, headerH)
	if got, want := Dump(tplView), Dump(goView); got != want {
		t.Fatalf("template:\n%s\nGo:\n%s", got, want)
	}
}
