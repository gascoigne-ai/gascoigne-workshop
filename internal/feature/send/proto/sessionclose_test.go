package proto

import (
	"context"
	"encoding/json"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/request"
	"strings"
	"testing"
	"time"
)

// closesWithinASecond runs closeSession and fails the test when it blocks.
func closesWithinASecond(t *testing.T, closeSession func()) {
	t.Helper()
	closed := make(chan struct{})
	go func() {
		closeSession()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Close blocks: the reader waits for room in a full event channel and holds the lock Close needs")
	}
}

// waitsForRoom starts emit calls until one of them blocks on the full
// channel, the way a reader does when nobody drains the events.
func waitsForRoom(emit func()) {
	go func() {
		for range 3 {
			emit()
		}
	}()
	time.Sleep(50 * time.Millisecond)
}

// emit held the session lock while it waited for room in a full event
// channel, and Close needed that lock to signal done, so a flood of
// events plus Close on the UI thread hung the app.
func TestClosingAnSSESessionWithAFullEventChannelDoesNotBlock(t *testing.T) {
	s := &sseSession{eventLog: newTestLog(1), cancel: func() {}}
	waitsForRoom(func() { s.emit(request.Event{}) })
	closesWithinASecond(t, func() { s.Close(context.Background()) })
}

func TestClosingAGRPCSessionWithAFullEventChannelDoesNotBlock(t *testing.T) {
	s := &grpcSession{eventLog: newTestLog(1), cancel: func() {}}
	waitsForRoom(func() { s.emit(request.Event{}) })
	closesWithinASecond(t, func() { s.Close(context.Background()) })
}

// newTestLog is an event log with a queue of size events.
func newTestLog(size int) *eventLog {
	log := newEventLog()
	log.events = make(chan request.Event, size)
	return log
}

type closedTransport struct{}

func (closedTransport) send(context.Context, []byte) error { return nil }
func (closedTransport) setVersion(string)                  {}
func (closedTransport) close(context.Context) error        { return nil }

// finish emitted its "connection lost" note under the once and the lock
// while nobody read the events, so Close waited on that once forever.
func TestClosingAnMCPSessionWhileItsEndNoteWaitsForRoomDoesNotBlock(t *testing.T) {
	s := &mcpSession{eventLog: newTestLog(1), tr: closedTransport{}, waiters: map[int64]chan rpcAnswer{}}
	s.events <- request.Event{} // the channel is full
	go s.finish(context.DeadlineExceeded)
	time.Sleep(50 * time.Millisecond)
	closesWithinASecond(t, func() { s.Close(context.Background()) })
}

type failingTransport struct{ closedTransport }

func (failingTransport) send(context.Context, []byte) error { return context.Canceled }

// call registered its waiter before writing and left it registered when
// the write failed or the context ended, so each failed call leaked one.
func TestAFailedMCPCallLeavesNoWaiterBehind(t *testing.T) {
	s := &mcpSession{eventLog: newTestLog(4), tr: failingTransport{}, waiters: map[int64]chan rpcAnswer{}}
	if _, err := s.call(context.Background(), "ping", nil); err == nil {
		t.Fatal("call on a failing transport succeeded")
	}
	s.mu.Lock()
	left := len(s.waiters)
	s.mu.Unlock()
	if left != 0 {
		t.Errorf("%d waiters left after a failed write", left)
	}

	s.tr = closedTransport{} // writes work, nobody answers
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := s.call(ctx, "ping", nil); err == nil {
		t.Fatal("call without an answer succeeded")
	}
	s.mu.Lock()
	left = len(s.waiters)
	s.mu.Unlock()
	if left != 0 {
		t.Errorf("%d waiters left after a timed-out call", left)
	}
}

// answerServer dropped the error of writing its reply, so a reply that
// never left was invisible and the server waited for it forever.
func TestAFailedReplyToTheServerIsLogged(t *testing.T) {
	s := &mcpSession{eventLog: newTestLog(4), tr: failingTransport{}, waiters: map[int64]chan rpcAnswer{}}
	id := json.RawMessage(`7`)
	s.answerServer(rpcMessage{ID: &id, Method: "ping"})
	close(s.events)
	var kinds []request.EventKind
	for e := range s.events {
		kinds = append(kinds, e.Kind)
		if e.Dir == request.Info && e.Kind == request.KindError && strings.Contains(e.Text, "ping") {
			return
		}
	}
	t.Fatalf("no error event naming the unanswered ping, got %v", kinds)
}
