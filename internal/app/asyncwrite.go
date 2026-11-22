package app

import (
	"context"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/serial"
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

// writeFile writes data to a file the user named through the Folders
// port, off the UI goroutine; done gets the outcome on it. Writes are
// queued by path, so two saves to one file never interleave.
func (a *App) writeFile(path string, data []byte, done func(error)) {
	files := a.folders
	a.write("file "+path, serial.Job{
		Run:  func() error { return files.Write(context.Background(), path, data) },
		Done: done,
	})
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

// WaitWrites blocks until the background writes are done (or writeWait
// passes), for the paths that must not lose data: quitting and opening
// another collection. Their outcomes are handled as the frame would. It is
// exported for apptest, whose tests wait for saves before they read files.
func (a *App) WaitWrites() {
	deadline := time.Now().Add(writeWait)
	for a.writes.Busy() && time.Now().Before(deadline) {
		if !a.drain() {
			time.Sleep(time.Millisecond)
		}
	}
}
