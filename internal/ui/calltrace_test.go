package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

func traceText(a *App) string { return strings.Join(a.traceLines(), "\n") }

func TestASendIsTracedWithItsOutcome(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	server := jsonServer(t)
	pointTabAt(tab, server.URL)
	a.send(tab)
	waitForResponse(t, a, tab)

	text := traceText(a)
	for _, want := range []string{"→ " + tab.req.Method + " " + server.URL, "← 200", "11B"} {
		if !strings.Contains(text, want) {
			t.Errorf("trace lacks %q:\n%s", want, text)
		}
	}
}

func TestAFailedSendIsTracedAsAnError(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	pointTabAt(tab, "http://127.0.0.1:1/")
	a.send(tab)
	waitForResponse(t, a, tab)

	var errors int
	for _, entry := range a.trace {
		if entry.isErr {
			errors++
		}
	}
	if errors != 1 || !strings.Contains(traceText(a), "failed") {
		t.Fatalf("errors = %d, trace:\n%s", errors, traceText(a))
	}
}

func TestTheTraceKeepsOnlyTheNewestEntries(t *testing.T) {
	a, _ := newTestApp(t)
	for i := range traceLimit + 50 {
		a.traceInfo(fmt.Sprintf("entry %d", i))
	}
	lines := a.traceLines()
	if len(lines) != traceLimit {
		t.Fatalf("%d lines, want %d", len(lines), traceLimit)
	}
	if !strings.HasSuffix(lines[len(lines)-1], fmt.Sprintf("entry %d", traceLimit+49)) {
		t.Fatalf("the newest entry is missing, last line %q", lines[len(lines)-1])
	}
}

func TestTheTraceFilterNarrowsAndClearEmpties(t *testing.T) {
	a, _ := newTestApp(t)
	a.traceInfo("→ GET https://one.test")
	a.traceInfo("→ GET https://two.test")
	a.traceFilter.Text = "TWO"
	if lines := a.traceLines(); len(lines) != 1 || !strings.Contains(lines[0], "two.test") {
		t.Fatalf("filtered = %q", lines)
	}
	a.traceFilter.Text = ""
	a.clearTrace()
	if len(a.traceLines()) != 0 {
		t.Fatal("clear left entries")
	}
}

func TestConnectionEventsAreTraced(t *testing.T) {
	a, _ := newTestApp(t)
	tab := a.cur()
	tab.req.URL = "ws://example.test/socket"
	a.applyWS(wsEvent{tab: tab, gen: tab.wsGen, err: fmt.Errorf("dial refused")})
	if text := traceText(a); !strings.Contains(text, "dial refused") || !strings.Contains(text, "ws://example.test/socket") {
		t.Fatalf("trace:\n%s", text)
	}
}

func TestTheStatusBarLogButtonOpensTheTraceView(t *testing.T) {
	a, _ := newTestApp(t)
	a.traceInfo("→ GET https://one.test")
	if dump := view.Dump(a.statusBar()); !strings.Contains(dump, `"Log"`) {
		t.Fatalf("no Log button in the status bar:\n%s", dump)
	}
	if a.traceView() != nil {
		t.Fatal("the trace view is shown while closed")
	}
	a.openTrace()
	a.step(core.Input{X: 5000, Y: 5000})
	if dump := view.Dump(a.traceView()); !strings.Contains(dump, "Log") {
		t.Fatalf("trace view:\n%s", dump)
	}
	a.closeTrace()
	if a.traceView() != nil {
		t.Fatal("the trace view stayed after close")
	}
}

func TestEscapeClosesTheTraceView(t *testing.T) {
	a, _ := newTestApp(t)
	a.openTrace()
	a.step(core.Input{X: 5000, Y: 5000, Escape: true})
	if a.traceShown {
		t.Fatal("Escape left the trace view open")
	}
}
