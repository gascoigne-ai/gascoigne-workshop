// Package serial decides when a write to a file may start: one write per
// name at a time, and a write asked for while one runs waits and replaces
// any write already waiting, so the newest data is written last and an
// older write never lands after a newer one. It does no I/O itself.
package serial

// Job is a write and what to do with its outcome.
type Job struct {
	Run  func() error // writes; runs off the UI goroutine
	Done func(error)  // runs on the UI goroutine with the outcome
}

// Queue tracks the writes in flight and waiting by name.
type Queue struct {
	busy    map[string]bool
	waiting map[string]Job
}

// Submit reports whether job may start now. When a write of name is
// running it is kept, replacing one that was already waiting, and Submit
// reports false.
//
// A job that is replaced while waiting is dropped, its Done never runs:
// the newer one writes the newer data and reports for both.
func (q *Queue) Submit(name string, job Job) (start bool) {
	if q.busy == nil {
		q.busy, q.waiting = map[string]bool{}, map[string]Job{}
	}
	if q.busy[name] {
		q.waiting[name] = job
		return false
	}
	q.busy[name] = true
	return true
}

// Finish records that the write of name ended and returns the one that was
// waiting for it, if any, which then counts as running.
func (q *Queue) Finish(name string) (next Job, ok bool) {
	next, ok = q.waiting[name]
	delete(q.waiting, name)
	if !ok {
		delete(q.busy, name)
	}
	return next, ok
}

// Busy reports whether any write is running.
func (q *Queue) Busy() bool { return len(q.busy) > 0 }
