package app

import (
	"testing"

	reqport "github.com/gascoigne-ai/gascoigne-workshop/internal/request/port"
	w "github.com/gascoigne-ai/gascoigne-workshop/uitoolkit/core"
)

// lastLine is the text of the last log line.
func lastLine(t *testing.T, tab *reqTab) string {
	t.Helper()
	lines := tab.wsView.Lines
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
	a.logEvent(tab, reqport.Event{Dir: reqport.Received, Kind: reqport.KindMessage, Name: "initialize", Text: "hello"})
	if got := lastLine(t, tab); got != "hello" {
		t.Errorf("last line %q, want the event's data", got)
	}

	before := len(tab.wsView.Lines)
	result := `{"protocolVersion":"2025-06-18","serverInfo":{"name":"mock","version":"1"}}`
	a.applyWS(wsReceived{wsTarget{tab: tab}, reqport.Event{Dir: reqport.Info, Kind: reqport.KindInitialize, Text: result}})
	if len(tab.wsView.Lines) != before {
		t.Errorf("the MCP initialize note was logged")
	}
	if tab.Server == nil || tab.Server.Name != "mock" {
		t.Errorf("Server = %+v", tab.Server)
	}
}

func TestAnEmptyEventShowsItsLabel(t *testing.T) {
	cases := []struct {
		name  string
		event reqport.Event
		want  string
	}{
		{"named event", reqport.Event{Dir: reqport.Received, Kind: reqport.KindMessage, Name: "tick"}, "tick"},
		{"unnamed event", reqport.Event{Dir: reqport.Received, Kind: reqport.KindMessage}, "message"},
		{"sent ping", reqport.Event{Dir: reqport.Sent, Kind: reqport.KindPing}, "ping"},
		{"empty text frame", reqport.Event{Dir: reqport.Received, Kind: reqport.KindText}, ""},
		{"empty binary frame", reqport.Event{Dir: reqport.Received, Kind: reqport.KindBinary}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := testApp(t)
			tab := a.cur()
			a.logEvent(tab, tc.event)
			if got := lastLine(t, tab); got != tc.want {
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
	a.logEvent(tab, reqport.Event{Dir: reqport.Received, Kind: reqport.KindMessage, Name: "error", Text: "boom"})
	lines := tab.wsView.Lines
	spans := lines[len(lines)-1].Spans
	if got := spans[len(spans)-1]; got.Text != "boom" || got.Style != w.Text {
		t.Errorf("span %+v, want plain text", got)
	}
}
