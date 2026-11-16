package proto

import (
	"context"
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
	s := &sseSession{events: make(chan Event, 1), done: make(chan struct{}), cancel: func() {}}
	waitsForRoom(func() { s.emit(Event{}) })
	closesWithinASecond(t, func() { s.Close() })
}

func TestClosingAGRPCSessionWithAFullEventChannelDoesNotBlock(t *testing.T) {
	s := &grpcSession{events: make(chan Event, 1), done: make(chan struct{}), cancel: func() {}}
	waitsForRoom(func() { s.emit(Event{}) })
	closesWithinASecond(t, func() { s.Close() })
}

type closedTransport struct{}

func (closedTransport) send(context.Context, []byte) error { return nil }
func (closedTransport) setVersion(string)                  {}
func (closedTransport) close() error                       { return nil }

// finish emitted its "connection lost" note under the once and the lock
// while nobody read the events, so Close waited on that once forever.
func TestClosingAnMCPSessionWhileItsEndNoteWaitsForRoomDoesNotBlock(t *testing.T) {
	s := &mcpSession{tr: closedTransport{}, events: make(chan Event, 1), done: make(chan struct{}), now: time.Now, waiters: map[int64]chan rpcAnswer{}}
	s.events <- Event{} // the channel is full
	go s.finish(context.DeadlineExceeded)
	time.Sleep(50 * time.Millisecond)
	closesWithinASecond(t, func() { s.Close() })
}
