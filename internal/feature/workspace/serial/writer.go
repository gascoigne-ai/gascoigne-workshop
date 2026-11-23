package serial

import (
	"context"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
)

// writeWait bounds how long Wait blocks for background writes.
const writeWait = 3 * time.Second

// Writer runs jobs off the UI goroutine under the Queue's rule: one write
// per name at a time, the newest waiting one last. A job's Run must work on
// data it owns (a clone), since the UI keeps changing the originals.
//
// Its methods are for the UI goroutine only: Done and the start of the next
// job run there, when the bus is drained. Why the Queue stays a separate
// type: it is the rule and needs no goroutine to be tested; the Writer is
// the thin glue to the bus.
type Writer struct {
	queue Queue
	bus   *bus.Bus
	drain func() bool // delivers the bus messages on the UI goroutine
}

// NewWriter builds a Writer that reports on queue and lets Wait deliver
// those reports through drain, which says whether it handled any message.
func NewWriter(queue *bus.Bus, drain func() bool) *Writer {
	return &Writer{bus: queue, drain: drain}
}

// Write runs job off the UI goroutine, or after the write of name that is
// already running.
func (writer *Writer) Write(name string, job Job) {
	if writer.queue.Submit(name, job) {
		writer.start(name, job)
	}
}

func (writer *Writer) start(name string, job Job) {
	writer.bus.Run(context.Background(), func(context.Context) bus.Msg {
		err := job.Run()
		return func() { // on the UI goroutine
			if job.Done != nil {
				job.Done(err)
			}
			if next, ok := writer.queue.Finish(name); ok {
				writer.start(name, next)
			}
		}
	})
}

// Wait blocks until the background writes are done (or writeWait passes),
// for the paths that must not lose data: quitting and opening another
// collection. Their outcomes are handled as the frame would. The give-up
// bound is not covered by a unit test: it would take three seconds.
func (writer *Writer) Wait() {
	deadline := time.Now().Add(writeWait)
	for writer.queue.Busy() && time.Now().Before(deadline) {
		if !writer.drain() {
			time.Sleep(time.Millisecond)
		}
	}
}
