package proto

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// maxEventBytes bounds one line and one event of an event stream (SSE and
// MCP over HTTP), so a server that never ends an event cannot grow memory
// without end.
const maxEventBytes = 16 << 20

// errEventTooLarge ends a stream whose event grew past maxEventBytes.
var errEventTooLarge = fmt.Errorf("an event is larger than %d MiB", maxEventBytes>>20)

// readEventStream parses the event-stream format from reader, emitting each
// event, until the stream ends. It returns the read error, or an error when
// a line or an event is larger than maxEventBytes.
func readEventStream(reader io.Reader, emit func(sseEvent)) error {
	sc := bufio.NewScanner(reader)
	sc.Buffer(make([]byte, 64<<10), maxEventBytes)
	return readEvents(sc, emit)
}

// readEvents parses the event-stream format from sc until it ends and
// calls emit for each event that has data. Comment lines (":") and the
// retry field are ignored. An event whose data grows past maxEventBytes
// is dropped and ends the stream with errEventTooLarge.
func readEvents(sc *bufio.Scanner, emit func(sseEvent)) error {
	var ev sseEvent
	var data []string
	size := 0
	flush := func() {
		if len(data) > 0 {
			ev.data = strings.Join(data, "\n")
			emit(ev)
		}
		ev, data, size = sseEvent{id: ev.id}, nil, 0 // the id sticks until a new one is sent
	}
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			flush()
			continue
		}
		field, value, _ := strings.Cut(line, ":")
		value = strings.TrimPrefix(value, " ")
		switch {
		case strings.HasPrefix(line, ":"):
		case field == "event":
			ev.name = value
		case field == "data":
			size += len(value) + 1 // the newline that joins it to the next line
			if size > maxEventBytes {
				return errEventTooLarge
			}
			data = append(data, value)
		case field == "id":
			ev.id = value
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	flush()
	return nil
}
