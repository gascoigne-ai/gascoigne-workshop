package bus

import (
	"slices"
	"sync"
	"testing"
	"time"
)

func drained(queue *Bus) []Msg {
	var got []Msg
	queue.Drain(func(msg Msg) { got = append(got, msg) })
	return got
}

func TestDrainDeliversMessagesInTheOrderTheyWereSent(t *testing.T) {
	queue := New()
	queue.Dispatch("first")
	queue.Dispatch(2)
	queue.Dispatch("third")

	got := drained(queue)

	if want := []Msg{"first", 2, "third"}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestDrainReportsWhetherAnythingArrived(t *testing.T) {
	cases := []struct {
		name     string
		dispatch bool
		want     bool
	}{
		{"empty bus leaves the frame clean", false, false},
		{"a message marks the frame dirty", true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			queue := New()
			if c.dispatch {
				queue.Dispatch("msg")
			}
			if got := queue.Drain(func(Msg) {}); got != c.want {
				t.Fatalf("Drain = %v, want %v", got, c.want)
			}
		})
	}
}

func TestDrainDeliversEachMessageOnce(t *testing.T) {
	queue := New()
	queue.Dispatch("once")
	drained(queue)

	if got := drained(queue); len(got) != 0 {
		t.Fatalf("second drain delivered %v", got)
	}
}

func TestMessageDispatchedWhileDrainingWaitsForTheNextFrame(t *testing.T) {
	queue := New()
	queue.Dispatch("a")
	var seen []Msg
	queue.Drain(func(msg Msg) {
		seen = append(seen, msg)
		if msg == "a" {
			queue.Dispatch("b") // a handler reacting to a message
		}
	})
	if !slices.Equal(seen, []Msg{"a"}) {
		t.Fatalf("this frame delivered %v, want only a", seen)
	}

	if got := drained(queue); !slices.Equal(got, []Msg{"b"}) {
		t.Fatalf("next frame delivered %v, want b", got)
	}
}

func TestDispatchNeverBlocksTheCallerWhateverIsQueued(t *testing.T) {
	queue := New()
	done := make(chan struct{})
	go func() {
		for range 100_000 { // far more than any fixed buffer, and nobody drains
			queue.Dispatch("msg")
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Dispatch blocked with a long queue")
	}
}

func TestDispatchIsSafeFromManyGoroutinesWhileDraining(t *testing.T) {
	const senders, each = 8, 500
	queue := New()
	var wg sync.WaitGroup
	for range senders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range each {
				queue.Dispatch("msg")
			}
		}()
	}
	total := 0
	count := func(Msg) { total++ }
	finished := make(chan struct{})
	go func() { wg.Wait(); close(finished) }()
	for done := false; !done; {
		select {
		case <-finished:
			done = true
		default:
		}
		queue.Drain(count)
	}
	queue.Drain(count)

	if total != senders*each {
		t.Fatalf("delivered %d messages, want %d", total, senders*each)
	}
}
