package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/sizes"
	"testing"

	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/widget"
)

// The inline status bar template renders exactly what the Go view it replaced did.
func TestStatusBarTemplateMatchesTheGoView(t *testing.T) {
	a := testApp(t)
	a.notices.SetStatus("Sent 200 OK in 120ms")
	goView := Row(H(sizes.StatusH), PadX(w.Gutter), Gap(w.Gutter), Bg(w.Frame), Control("statusbar"), BorderT(w.Border),
		Text(a.notices.Status(), Small(), Muted(), Ellipsis(), Grow(1)),
		Button("Log", W(44), H(20), OnClick(a.calls.Open), Tooltip("Show the log of calls")),
		Row(OnClick(a.OpenWelcome), Tooltip("What is in this version"),
			Text(version, Small(), Muted()),
		),
	)
	tplView := a.statusBar()
	Layout(a.ctx, goView, 900, sizes.StatusH)
	Layout(a.ctx, tplView, 900, sizes.StatusH)
	if got, want := Dump(tplView), Dump(goView); got != want {
		t.Fatalf("template:\n%s\nGo:\n%s", got, want)
	}
}

// The header template renders exactly what the Go view it replaced did.
func TestHeaderTemplateMatchesTheGoView(t *testing.T) {
	a := testApp(t)
	toggle := OnClick(func() { a.prefs.Toggle() })
	goView := Row(H(headerH), PadX(w.Gutter), Bg(w.Hero), Control("hero"),
		Text(a.col().Name, Bold(), Ellipsis(), Shrink(), MinW(48), Color(w.HeroText)),
		Spacer(),
		a.envChip(),
		Spacer(W(w.Gutter)),
		Row(W(headerH), toggle, Tooltip(a.hint("Settings", "settings")),
			GearIcon(Grow(1), Color(w.HeroMuted), HoverColor(w.HeroText)),
		),
	)
	tplView := a.header()
	Layout(a.ctx, goView, 900, headerH)
	Layout(a.ctx, tplView, 900, headerH)
	if got, want := Dump(tplView), Dump(goView); got != want {
		t.Fatalf("template:\n%s\nGo:\n%s", got, want)
	}
}
