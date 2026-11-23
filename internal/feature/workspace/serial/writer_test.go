package serial

import (
	"errors"
	"testing"
	"time"

	"github.com/gascoigne-ai/gascoigne-workshop/internal/feature/workspace/bus"
)

// newTestWriter returns a writer and the drain the UI loop would run: a
// bus message from a job is a func() the loop calls.
func newTestWriter() (*Writer, func() bool) {
	queue := bus.New()
	drain := func() bool {
		return queue.Drain(func(msg bus.Msg) {
			if run, ok := msg.(func()); ok {
				run()
			}
		})
	}
	return NewWriter(queue, drain), drain
}

func TestWriterRunsAJobOffTheCallerAndReportsOnDrain(t *testing.T) {
	writer, drain := newTestWriter()
	release := make(chan struct{})
	var got error
	reported := false
	writer.Write("file", Job{
		Run:  func() error { <-release; return errors.New("disk full") },
		Done: func(err error) { got, reported = err, true },
	})
	// Run is blocked, yet Write returned: the job is off the caller.
	if reported {
		t.Fatal("Done must not run before the job ends")
	}
	close(release)
	deadline := time.Now().Add(2 * time.Second)
	for !reported && time.Now().Before(deadline) {
		drain()
		time.Sleep(time.Millisecond)
	}
	if !reported || got == nil || got.Error() != "disk full" {
		t.Fatalf("Done got %v (reported %v), want the job's error on the draining goroutine", got, reported)
	}
}

func TestAWriteAskedWhileOneRunsWaitsAndOnlyTheNewestRuns(t *testing.T) {
	writer, _ := newTestWriter()
	release := make(chan struct{})
	var ran, done []string
	job := func(label string, gate chan struct{}) Job {
		return Job{
			Run:  func() error { <-gate; ran = append(ran, label); return nil },
			Done: func(error) { done = append(done, label) },
		}
	}
	writer.Write("file", job("first", release))
	open := make(chan struct{})
	close(open)
	writer.Write("file", job("second", open))
	writer.Write("file", job("third", open))
	close(release)
	writer.Wait()
	if want := []string{"first", "third"}; len(ran) != 2 || ran[0] != want[0] || ran[1] != want[1] {
		t.Fatalf("ran %v, want %v: the replaced job never runs", ran, want)
	}
	if want := []string{"first", "third"}; len(done) != 2 || done[0] != want[0] || done[1] != want[1] {
		t.Fatalf("done %v, want %v: the replaced job's Done never runs", done, want)
	}
}

func TestWaitReturnsOnceTheWritesAreDone(t *testing.T) {
	writer, _ := newTestWriter()
	finished := false
	writer.Write("file", Job{Run: func() error { time.Sleep(20 * time.Millisecond); finished = true; return nil }})
	writer.Wait()
	if !finished {
		t.Fatal("Wait returned before the write ended")
	}
}
