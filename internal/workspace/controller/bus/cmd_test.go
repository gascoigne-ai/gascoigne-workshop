package bus

import (
	"context"
	"slices"
	"testing"
	"time"
)

// waitFor drains until want messages have arrived or a second has passed.
func waitFor(t *testing.T, queue *Bus, want int) []Msg {
	t.Helper()
	var got []Msg
	deadline := time.Now().Add(time.Second)
	for len(got) < want && time.Now().Before(deadline) {
		queue.Drain(func(msg Msg) { got = append(got, msg) })
		time.Sleep(time.Millisecond)
	}
	return got
}

func TestRunDispatchesTheResultOfTheCommand(t *testing.T) {
	queue := New()

	queue.Run(context.Background(), func(context.Context) Msg { return "done" })

	if got := waitFor(t, queue, 1); !slices.Equal(got, []Msg{"done"}) {
		t.Fatalf("got %v, want [done]", got)
	}
}

func TestRunDoesNotBlockTheCallerWhileTheCommandWorks(t *testing.T) {
	queue := New()
	release := make(chan struct{})
	defer close(release)
	returned := make(chan struct{})

	go func() {
		queue.Run(context.Background(), func(context.Context) Msg { <-release; return nil })
		close(returned)
	}()

	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("Run waited for the command")
	}
}

func TestRunGivesTheCommandTheContextItWasStartedWith(t *testing.T) {
	queue := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	queue.Run(ctx, func(ctx context.Context) Msg { return ctx.Err() })

	got := waitFor(t, queue, 1)
	if len(got) != 1 || got[0] != context.Canceled {
		t.Fatalf("got %v, want the cancelled context's error", got)
	}
}

func TestRunDispatchesNothingForACommandWithoutResult(t *testing.T) {
	queue := New()
	finished := make(chan struct{})

	queue.Run(context.Background(), func(context.Context) Msg { defer close(finished); return nil })
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("the command never ran")
	}
	time.Sleep(10 * time.Millisecond) // the dispatch would follow the return at once

	if got := drained(queue); len(got) != 0 {
		t.Fatalf("got %v, want no message", got)
	}
}
