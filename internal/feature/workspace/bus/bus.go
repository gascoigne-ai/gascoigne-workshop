// Package bus carries messages from anywhere, goroutines included, to the
// UI goroutine, which drains them once per frame.
package bus

import "sync"

// Msg is anything sent over the bus; each domain defines its own types.
type Msg any

// Bus is the queue of messages waiting for the next frame. The zero value
// is ready to use; a Bus must not be copied.
type Bus struct {
	mu      sync.Mutex
	waiting []Msg
}

// New makes an empty bus.
func New() *Bus { return &Bus{} }

// Dispatch queues msg for the next Drain. It is safe from any goroutine and
// never blocks beyond the short lock: the queue grows instead of making the
// sender wait, so the UI goroutine may dispatch too.
func (queue *Bus) Dispatch(msg Msg) {
	queue.mu.Lock()
	queue.waiting = append(queue.waiting, msg)
	queue.mu.Unlock()
}

// Drain hands every queued message to handle, in the order they were sent,
// and reports whether there were any, which is what marks the frame dirty.
// A message that handle dispatches waits for the next Drain, so a handler
// that answers its own message cannot keep one frame busy forever.
func (queue *Bus) Drain(handle func(Msg)) bool {
	queue.mu.Lock()
	batch := queue.waiting
	queue.waiting = nil
	queue.mu.Unlock()

	for _, msg := range batch {
		handle(msg)
	}
	return len(batch) > 0
}
