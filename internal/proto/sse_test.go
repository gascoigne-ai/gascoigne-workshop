package proto

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/model"
)

func TestReadEventsFollowsTheStreamFormat(t *testing.T) {
	stream := ": comment\nid: 7\nevent: tick\ndata: one\n\ndata: a\ndata: b\n\nretry: 100\n\nevent: noData\n\ndata: last"
	var got []sseEvent
	readEvents(bufio.NewScanner(strings.NewReader(stream)), func(e sseEvent) { got = append(got, e) })
	want := []sseEvent{{"tick", "one", "7"}, {"", "a\nb", "7"}, {"", "last", "7"}}
	if len(got) != len(want) {
		t.Fatalf("%+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func sseServer(t *testing.T, events int, hold bool) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "text/event-stream" {
			http.Error(w, "wrong Accept", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for i := 1; i <= events; i++ {
			fmt.Fprintf(w, "id: %d\nevent: n\ndata: %d\n\n", i, i)
			w.(http.Flusher).Flush()
		}
		if hold {
			<-r.Context().Done()
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestSSESessionLogsEventsAndEndsWithTheStream(t *testing.T) {
	s, err := SSE{}.Connect(context.Background(), model.Request{URL: sseServer(t, 2, false)})
	if err != nil {
		t.Fatal(err)
	}
	if e := next(t, s); e.Dir != Info || !strings.Contains(e.Text, "Connected") {
		t.Fatalf("%+v", e)
	}
	if e := next(t, s); e.Dir != Received || e.Kind != "n" || e.Text != "[1] 1" {
		t.Fatalf("%+v", e)
	}
	next(t, s)
	if e := next(t, s); e.Kind != "close" || !strings.Contains(e.Text, "ended") {
		t.Fatalf("%+v", e)
	}
	select {
	case _, ok := <-s.Events():
		if ok {
			t.Fatal("events must end with the stream")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("events not closed")
	}
	if s.Send("x") == nil || s.Ping() == nil {
		t.Fatal("a receive-only session must refuse sends")
	}
}

func TestSSECloseStopsAHeldStreamWithoutALossNote(t *testing.T) {
	s, err := SSE{}.Connect(context.Background(), model.Request{URL: sseServer(t, 1, true)})
	if err != nil {
		t.Fatal(err)
	}
	next(t, s)
	next(t, s)
	s.Close()
	for e := range s.Events() {
		if e.Kind == "close" {
			t.Fatalf("a deliberate close reported: %+v", e)
		}
	}
}

func TestSSEOneShotSendReturnsTheFirstEventAndErrorsAreExplained(t *testing.T) {
	resp, _ := SSE{}.Send(context.Background(), model.Request{URL: sseServer(t, 3, true)})
	if resp.Err != "" || string(resp.Body) != "[1] 1" || resp.Status != 200 {
		t.Fatalf("%+v", resp)
	}
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "no", http.StatusForbidden) }))
	defer bad.Close()
	if _, err := (SSE{}).Connect(context.Background(), model.Request{URL: bad.URL}); err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("err = %v", err)
	}
	if _, err := (SSE{}).Connect(context.Background(), model.Request{URL: "ftp://x"}); err == nil {
		t.Fatal("ftp accepted")
	}
}
