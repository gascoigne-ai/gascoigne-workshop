package ui

import (
	"context"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/workspace/controller/serial"
)

// writeWait bounds how long quitting waits for background writes.
const writeWait = 3 * time.Second

// write runs job off the UI goroutine: one write per name at a time, the
// newest one last (serial.Queue). The job's Run must work on data it owns
// (a clone), since the UI keeps changing the originals.
func (a *App) write(name string, job serial.Job) {
	if a.writes.Submit(name, job) {
		a.startWrite(name, job)
	}
}

func (a *App) startWrite(name string, job serial.Job) {
	a.bus.Run(context.Background(), func(context.Context) bus.Msg {
		err := job.Run()
		return func() { // on the UI goroutine
			if job.Done != nil {
				job.Done(err)
			}
			if next, ok := a.writes.Finish(name); ok {
				a.startWrite(name, next)
			}
		}
	})
}

// waitWrites blocks until the background writes are done (or writeWait
// passes), for the paths that must not lose data: quitting and opening
// another collection. Their outcomes are handled as the frame would.
func (a *App) waitWrites() {
	deadline := time.Now().Add(writeWait)
	for a.writes.Busy() && time.Now().Before(deadline) {
		if !a.drain() {
			time.Sleep(time.Millisecond)
		}
	}
}
