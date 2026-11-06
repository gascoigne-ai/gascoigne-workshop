package ui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/vml"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// focusName says which text widget of the window has the keyboard focus.
func focusName(a *App) string {
	t := a.cur()
	switch {
	case a.filter.Focused:
		return "filter"
	case t.url.Focused:
		return "url"
	case t.find.Focused:
		return "find"
	case t.body.Focused:
		return "body"
	case t.view.Focused:
		return "response"
	}
	return "none"
}

func TestTabVisitsTheWindowInReadingOrderAndCycles(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	pointTabAt(tab, jsonServer(t).URL)
	a.send(tab)
	waitForResponse(t, a, tab)
	var order []string
	for range 8 {
		a.step(core.Input{X: 5000, Y: 5000, Tab: true})
		order = append(order, focusName(a))
	}
	want := []string{"filter", "url", "response", "filter", "url", "response", "filter", "url"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("focus order %v, want %v", order, want)
	}
}

// iconOnlyWithoutHint lists the clickable elements of the window whose
// only content is an icon and that have no tooltip: a user who cannot
// tell what the icon means has nothing to read.
func iconOnlyWithoutHint(dump string) []string {
	lines := strings.Split(dump, "\n")
	var bad []string
	for i, line := range lines {
		if !strings.Contains(line, "onClick=true") || strings.Contains(line, "tip=") || i+1 >= len(lines) {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		next := lines[i+1]
		nextIndent := len(next) - len(strings.TrimLeft(next, " "))
		if nextIndent > indent && strings.Contains(next, "iconB") && !(i+2 < len(lines) && len(lines[i+2])-len(strings.TrimLeft(lines[i+2], " ")) > indent && !strings.Contains(lines[i+2], "iconB")) {
			bad = append(bad, strings.TrimSpace(line)+" -> "+strings.TrimSpace(next))
		}
	}
	return bad
}

func TestEveryIconOnlyControlHasATooltip(t *testing.T) {
	a, _ := newTestApp(t)
	a.step(core.Input{X: 5000, Y: 5000})
	root := a.view()
	vml.Layout(a.ctx, root, a.w, a.h)
	if bad := iconOnlyWithoutHint(vml.Dump(root)); len(bad) > 0 {
		t.Fatalf("icon-only controls without a tooltip:\n%s", strings.Join(bad, "\n"))
	}
}

func TestTheIconHintCheckFlagsAnIconButtonWithoutATooltip(t *testing.T) {
	dump := "flow w=44 onClick=true\n  iconB icon=dots\nflow w=44 tip=Settings onClick=true\n  iconB icon=dots\nflow onClick=true\n  textB text=\"Send\"\n"
	bad := iconOnlyWithoutHint(dump)
	if len(bad) != 1 || !strings.Contains(bad[0], "icon=dots") {
		t.Fatalf("flagged %q, want only the first control", bad)
	}
}
