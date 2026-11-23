package app

import (
	"github.com/gascoigne-ai/gascoigne-workshop/internal/ui/editor/urlbar"
	"regexp"
	"strconv"
	"strings"
	"testing"

	. "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

var (
	dividerLine = regexp.MustCompile(`plain w=1 bg=border rect=(\d+),`)
	textLine    = regexp.MustCompile(`textB text="([^"]*)".* rect=(\d+),`)
)

// dividerXs are the x of every 1 px divider of the strip, in tree order.
func dividerXs(dump string) []int {
	var xs []int
	for _, m := range dividerLine.FindAllStringSubmatch(dump, -1) {
		x, _ := strconv.Atoi(m[1])
		xs = append(xs, x)
	}
	return xs
}

// textX is the x of the text node that shows text.
func textX(t *testing.T, dump, text string) int {
	t.Helper()
	for _, m := range textLine.FindAllStringSubmatch(dump, -1) {
		if m[1] == text {
			x, _ := strconv.Atoi(m[2])
			return x
		}
	}
	t.Fatalf("no text %q in\n%s", text, dump)
	return 0
}

// The gRPC strip reads as three chips, not as a bar glued to a label: a
// divider sits between the protocol and the method and between the method
// and Stream, and the method text keeps its distance from the divider.
func TestTheGRPCStripSeparatesItsChips(t *testing.T) {
	for _, state := range []string{"grpc-method", "grpc-stream", "grpc-no-method"} {
		t.Run(state, func(t *testing.T) {
			a := testApp(t)
			t.Cleanup(a.WaitWrites)
			tab := a.cur()
			stateNamed(t, state).setUp(t, a, tab)
			root := a.urlbar.View(a.ctx)
			Layout(a.ctx, root, urlWideW, urlH)
			dump := Dump(root)
			dividers := dividerXs(dump)
			if len(dividers) != 2 {
				t.Fatalf("%d dividers, want 2 (after the protocol and after the method):\n%s", len(dividers), dump)
			}
			label, _ := urlbar.MethodLabel(tab.Req)
			if gap := textX(t, dump, label) - dividers[0]; gap < 8 {
				t.Fatalf("the method text starts %d px after the first divider, want at least 8", gap)
			}
			if streamX := textX(t, dump, "Stream"); streamX < dividers[1] {
				t.Fatalf("Stream at %d is left of the second divider at %d", streamX, dividers[1])
			}
		})
	}
}

// Stream shows whether it is on: a tinted chip while the tab streams, a
// plain one otherwise.
func TestTheStreamChipShowsItsState(t *testing.T) {
	a := testApp(t)
	t.Cleanup(a.WaitWrites)
	tab := a.cur()
	stateNamed(t, "grpc-method").setUp(t, a, tab)
	off := Dump(layoutBar(a, tab))
	stateNamed(t, "grpc-stream").setUp(t, a, tab)
	on := Dump(layoutBar(a, tab))
	if chipLine(off) == "" || strings.Contains(chipLine(off), "bg=selected") {
		t.Fatalf("Stream is off, its chip must not be tinted:\n%s", chipLine(off))
	}
	if !strings.Contains(chipLine(on), "bg=selected") {
		t.Fatalf("Stream is on, its chip must be tinted:\n%s", chipLine(on))
	}
}

func layoutBar(app *App, tab *reqTab) Node {
	root := app.urlbar.View(app.ctx)
	Layout(app.ctx, root, urlWideW, urlH)
	return root
}

// chipLine is the line of the Stream chip's row.
func chipLine(dump string) string {
	for _, line := range strings.Split(dump, "\n") {
		if strings.Contains(line, "tip=Stream:") {
			return line
		}
	}
	return ""
}
