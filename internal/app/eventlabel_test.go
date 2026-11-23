package app

import (
	"testing"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// lastLine is the text of the last log line.
func lastLine(t *testing.T, a *App, tab *reqTab) string {
	t.Helper()
	lines := a.ws.Lines(tab.Tab)
	if len(lines) == 0 {
		t.Fatal("nothing was logged")
	}
	spans := lines[len(lines)-1].Spans
	return spans[len(spans)-1].Text
}

// A server could name an event "initialize" and the log dropped it, as
// it drops the MCP note of that kind.
func TestAnSSEEventNamedInitializeIsLogged(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	a.ws.Log(tab.Tab, request.Event{Dir: request.Received, Kind: request.KindMessage, Name: "initialize", Text: "hello"})
	if got := lastLine(t, a, tab); got != "hello" {
		t.Errorf("last line %q, want the event's data", got)
	}

	before := len(a.ws.Lines(tab.Tab))
	result := `{"protocolVersion":"2025-06-18","serverInfo":{"name":"mock","version":"1"}}`
	a.applyWS(wsReceived{wsTarget{tab: tab}, request.Event{Dir: request.Info, Kind: request.KindInitialize, Text: result}})
	if len(a.ws.Lines(tab.Tab)) != before {
		t.Errorf("the MCP initialize note was logged")
	}
	if tab.Server == nil || tab.Server.Name != "mock" {
		t.Errorf("Server = %+v", tab.Server)
	}
}

func TestAnEmptyEventShowsItsLabel(t *testing.T) {
	cases := []struct {
		name  string
		event request.Event
		want  string
	}{
		{"named event", request.Event{Dir: request.Received, Kind: request.KindMessage, Name: "tick"}, "tick"},
		{"unnamed event", request.Event{Dir: request.Received, Kind: request.KindMessage}, "message"},
		{"sent ping", request.Event{Dir: request.Sent, Kind: request.KindPing}, "ping"},
		{"empty text frame", request.Event{Dir: request.Received, Kind: request.KindText}, ""},
		{"empty binary frame", request.Event{Dir: request.Received, Kind: request.KindBinary}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := testApp(t)
			tab := a.cur()
			a.ws.Log(tab.Tab, tc.event)
			if got := lastLine(t, a, tab); got != tc.want {
				t.Errorf("line %q, want %q", got, tc.want)
			}
		})
	}
}

// Pins that the peer's name never styles a line: only the session's own
// kind and direction do.
func TestAnEventNamedErrorRendersAsNormalText(t *testing.T) {
	a := testApp(t)
	tab := a.cur()
	a.ws.Log(tab.Tab, request.Event{Dir: request.Received, Kind: request.KindMessage, Name: "error", Text: "boom"})
	lines := a.ws.Lines(tab.Tab)
	spans := lines[len(lines)-1].Spans
	if got := spans[len(spans)-1]; got.Text != "boom" || got.Style != w.Text {
		t.Errorf("span %+v, want plain text", got)
	}
}
