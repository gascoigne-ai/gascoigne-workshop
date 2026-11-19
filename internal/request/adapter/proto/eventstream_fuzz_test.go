package proto

import (
	"strings"
	"testing"
)

// An event stream from a server parses to events within the size bound,
// or to an error, and never panics.
func FuzzReadEvents(fuzz *testing.F) {
	fuzz.Add(": comment\nid: 7\nevent: tick\ndata: one\n\ndata: a\ndata: b\n\nretry: 100\n\nevent: noData\n\ndata: last")
	fuzz.Add("data: {\"jsonrpc\":\"2.0\",\"method\":\"notifications/tools/list_changed\"}\n\n")
	fuzz.Add("id: 1\nevent: n\ndata: 1\n\nid: 2\nevent: n\ndata: 2\n\n")
	fuzz.Fuzz(func(t *testing.T, stream string) {
		_ = readEventStream(strings.NewReader(stream), func(event sseEvent) {
			if len(event.data) > maxEventBytes {
				t.Fatalf("an event of %d bytes passed the bound", len(event.data))
			}
		})
	})
}
