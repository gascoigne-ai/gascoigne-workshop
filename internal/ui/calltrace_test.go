package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
	"github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/view"
)

func traceText(a *App) string { return strings.Join(a.calls.Lines(), "\n") }

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
	for _, entry := range a.calls.Entries() {
		if entry.IsErr {
			errors++
		}
	}
	if errors != 1 || !strings.Contains(traceText(a), "failed") {
		t.Fatalf("errors = %d, trace:\n%s", errors, traceText(a))
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
	a.calls.Info(a.now(), "→ GET https://one.test")
	if dump := view.Dump(a.statusBar()); !strings.Contains(dump, `"Log"`) {
		t.Fatalf("no Log button in the status bar:\n%s", dump)
	}
	if a.calls.View(a) != nil {
		t.Fatal("the trace view is shown while closed")
	}
	a.calls.Open()
	a.step(core.Input{X: 5000, Y: 5000})
	if dump := view.Dump(a.calls.View(a)); !strings.Contains(dump, "Log") {
		t.Fatalf("trace view:\n%s", dump)
	}
	a.calls.Close()
	if a.calls.View(a) != nil {
		t.Fatal("the trace view stayed after close")
	}
}

func TestEscapeClosesTheTraceView(t *testing.T) {
	a, _ := newTestApp(t)
	a.calls.Open()
	a.step(core.Input{X: 5000, Y: 5000, Escape: true})
	if a.calls.Shown() {
		t.Fatal("Escape left the trace view open")
	}
}
