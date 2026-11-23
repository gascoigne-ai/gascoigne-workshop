package protocol

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// slotSet is a counting semaphore whose zero value is usable: its size is
// given on the first take.
type slotSet struct {
	once sync.Once
	free chan struct{}
}

// tryTake claims a slot without waiting and reports whether there was one.
func (slots *slotSet) tryTake(size int) bool {
	slots.once.Do(func() { slots.free = make(chan struct{}, size) })
	select {
	case slots.free <- struct{}{}:
		return true
	default:
		return false
	}
}

// release gives back a slot taken by tryTake.
func (slots *slotSet) release() { <-slots.free }

// used is the number of slots taken.
func (slots *slotSet) used() int { return len(slots.free) }

// spawn runs task on its own goroutine when fewer than maxServerWork tasks
// run, and reports false, without running it, when the session is busy.
func (s *mcpSession) spawn(task func()) bool {
	if !s.work.tryTake(maxServerWork) {
		return false
	}
	go func() {
		defer s.work.release()
		task()
	}()
	return true
}

// busyNoteInterval is the least time between two notes about refused work.
const busyNoteInterval = time.Second

// serve answers a request of the server on a worker. When none is free the
// request is refused with a busy error, so the server is not left waiting.
func (s *mcpSession) serve(message rpcMessage) {
	if s.spawn(func() { s.answerServer(message) }) {
		return
	}
	s.refuse(message)
}

// refuse queues the busy reply for the goroutine that writes them: the
// reader, which calls this, must not wait for a slow send. A full queue
// drops the reply; the server's own timeout ends that request.
func (s *mcpSession) refuse(message rpcMessage) {
	s.refusedOne.Do(func() {
		s.refused = make(chan rpcMessage, maxPending)
		go s.answerRefused()
	})
	select {
	case s.refused <- message:
	default:
	}
	s.noteBusy("server requests")
}

// answerRefused writes the busy replies, one at a time, until the session ends.
func (s *mcpSession) answerRefused() {
	for {
		select {
		case message := <-s.refused:
			s.reply(message.Method, errorReply(*message.ID, rpcBusy, "client busy"))
		case <-s.done:
			return
		}
	}
}

// noteBusy tells the user that work was refused, once per busyNoteInterval
// with the count since the last note, however long the flood is.
func (s *mcpSession) noteBusy(what string) {
	s.mu.Lock()
	s.busyCount++
	now := s.now()
	if now.Sub(s.lastBusy) < busyNoteInterval {
		s.mu.Unlock()
		return
	}
	count := s.busyCount
	s.busyCount, s.lastBusy = 0, now
	s.mu.Unlock()
	s.note(fmt.Errorf("client busy: refused %d %s", count, what))
}

// refreshState is the refresh of one list: running while a request for it
// is being sent, and again when a notification came in meanwhile.
type refreshState struct{ running, again bool }

// refreshList asks the server for a list again. At most one refresh of a
// list runs; notifications that arrive meanwhile make it run once more.
// With no free worker the refresh is dropped with a note.
func (s *mcpSession) refreshList(method string) {
	s.mu.Lock()
	if s.refreshing == nil {
		s.refreshing = map[string]refreshState{}
	}
	state := s.refreshing[method]
	if state.running {
		state.again = true
		s.refreshing[method] = state
		s.mu.Unlock()
		return
	}
	s.refreshing[method] = refreshState{running: true}
	s.mu.Unlock()
	if s.spawn(func() { s.refreshUntilQuiet(method) }) {
		return
	}
	s.mu.Lock()
	delete(s.refreshing, method)
	s.mu.Unlock()
	s.noteBusy("list refreshes")
}

func (s *mcpSession) refreshUntilQuiet(method string) {
	for {
		if _, err := s.request(context.Background(), method, nil); err != nil {
			s.note(err)
		}
		s.mu.Lock()
		if !s.refreshing[method].again {
			delete(s.refreshing, method)
			s.mu.Unlock()
			return
		}
		s.refreshing[method] = refreshState{running: true}
		s.mu.Unlock()
	}
}
