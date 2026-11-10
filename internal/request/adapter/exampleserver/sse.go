package exampleserver

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const (
	defaultTicks   = 5
	maxTicks       = 1000
	defaultTickGap = 500 * time.Millisecond
)

// sseHandler streams "tick n" events: ?count= says how many (default 5)
// and ?interval= the gap between them (default 500ms). The stream ends
// after the last one.
func sseHandler() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		count, interval, err := tickParams(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		flusher, ok := writer.(http.Flusher)
		if !ok {
			http.Error(writer, "streaming is not supported", http.StatusInternalServerError)
			return
		}
		writer.Header().Set("Content-Type", "text/event-stream")
		writer.Header().Set("Cache-Control", "no-cache")
		for tick := 1; tick <= count; tick++ {
			fmt.Fprintf(writer, "id: %d\nevent: tick\ndata: tick %d\n\n", tick, tick)
			flusher.Flush()
			if tick == count {
				return
			}
			select {
			case <-time.After(interval):
			case <-request.Context().Done():
				return
			}
		}
	})
}

func tickParams(request *http.Request) (count int, interval time.Duration, err error) {
	count, interval = defaultTicks, defaultTickGap
	query := request.URL.Query()
	if text := query.Get("count"); text != "" {
		count, err = strconv.Atoi(text)
		if err != nil || count < 1 || count > maxTicks {
			return 0, 0, fmt.Errorf("count must be a number from 1 to %d", maxTicks)
		}
	}
	if text := query.Get("interval"); text != "" {
		interval, err = time.ParseDuration(text)
		if err != nil || interval < 0 {
			return 0, 0, fmt.Errorf("interval must be a duration such as 250ms")
		}
	}
	return count, interval, nil
}
